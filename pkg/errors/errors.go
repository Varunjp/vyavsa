package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Standard Error Codes
const (
	CodeNotFound           = "NOT_FOUND"
	CodeUnauthorized       = "UNAUTHORIZED"
	CodeForbidden          = "FORBIDDEN"
	CodeValidation         = "VALIDATION_ERROR"
	CodeConflict           = "CONFLICT"
	CodeInternal           = "INTERNAL_ERROR"
	CodeDatabase           = "DATABASE_ERROR"
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	CodeBadRequest         = "BAD_REQUEST"
)

// AppError represents an application-level structured error
type AppError struct {
	Code       string            `json:"code"`
	Message    string            `json:"message"`
	HTTPStatus int               `json:"-"`
	Err        error             `json:"-"`
	Details    map[string]string `json:"details,omitempty"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new custom AppError
func New(code, message string, httpStatus int, err error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Err:        err,
	}
}

// NewNotFound creates a 404 Not Found error
func NewNotFound(message string) *AppError {
	return &AppError{
		Code:       CodeNotFound,
		Message:    message,
		HTTPStatus: http.StatusNotFound,
	}
}

// NewUnauthorized creates a 401 Unauthorized error
func NewUnauthorized(message string) *AppError {
	return &AppError{
		Code:       CodeUnauthorized,
		Message:    message,
		HTTPStatus: http.StatusUnauthorized,
	}
}

// NewForbidden creates a 403 Forbidden error
func NewForbidden(message string) *AppError {
	return &AppError{
		Code:       CodeForbidden,
		Message:    message,
		HTTPStatus: http.StatusForbidden,
	}
}

// NewValidation creates a 422 Unprocessable Entity validation error
func NewValidation(message string, details map[string]string) *AppError {
	return &AppError{
		Code:       CodeValidation,
		Message:    message,
		HTTPStatus: http.StatusUnprocessableEntity,
		Details:    details,
	}
}

// NewConflict creates a 409 Conflict error
func NewConflict(message string) *AppError {
	return &AppError{
		Code:       CodeConflict,
		Message:    message,
		HTTPStatus: http.StatusConflict,
	}
}

// NewBadRequest creates a 400 Bad Request error
func NewBadRequest(message string) *AppError {
	return &AppError{
		Code:       CodeBadRequest,
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
	}
}

// NewInternal creates a 500 Internal Server Error
func NewInternal(err error) *AppError {
	return &AppError{
		Code:       CodeInternal,
		Message:    "an unexpected internal error occurred",
		HTTPStatus: http.StatusInternalServerError,
		Err:        err,
	}
}

// NewDatabase creates a 500 Database Error
func NewDatabase(err error) *AppError {
	return &AppError{
		Code:       CodeDatabase,
		Message:    "a database error occurred",
		HTTPStatus: http.StatusInternalServerError,
		Err:        err,
	}
}

// NewServiceUnavailable creates a 503 Service Unavailable error
func NewServiceUnavailable(message string) *AppError {
	return &AppError{
		Code:       CodeServiceUnavailable,
		Message:    message,
		HTTPStatus: http.StatusServiceUnavailable,
	}
}

// FromError extracts or converts an arbitrary error into an *AppError
func FromError(err error) *AppError {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	return NewInternal(err)
}

// IsNotFound checks whether the error is a NOT_FOUND AppError
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeNotFound
	}
	return false
}

