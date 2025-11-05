package smartbear

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-playground/validator/v10"
	"github.com/purposeinplay/go-commons/errors"
	"k8s.io/utils/ptr"
)

var (
	// InvalidInput indicates invalid input was provided.
	InvalidInput string = "INVALID_INPUT"
)

type ErrorReporter interface {
	ReportError(ctx context.Context, err error)
}

// ErrorHandler is a middleware that processes errors and writes appropriate
// error responses to the client.
type ErrorHandler struct {
	ErrorReporter ErrorReporter
	Logger        *slog.Logger
}

// WriteErrorResponse writes an error response to the client.
func (e ErrorHandler) WriteErrorResponse(
	ctx context.Context,
	w http.ResponseWriter,
	targetErr error,
) {
	var (
		appError         *errors.Error                // Type for application-specific errors
		validationErrors validator.ValidationErrors   // Type for validation errors
		requestError     *openapi3filter.RequestError // Type for openapi3filter request errors

		response = ProblemDetails{ // Default to internal server error
			Title:  "Internal Server Error",
			Status: ptr.To(int32(http.StatusInternalServerError)),
		}
	)

	// Second defer: ensures response is always written and errors are reported
	// This runs after error type checking and response preparation
	defer func() {
		// Report internal server errors to Sentry
		if *response.Status == int32(http.StatusInternalServerError) {
			e.Logger.Error(
				"internal server error",
				slog.String("error", targetErr.Error()),
			)
			go e.ErrorReporter.ReportError(
				ctx,
				targetErr,
			)
		}

		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(int(*response.Status))

		if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
			e.Logger.Error(
				"error encoding response",
				slog.String("encode_error", encodeErr.Error()))

			go e.ErrorReporter.ReportError(
				ctx,
				targetErr,
			)
		}
	}()

	// Process different error types in order of specificity
	switch {
	case errors.As(targetErr, &requestError):
		r := e.handleOpenAPI3Err(requestError)
		if r == nil {
			return
		}

		response = *r

	case errors.As(targetErr, &validationErrors):
		errs := make(Errors, len(validationErrors))

		for i, validationError := range validationErrors {
			errs[i] = ErrorDetail{
				Pointer: ptr.To(validationError.Field()),
				Detail:  "Validation failed",
			}
		}

		response = newValidationErrorProblemDetails(&errs)

	case errors.As(targetErr, &appError):
		// Handle application-specific errors
		// Uses the error's status code and converts error errs
		errs := make([]ErrorDetail, len(appError.ErrorDetails))

		for i, detail := range appError.ErrorDetails {
			errs[i] = ErrorDetail{
				Code:   detail.Code.StringPtr(),
				Detail: detail.Message,
			}
		}

		response = ProblemDetails{
			Title:  http.StatusText(appError.Type.HTTPStatus()),
			Status: appError.Type.HTTPStatusInt32Ptr(),
			Code:   appError.Code.StringPtr(),
			Detail: ptr.To(appError.Message),
			Errors: &errs,
		}

	default:
		return
	}
}

func (ErrorHandler) handleOpenAPI3Err(
	requestErr *openapi3filter.RequestError,
) *ProblemDetails {
	switch {
	case requestErr.Parameter != nil:
		pb := newValidationErrorProblemDetails(&Errors{
			{
				Parameter: &requestErr.Parameter.Name,
				Code:      &InvalidInput,
				Detail:    requestErr.Reason,
			},
		})

		return &pb
	default:
		var me openapi3.MultiError

		if !errors.As(requestErr.Err, &me) {
			return nil
		}

		errs := make(Errors, len(me))

		for i, e := range me {
			var schemaErr *openapi3.SchemaError

			if !errors.As(e, &schemaErr) {
				continue
			}

			errs[i] = ErrorDetail{
				Code:    &InvalidInput,
				Detail:  schemaErr.Reason,
				Pointer: ptr.To(strings.Join(schemaErr.JSONPointer(), "/")),
			}
		}

		pb := newValidationErrorProblemDetails(&errs)

		return &pb
	}
}
