// Package errors provides a custom error handling system that extends the
// standard library errors package.
// It includes support for typed errors with HTTP status code mapping, error codes,
// and detailed error information.
// This package is designed to provide consistent error handling and reporting across applications.
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

// ErrorType represents a categorical classification of errors that can be mapped to
// HTTP status codes.
// It helps in providing consistent error responses across the application.
type (
	ErrorType string
	// ErrorCode is a unique identifier for specific error scenarios within the application.
	// It allows for precise error identification and handling.
	ErrorCode string
)

func (c ErrorCode) String() string { return string(c) }

// StringPtr returns the ErrorCode as a string pointer.
func (c ErrorCode) StringPtr() *string { p := string(c); return &p }

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
	case ErrorTypeConflict:
		return http.StatusConflict
	case ErrorTypeInternalError, ErrorTypePanic:
		return http.StatusInternalServerError
	default:
		return -1
	}
}

// HTTPStatusInt32Ptr returns the corresponding HTTP Status as an int32 pointer.
func (t ErrorType) HTTPStatusInt32Ptr() *int32 {
	//nolint:gosec // disable G115
	sts := int32(t.HTTPStatus())

	return &sts
}

// Available error types mapped to their corresponding string representations.
// These types are designed to align with common HTTP status codes for consistent API responses.
const (
	// ErrorTypeInvalid represents validation errors or invalid input.
	ErrorTypeInvalid ErrorType = "invalid"
	// ErrorTypeNotFound represents resource not found errors.
	ErrorTypeNotFound ErrorType = "not-found"
	// ErrorTypeConflict represents resource already exists errors.
	ErrorTypeConflict ErrorType = "conflict"
	// ErrorTypeUnprocessableContent represents semantic errors in the request content.
	ErrorTypeUnprocessableContent ErrorType = "unprocessable-content"
	// ErrorTypeUnauthorized represents permission denied errors.
	ErrorTypeUnauthorized ErrorType = "unauthorzied"
	// ErrorTypeUnauthenticated represents authentication failures.
	ErrorTypeUnauthenticated ErrorType = "unauthenticated"
	// ErrorTypeInternalError represents unexpected internal server errors.
	ErrorTypeInternalError ErrorType = "internal-error"
	// ErrorTypePanic represents errors from recovered panics.
	ErrorTypePanic ErrorType = "panic"
)

// Error represents a structured error with additional context and details.
// It implements the error interface while providing rich error information
// that can be used for both logging and client responses.
type Error struct {
	// Type categorizes the error and determines its HTTP status code
	Type ErrorType
	// Code identifies the specific error scenario
	Code ErrorCode
	// Message is a user-friendly error description
	Message string
	// InternalMessage contains technical details (not intended for client response)
	InternalMessage string
	// ErrorDetails provides additional structured information about the error
	ErrorDetails ErrorDetails
}

func (e *Error) Error() string {
	return fmt.Sprintf(
		"type: %s, code: %s, message: %s, internal message: %s",
		e.Type,
		e.Code,
		e.Message,
		e.InternalMessage,
	)
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

// IsErrorDetailCode checks if the error contains a specific ErrorDetailCode.
func IsErrorDetailCode(err error, code ErrorDetailCode) bool {
	var e *Error

	if errors.As(err, &e) {
		return e.ErrorDetails.ContainsErrorCode(code)
	}

	return false
}

// ErrorDetailCode contains additional specific codes to provide context to the error.
type ErrorDetailCode string

func (c ErrorDetailCode) String() string { return string(c) }

// StringPtr returns the ErrorDetailCode as a string pointer.
func (c ErrorDetailCode) StringPtr() *string { p := string(c); return &p }

// ErrorDetail provides additional context for an error through a code and message.
// It can be used to communicate multiple specific issues within a single error.
type ErrorDetail struct {
	// Code identifies the specific detail type
	Code ErrorDetailCode
	// Message provides a description of the specific detail
	Message string
}

// ErrorDetails is a collection of ErrorDetail that allows for reporting multiple
// related issues within a single error response.
type ErrorDetails []ErrorDetail

// ContainsErrorCode checks if the ErrorDetails contains a specific ErrorDetailCode.
func (e ErrorDetails) ContainsErrorCode(code ErrorDetailCode) bool {
	for _, d := range e {
		if d.Code == code {
			return true
		}
	}

	return false
}
