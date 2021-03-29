package httperr

import (
	"fmt"
	"github.com/go-chi/chi/middleware"
	"github.com/purposeinplay/go-commons/http/render"
	"github.com/purposeinplay/go-commons/logs"
	"go.uber.org/zap"
	"net/http"
)

var oauthErrorMap = map[int]string{
	http.StatusBadRequest:          "invalid_request",
	http.StatusUnauthorized:        "unauthorized_client",
	http.StatusForbidden:           "access_denied",
	http.StatusInternalServerError: "server_error",
	http.StatusServiceUnavailable:  "temporarily_unavailable",
}

// OAuthError is the JSON handler for OAuth2 error responses
type OAuthError struct {
	Err             string `json:"error"`
	Description     string `json:"error_description,omitempty"`
	InternalError   error  `json:"-"`
	InternalMessage string `json:"-"`
}

func (e *OAuthError) Error() string {
	if e.InternalMessage != "" {
		return e.InternalMessage
	}
	return fmt.Sprintf("%s: %s", e.Err, e.Description)
}

// WithInternalError adds internal error information to the error
func (e *OAuthError) WithInternalError(err error) *OAuthError {
	e.InternalError = err
	return e
}

// WithInternalMessage adds internal message information to the error
func (e *OAuthError) WithInternalMessage(fmtString string, args ...interface{}) *OAuthError {
	e.InternalMessage = fmt.Sprintf(fmtString, args...)
	return e
}

// Cause returns the root cause error
func (e *OAuthError) Cause() error {
	if e.InternalError != nil {
		return e.InternalError
	}
	return e
}

func OauthError(err string, description string) *OAuthError {
	return &OAuthError{Err: err, Description: description}
}

func BadRequestError(fmtString string, args ...interface{}) *HTTPError {
	return httpError(http.StatusBadRequest, fmtString, args...)
}

func InternalServerError(fmtString string, args ...interface{}) *HTTPError {
	return httpError(http.StatusInternalServerError, fmtString, args...)
}

func NotFoundError(fmtString string, args ...interface{}) *HTTPError {
	return httpError(http.StatusNotFound, fmtString, args...)
}

func UnauthorizedError(fmtString string, args ...interface{}) *HTTPError {
	return httpError(http.StatusUnauthorized, fmtString, args...)
}

func ForbiddenError(fmtString string, args ...interface{}) *HTTPError {
	return httpError(http.StatusForbidden, fmtString, args...)
}

func UnprocessableEntityError(fmtString string, args ...interface{}) *HTTPError {
	return httpError(http.StatusUnprocessableEntity, fmtString, args...)
}

// HTTPError is an error with a message and an HTTP status code.
type HTTPError struct {
	Code            int    `json:"code"`
	Message         string `json:"msg"`
	InternalError   error  `json:"-"`
	InternalMessage string `json:"-"`
	ErrorID         string `json:"error_id,omitempty"`
}

func (e *HTTPError) Error() string {
	if e.InternalMessage != "" {
		return e.InternalMessage
	}
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

// Cause returns the root cause error
func (e *HTTPError) Cause() error {
	if e.InternalError != nil {
		return e.InternalError
	}
	return e
}

// WithInternalError adds internal error information to the error
func (e *HTTPError) WithInternalError(err error) *HTTPError {
	e.InternalError = err
	return e
}

// WithInternalMessage adds internal message information to the error
func (e *HTTPError) WithInternalMessage(fmtString string, args ...interface{}) *HTTPError {
	e.InternalMessage = fmt.Sprintf(fmtString, args...)
	return e
}

func httpError(code int, fmtString string, args ...interface{}) *HTTPError {
	return &HTTPError{
		Code:    code,
		Message: fmt.Sprintf(fmtString, args...),
	}
}

type ErrorCause interface {
	Cause() error
}

func HandleError(err error, w http.ResponseWriter, r *http.Request) {
	log := logs.GetLogEntry(r)
	errorID := middleware.GetReqID(r.Context())
	switch e := err.(type) {
	case *HTTPError:
		if e.Code >= http.StatusInternalServerError {
			e.ErrorID = errorID
			// this will get us the stack trace too
			log.With(zap.Error(e.Cause())).Error(e.Error())
		} else {
			log.With(zap.Error(e.Cause())).Warn(e.Error())
		}
		if jsonErr := render.SendJSON(w, e.Code, e); jsonErr != nil {
			HandleError(jsonErr, w, r)
		}
	case *OAuthError:
		log.With(zap.Error(e.Cause())).Info(e.Error())
		if jsonErr := render.SendJSON(w, http.StatusBadRequest, e); jsonErr != nil {
			HandleError(jsonErr, w, r)
		}
	case ErrorCause:
		HandleError(e.Cause(), w, r)
	default:
		log.With(zap.Error(e)).Error(e.Error())

		// hide real error details from response to prevent info leaks
		w.WriteHeader(http.StatusInternalServerError)
		if _, writeErr := w.Write([]byte(`{"code":500,"msg":"Internal server error","error_id":"` + errorID + `"}`)); writeErr != nil {
			log.With(zap.Error(writeErr)).Error(e.Error())
		}
	}
}
