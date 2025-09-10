package grpc

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/golang/protobuf/proto"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/purposeinplay/go-commons/errors"
	commonserr "github.com/purposeinplay/go-commons/errors/proto/commons/error/v1"
	"github.com/purposeinplay/go-commons/grpc/grpcutils"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/filters"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorReporter is the interface that wraps the ReportError method.
type ErrorReporter interface {
	ReportError(ctx context.Context, err error)
}

// NewSimpleServer creates a new production server instance without starting it.
func NewSimpleServer(
	logger *slog.Logger,
	errorReporter ErrorReporter,
	registerFunc func(server *grpc.Server),
) (*Server, error) {
	return NewServer(
		WithOTEL(
			otelgrpc.WithFilter(filters.None(
				filters.MethodName("Check"),
				filters.MethodName("Healthcheck"),
			))),
		WithUnaryServerInterceptor(grpcutils.PassRequestIDUnaryInterceptor()),
		WithUnaryServerInterceptor(errorHandlerUnaryServerInterceptor(errorReporter)),
		WithUnaryServerInterceptor(grpcrecovery.UnaryServerInterceptor(
			grpcrecovery.WithRecoveryHandler(func(
				p interface{},
			) error {
				return &errors.Error{
					Type:    errors.ErrorTypePanic,
					Message: fmt.Sprintf("%+v", p),
				}
			}),
		)),
		WithUnaryServerInterceptorLogger(logger),
		WithRegisterServerFunc(registerFunc),
		WithNoGateway(),
	)
}

// errorHandlerUnaryServerInterceptor returns a grpc.UnaryServerInterceptor that handles errors
func errorHandlerUnaryServerInterceptor(
	errorReporter ErrorReporter,
) grpc.UnaryServerInterceptor {
	handleError := func(
		ctx context.Context,
		info *grpc.UnaryServerInfo,
		err error,
	) *status.Status {
		if errors.Is(err, context.Canceled) {
			return nil
		}

		if sts := isGRPCStatus(err); sts != nil {
			return sts
		}

		var grpcStatus *status.Status

		var appError *errors.Error

		if errors.As(err, &appError) {
			if appError.Type == errors.ErrorTypePanic || appError.Type == errors.ErrorTypeInternalError {
				errorReporter.ReportError(ctx, appError)
			}

			code := codes.Unknown

			var errorTypeToStatusCode = map[errors.ErrorType]codes.Code{
				errors.ErrorTypeInvalid:              codes.InvalidArgument,
				errors.ErrorTypeNotFound:             codes.NotFound,
				errors.ErrorTypeUnprocessableContent: codes.Internal,
				errors.ErrorTypeUnauthorized:         codes.PermissionDenied,
				errors.ErrorTypeUnauthenticated:      codes.Unauthenticated,
				errors.ErrorTypeInternalError:        codes.Internal,
				errors.ErrorTypePanic:                codes.Internal,
			}

			c, ok := errorTypeToStatusCode[appError.Type]
			if ok {
				code = c
			}

			grpcStatus = status.New(code, appError.Type.String())

			grpcStatus, _ = grpcStatus.WithDetails(
				proto.MessageV1(
					&commonserr.ErrorResponse{
						ErrorCode: appError.Code.String(),
						Message:   appError.Message,
					}))
		} else {
			grpcStatus = status.New(codes.Internal, err.Error())

			errorReporter.ReportError(ctx, err)
		}

		return grpcStatus
	}

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			return nil, handleError(ctx, info, err).Err()
		}

		return resp, nil
	}
}
