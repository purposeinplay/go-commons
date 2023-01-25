package grpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"strconv"
	"time"

	"contrib.go.opencensus.io/exporter/stackdriver"
	"github.com/google/uuid"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/purposeinplay/go-commons/grpc/grpcutils"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ server = (*grpcServer)(nil)

type grpcServer struct {
	internalGRPCServer *grpc.Server
}

func (s *grpcServer) Serve(listener net.Listener) error {
	return s.internalGRPCServer.Serve(listener)
}

func (s *grpcServer) Close() error {
	s.internalGRPCServer.GracefulStop()

	return nil
}

// nolint: gocyclo, revive // cyclomatic complexity is 9. FIXME
// revive complains about tracing being a control flag.
func newGRPCServerWithListener(
	listener net.Listener,
	address string,
	tracing bool,
	defaultGRPCServerOptions []grpc.ServerOption,
	unaryServerInterceptors []grpc.UnaryServerInterceptor,
	registerServer registerServerFunc,
	logging *logging,
	errorHandler ErrorHandler,
	panicHandler PanicHandler,
	monitorOperationer MonitorOperationer,
) (
	*serverWithListener,
	error,
) {
	grpcListener, err := newGRPCListener(listener, address)
	if err != nil {
		return nil, fmt.Errorf("new grpc listener: %w", err)
	}

	grpcServerOptions := defaultGRPCServerOptions

	if tracing {
		grpcServerOptions, err = setGRPCTracing(grpcServerOptions)
		if err != nil {
			return nil, fmt.Errorf("set grpc tracing tracing: %w", err)
		}
	}

	if !isErrorHandlerNil(errorHandler) {
		// nolint: revive // complains that this lines modifies
		// an input parameter.
		unaryServerInterceptors = prependErrorHandler(
			unaryServerInterceptors,
			errorHandler,
		)
	}

	if !isPanicHandlerNil(panicHandler) {
		// nolint: revive // complains that this lines modifies
		// an input parameter.
		unaryServerInterceptors = prependPanicHandler(
			unaryServerInterceptors,
			panicHandler,
		)
	}

	if logging != nil {
		// nolint: revive // complains that this lines modifies
		// an input parameter.
		unaryServerInterceptors = prependDebugInterceptor(
			unaryServerInterceptors,
			logging,
		)
	}

	if !isMonitorOperationerNil(monitorOperationer) {
		// nolint: revive // complains that this lines modifies
		// an input parameter.
		unaryServerInterceptors = append(
			unaryServerInterceptors,
			newMonitorOperationUnaryInterceptor(monitorOperationer),
		)
	}

	if len(unaryServerInterceptors) > 0 {
		grpcServerOptions = append(grpcServerOptions,
			grpc_middleware.WithUnaryServerChain(
				unaryServerInterceptors...,
			))
	}

	internalGRPCServer := grpc.NewServer(grpcServerOptions...)

	if registerServer != nil {
		registerServer(internalGRPCServer)
	}

	return &serverWithListener{
		server: &grpcServer{
			internalGRPCServer: internalGRPCServer,
		},
		listener: grpcListener,
	}, nil
}

func setGRPCTracing(
	serverOptions []grpc.ServerOption,
) ([]grpc.ServerOption, error) {
	exporter, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID: os.Getenv("GOOGLE_CLOUD_PROJECT"),
	})
	if err != nil {
		return nil, fmt.Errorf("new exporter: %w", err)
	}

	trace.RegisterExporter(exporter)
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	return append(
		serverOptions,
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),
	), nil
}

func newGRPCListener(
	defaultListener net.Listener,
	addr string,
) (net.Listener, error) {
	if defaultListener != nil {
		return defaultListener, nil
	}

	hostString, portString, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	port, err := strconv.Atoi(portString)
	if err != nil {
		return nil, fmt.Errorf("parse port: %w", err)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", hostString, port-1))
	if err != nil {
		return nil, fmt.Errorf("new net listener: %w", err)
	}

	return listener, nil
}

func prependDebugInterceptor(
	interceptors []grpc.UnaryServerInterceptor,
	logging *logging,
) []grpc.UnaryServerInterceptor {
	logging.ignoredMethods = append(
		logging.ignoredMethods,
		"Check",
		"Watch",
	)

	return prependServerOption(
		func(
			ctx context.Context,
			req interface{},
			info *grpc.UnaryServerInfo,
			handler grpc.UnaryHandler,
		) (resp interface{}, err error) {
			start := time.Now()

			method := path.Base(info.FullMethod)

			for _, m := range logging.ignoredMethods {
				if method == m {
					return handler(ctx, req)
				}
			}

			requestID, err := grpcutils.GetRequestIDFromCtx(ctx)
			if err != nil {
				requestID = uuid.Nil.String()
			}

			logging.logger.Debug(
				"request started",
				zap.String("trace_id", requestID),
				zap.String("method", method),
			)

			request, err := handler(ctx, req)

			code := status.Code(err)

			if err != nil {
				logging.logger.Debug(
					"request completed with error",
					zap.String("trace_id", requestID),
					zap.String("method", method),
					zap.Any("request", req),
					zap.Error(err),
					zap.String("code", code.String()),
					zap.Duration("duration", time.Since(start)),
				)

				return request, err
			}

			logging.logger.Debug(
				"request completed successfully",
				zap.String("trace_id", requestID),
				zap.String("method", method),
				zap.String("code", code.String()),
				zap.Duration("duration", time.Since(start)),
			)

			return request, err
		},
		interceptors,
	)
}

// PanicHandler defines methods for handling a panic.
type PanicHandler interface {
	ReportPanic(context.Context, interface{}) error
	LogPanic(interface{})
	LogError(error)
}

func newRecoveryFunc(
	panicHandler PanicHandler,
) grpc_recovery.RecoveryHandlerFunc {
	return func(p interface{}) error {
		ctx, cancelCtx := context.WithTimeout(
			context.Background(),
			time.Second,
		)
		defer cancelCtx()

		panicHandler.LogPanic(p)

		reportPanicErr := panicHandler.ReportPanic(ctx, p)
		if reportPanicErr != nil {
			panicHandler.LogError(fmt.Errorf(
				"error while reporting panic %q: %w",
				p,
				reportPanicErr,
			))
		}

		return status.Error(codes.Internal, "internal error.")
	}
}

func prependPanicHandler(
	interceptors []grpc.UnaryServerInterceptor,
	panicHandler PanicHandler,
) []grpc.UnaryServerInterceptor {
	return prependServerOption(
		grpc_recovery.UnaryServerInterceptor(
			grpc_recovery.WithRecoveryHandler(newRecoveryFunc(panicHandler)),
		),
		interceptors,
	)
}

// ErrorHandler defines methods for handling an error.
type ErrorHandler interface {
	LogError(error)
	IsApplicationError(error) bool
	ReportError(context.Context, error) error
	ErrorToGRPCStatus(error) (*status.Status, error)
}

// HandleError proposes a way of handling GRPC errors.
// It logs and reports the error to an external service, everything
// under a one-second timeout to avoid increasing the response time.
// nolint: gocognit // high cognitive complexity, fix later.
func HandleError(
	targetErr error,
	errorHandler ErrorHandler,
) error {
	const timeout = time.Second

	var (
		grpcStatus     *status.Status
		ctx, cancelCtx = context.WithTimeout(context.Background(), timeout)
	)

	defer cancelCtx()

	// In order to preserve space it would be better
	// to only log internal errors.
	errorHandler.LogError(targetErr)

	if errors.Is(targetErr, context.Canceled) {
		return nil
	}

	// Check if the error is an application error or an
	// internal error
	switch {
	// If the error is an application error prepare the grpc
	// response.
	case errorHandler.IsApplicationError(targetErr):
		// Convert the application error type to a GRPC status.
		sts, toGrpcStatusErr := errorHandler.ErrorToGRPCStatus(targetErr)
		// on success break
		if toGrpcStatusErr == nil {
			grpcStatus = sts

			break
		}

		// handle err
		grpcStatus = status.New(codes.Internal, "internal error.")

		toGrpcStatusErr = fmt.Errorf(
			"error to grpc status: %w",
			toGrpcStatusErr,
		)

		errorHandler.LogError(toGrpcStatusErr)

		go func() {
			reportErrErr := errorHandler.ReportError(ctx, toGrpcStatusErr)
			if reportErrErr == nil {
				return
			}

			errorHandler.LogError(fmt.Errorf(
				"error while reporting this error %q: %w",
				toGrpcStatusErr.Error(),
				reportErrErr,
			))
		}()

	case errors.Is(targetErr, context.Canceled):
		grpcStatus = status.New(codes.Internal, "context cancelled.")

	// If the error is an internal error, report it to an external
	// service.
	default:
		if s := isGRPCStatus(targetErr); s != nil {
			grpcStatus = s
			break
		}

		go func() {
			// Report the error to an external service
			reportErr := errorHandler.ReportError(ctx, targetErr)
			if reportErr != nil {
				// Log the error received from the external service
				// and continue execution.
				errorHandler.LogError(fmt.Errorf(
					"error while reporting this error %q: %w",
					targetErr.Error(),
					reportErr,
				))
			}
		}()

		// Create a GRPC status that doesn't leak any information
		// about the internal error.
		grpcStatus = status.New(codes.Internal, "internal error.")
	}

	// Return the grpc Status as an immutable error.
	return grpcStatus.Err()
}

func isGRPCStatus(err error) *status.Status {
	var statusCandidate error

	errCpy := err

	for {
		statusCandidate = errCpy

		errCpy = errors.Unwrap(statusCandidate)
		if errCpy == nil {
			break
		}
	}

	s, _ := status.FromError(statusCandidate)

	return s
}

func prependErrorHandler(
	interceptors []grpc.UnaryServerInterceptor,
	errorHandler ErrorHandler,
) []grpc.UnaryServerInterceptor {
	return prependServerOption(
		func(
			ctx context.Context,
			req interface{},
			info *grpc.UnaryServerInfo,
			handler grpc.UnaryHandler,
		) (interface{}, error) {
			resp, err := handler(ctx, req)
			if err != nil {
				// nolint: contextcheck // do not pass the request context
				// here as we do not want to pass the request context and have
				// the handler cancelled in case the client cancels
				// the request.
				return nil, HandleError(fmt.Errorf(
					"%q: %w",
					path.Base(info.FullMethod),
					err,
				), errorHandler)
			}

			return resp, nil
		},
		interceptors,
	)
}

// MonitorOperationer defines.
type MonitorOperationer interface {
	MonitorOperation(
		ctx context.Context,
		name string,
		traceID [16]byte,
		operationFunc func(context.Context),
	)
}

func newMonitorOperationUnaryInterceptor(
	monitorOperationer MonitorOperationer,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if info.FullMethod == "/grpc.health.v1.Health/Check" {
			return handler(ctx, req)
		}

		requestID, err := grpcutils.GetRequestIDFromCtx(ctx)
		if err != nil {
			requestID = uuid.Nil.String()
		}

		traceID, _ := uuid.Parse(requestID)

		var (
			resp       interface{}
			handlerErr error
		)

		monitorOperationer.MonitorOperation(
			ctx,
			info.FullMethod,
			traceID,
			func(ctx context.Context) {
				resp, handlerErr = handler(ctx, req)
			},
		)

		if handlerErr != nil {
			return nil, handlerErr
		}

		return resp, nil
	}
}
