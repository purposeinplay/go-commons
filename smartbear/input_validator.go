package smartbear

import (
	"context"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"k8s.io/utils/ptr"
)

var (
	// ValidationFailed indicates error related to invalid user actions or operations.
	ValidationFailed = "VALIDATION_FAILED"
)

// NewValidator creates a new OpenAPI 3 validator.
func NewValidator(
	swagger *openapi3.T,
	errorHandler ErrorHandler,
) *openapi3filter.Validator {
	swagger.Servers = nil

	r, err := gorillamux.NewRouter(swagger)
	if err != nil {
		panic(err)
	}

	return openapi3filter.NewValidator(
		r,
		openapi3filter.ValidationOptions(openapi3filter.Options{
			MultiError: true,
			AuthenticationFunc: func(context.Context, *openapi3filter.AuthenticationInput) error {
				return nil
			},
		}),
		openapi3filter.OnErr(func(
			ctx context.Context,
			w http.ResponseWriter,
			_ int,
			_ openapi3filter.ErrCode,
			err error,
		) {
			// nolint: contextcheck
			errorHandler.WriteErrorResponse(ctx, w, err)
		}),
	)
}

func newValidationErrorProblemDetails(errs *Errors) ProblemDetails {
	return ProblemDetails{
		Title:  http.StatusText(http.StatusBadRequest),
		Status: ptr.To(int32(http.StatusBadRequest)),
		Code:   &ValidationFailed,
		Detail: ptr.To("Request validation failed"),
		Errors: errs,
	}
}
