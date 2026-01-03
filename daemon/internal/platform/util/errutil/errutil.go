package errutil

import (
	"errors"
	"fmt"
)

// ErrorType represents the category of error.
type ErrorType string

const (
	ErrorTypeBadRequest      ErrorType = "bad_request"
	ErrorTypeUnauthorized    ErrorType = "unauthorized"
	ErrorTypeForbidden       ErrorType = "forbidden"
	ErrorTypeNotFound        ErrorType = "not_found"
	ErrorTypeInternal        ErrorType = "internal"
	ErrorTypePayloadTooLarge ErrorType = "payload_too_large"
)

// AppError represents a structured application error.
type AppError struct {
	Type    ErrorType
	Message string
	Cause   error
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying cause for error chain support.
func (e *AppError) Unwrap() error {
	return e.Cause
}

// BadRequest creates a bad request error.
func BadRequest(msg string) *AppError {
	return &AppError{Type: ErrorTypeBadRequest, Message: msg}
}

// BadRequestf creates a bad request error with formatted message.
func BadRequestf(format string, args ...any) *AppError {
	return &AppError{Type: ErrorTypeBadRequest, Message: fmt.Sprintf(format, args...)}
}

// Unauthorized creates an unauthorized error.
func Unauthorized(msg string) *AppError {
	return &AppError{Type: ErrorTypeUnauthorized, Message: msg}
}

// Unauthorizedf creates an unauthorized error with formatted message.
func Unauthorizedf(format string, args ...any) *AppError {
	return &AppError{Type: ErrorTypeUnauthorized, Message: fmt.Sprintf(format, args...)}
}

// Forbidden creates a forbidden error.
func Forbidden(msg string) *AppError {
	return &AppError{Type: ErrorTypeForbidden, Message: msg}
}

// Forbiddenf creates a forbidden error with formatted message.
func Forbiddenf(format string, args ...any) *AppError {
	return &AppError{Type: ErrorTypeForbidden, Message: fmt.Sprintf(format, args...)}
}

// NotFound creates a not found error.
func NotFound(msg string) *AppError {
	return &AppError{Type: ErrorTypeNotFound, Message: msg}
}

// NotFoundf creates a not found error with formatted message.
func NotFoundf(format string, args ...any) *AppError {
	return &AppError{Type: ErrorTypeNotFound, Message: fmt.Sprintf(format, args...)}
}

// Internal creates an internal error.
func Internal(msg string) *AppError {
	return &AppError{Type: ErrorTypeInternal, Message: msg}
}

// Internalf creates an internal error with formatted message.
func Internalf(format string, args ...any) *AppError {
	return &AppError{Type: ErrorTypeInternal, Message: fmt.Sprintf(format, args...)}
}

// Wrap wraps an existing error with an AppError type.
func Wrap(errType ErrorType, msg string, cause error) *AppError {
	return &AppError{Type: errType, Message: msg, Cause: cause}
}

// Is checks if the error is of a specific ErrorType.
func Is(err error, errType ErrorType) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == errType
	}
	return false
}

// IsBadRequest checks if the error is a bad request error.
func IsBadRequest(err error) bool {
	return Is(err, ErrorTypeBadRequest)
}

// IsUnauthorized checks if the error is an unauthorized error.
func IsUnauthorized(err error) bool {
	return Is(err, ErrorTypeUnauthorized)
}

// IsForbidden checks if the error is a forbidden error.
func IsForbidden(err error) bool {
	return Is(err, ErrorTypeForbidden)
}

// IsNotFound checks if the error is a not found error.
func IsNotFound(err error) bool {
	return Is(err, ErrorTypeNotFound)
}

// IsInternal checks if the error is an internal error.
func IsInternal(err error) bool {
	return Is(err, ErrorTypeInternal)
}

// PayloadTooLarge creates a payload too large error.
func PayloadTooLarge(msg string) *AppError {
	return &AppError{Type: ErrorTypePayloadTooLarge, Message: msg}
}

// PayloadTooLargef creates a payload too large error with formatted message.
func PayloadTooLargef(format string, args ...any) *AppError {
	return &AppError{Type: ErrorTypePayloadTooLarge, Message: fmt.Sprintf(format, args...)}
}

// IsPayloadTooLarge checks if the error is a payload too large error.
func IsPayloadTooLarge(err error) bool {
	return Is(err, ErrorTypePayloadTooLarge)
}
