package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// promote standard library errors package functions.
var (
	Is   = errors.Is
	As   = errors.As
	Join = errors.Join
	New  = errors.New
)

type (
	// ErrorType is mapped to HTTP codes.
	ErrorType string
	// ErrorCode represents application error codes.
	ErrorCode string
)

func (c ErrorCode) String() string { return string(c) }

func (t ErrorType) String() string {
	return string(t)
}

// HTTPStatus returns the corresponding HTTP Status.
func (t ErrorType) HTTPStatus() int {
	switch t {
	case ErrorTypeInvalid:
		return http.StatusBadRequest
	case ErrorTypeNotFound:
		return http.StatusNotFound
	case ErrorTypeUnprocessableContent:
		return http.StatusUnprocessableEntity
	case ErrorTypeUnauthorized:
		return http.StatusUnauthorized
	case ErrorTypeUnauthenticated:
		return http.StatusUnauthorized
	case ErrorTypeInternalError, ErrorTypePanic:
		return http.StatusInternalServerError
	default:
		return -1
	}
}

// HTTPStatusInt32Ptr returns the corresponding HTTP Status as an int32 pointer.
func (t ErrorType) HTTPStatusInt32Ptr() *int32 {
	sts := int32(t.HTTPStatus())

	return &sts
}

// Available error types.
const (
	ErrorTypeInvalid              ErrorType = "invalid"
	ErrorTypeNotFound             ErrorType = "not-found"
	ErrorTypeUnprocessableContent ErrorType = "unprocessable-content"
	ErrorTypeUnauthorized         ErrorType = "unauthorzied"
	ErrorTypeUnauthenticated      ErrorType = "unauthenticated"
	ErrorTypeInternalError        ErrorType = "internal-error"
	ErrorTypePanic                ErrorType = "panic"
)

// Error object.
type Error struct {
	Type         ErrorType
	Code         ErrorCode
	Message      string
	ErrorDetails []ErrorDetail
}

func (e *Error) Error() string {
	return fmt.Sprintf("type: %s, code: %s, details: %s", e.Type, e.Code, e.Message)
}

// IsErrorType checks if the error is of the given type.
func IsErrorType(err error, typ ErrorType) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.Type == typ
	}

	return false
}

// IsErrorCode checks if the error is of the given code.
func IsErrorCode(err error, code ErrorCode) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.Code == code
	}

	return false
}

// ErrorDetailCode contains additional specific codes to provide context to the error.
type ErrorDetailCode string

// ErrorDetail provides explicit details on an Error.
type ErrorDetail struct {
	Code    ErrorDetailCode
	Message string
}
