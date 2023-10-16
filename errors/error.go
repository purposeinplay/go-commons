package errors

import (
	"errors"
	"fmt"
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
	ErrorCode int
)

func (t ErrorType) String() string {
	return string(t)
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
	Type    ErrorType
	Code    ErrorCode
	Details string
}

func (e *Error) Error() string {
	return fmt.Sprintf("type: %s, code: %d, details: %s", e.Type, e.Code, e.Details)
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