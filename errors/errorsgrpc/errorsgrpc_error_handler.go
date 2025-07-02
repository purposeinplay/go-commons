package errorsgrpc

import (
	"context"
	"fmt"

	// nolint: staticcheck
	"log/slog"

	"github.com/golang/protobuf/proto"
	"github.com/purposeinplay/go-commons/errors"
	commonserr "github.com/purposeinplay/go-commons/errors/proto/commons/error/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ReportErrorer is the interface that wraps the ReportError method.
type ReportErrorer interface {
	ReportError(context.Context, error) error
}

// PanicErrorHandler handles Panic and Errors in GRPC context.
type PanicErrorHandler struct {
	reportErrorer ReportErrorer

	logger *slog.Logger
}

// MustNewPanicErrorHandler creates an initialized PanicErrorHandler.
// Panics if invalid data is passed.
func MustNewPanicErrorHandler(
	reportErrorer ReportErrorer,
	logger *slog.Logger,
) PanicErrorHandler {
	if reportErrorer == nil {
		panic("nil error reporter")
	}

	if logger == nil {
		panic("nil logger")
	}

	return PanicErrorHandler{
		reportErrorer: reportErrorer,
		logger:        logger,
	}
}

// LogError logs an error to STDERR.
func (h PanicErrorHandler) LogError(err error) {
	h.logger.Error("grpc handler encountered an error", "error", err)
}

// LogPanic logs a panic to STDERR.
func (h PanicErrorHandler) LogPanic(p any) {
	h.logger.Error(
		"grpc handler encountered a panic",
		"cause", p,
	)
}

// IsApplicationError checks whether the error is an Application Error.
func (PanicErrorHandler) IsApplicationError(err error) bool {
	var applicationError *errors.Error
	return errors.As(err, &applicationError)
}

var errorTypeToStatusCode = map[errors.ErrorType]codes.Code{
	errors.ErrorTypeInvalid:              codes.InvalidArgument,
	errors.ErrorTypeNotFound:             codes.NotFound,
	errors.ErrorTypeUnprocessableContent: codes.Internal,
	errors.ErrorTypeUnauthorized:         codes.PermissionDenied,
	errors.ErrorTypeUnauthenticated:      codes.Unauthenticated,
	errors.ErrorTypeInternalError:        codes.Internal,
}

// ErrNotApplicationError is returned whenever an error that is not an *errors.Error is given
// to the PanicErrorHandler.ErrorToGRPCStatus method.
var ErrNotApplicationError = errors.New("given error is not an application error")

// ErrorToGRPCStatus converts an error to a grpc status.
func (PanicErrorHandler) ErrorToGRPCStatus(
	err error,
) (*status.Status, error) {
	if s, ok := status.FromError(err); ok {
		return s, nil
	}

	var applicationError *errors.Error

	if !errors.As(err, &applicationError) {
		return nil, ErrNotApplicationError
	}

	code, ok := errorTypeToStatusCode[applicationError.Type]
	if !ok {
		code = codes.Unknown
	}

	grpcStatus := status.New(code, applicationError.Type.String())

	grpcStatusWithDetails, attachDetailsErr := grpcStatus.
		WithDetails(proto.MessageV1(errorToErrorResponse(applicationError)))
	if attachDetailsErr != nil {
		return nil, fmt.Errorf("attach details: %w", err)
	}

	return grpcStatusWithDetails, nil
}

// ReportPanic reports a panic to an external service.
func (h PanicErrorHandler) ReportPanic(
	ctx context.Context,
	p any,
) error {
	return h.reportErrorer.ReportError(
		ctx,
		&errors.Error{
			Type:    errors.ErrorTypePanic,
			Message: fmt.Sprintf("%+v", p),
		},
	)
}

// ReportError reports an error to an external service.
func (h PanicErrorHandler) ReportError(
	ctx context.Context,
	err error,
) error {
	return h.reportErrorer.ReportError(
		ctx,
		err,
	)
}

func errorToErrorResponse(err *errors.Error) *commonserr.ErrorResponse {
	return &commonserr.ErrorResponse{
		//nolint:gosec // disable G115
		ErrorCode: err.Code.String(),
		Message:   err.Message,
	}
}
