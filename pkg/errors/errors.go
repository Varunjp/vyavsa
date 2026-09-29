package errors

import (
	"context"
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
	CodeTooManyRequests    = "TOO_MANY_REQUESTS"
	CodeTimeout            = "REQUEST_TIMEOUT"
)

// Standard Sentinel Application Errors
var (
	ErrNotFound        = errors.New("resource not found")
	ErrInvalidInput    = errors.New("invalid input")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrForbidden       = errors.New("forbidden")
	ErrConflict        = errors.New("conflict")
	ErrValidation      = errors.New("validation failed")
	ErrInternal        = errors.New("internal server error")
	ErrUnavailable     = errors.New("service unavailable")
	ErrTimeout         = errors.New("operation timed out")
	ErrBadRequest      = errors.New("bad request")
	ErrTooManyRequests = errors.New("too many requests")
)

// AppError represents an application-level structured error
type AppError struct {
	Code       string            `json:"code"`
	Message    string            `json:"message"`
	HTTPStatus int               `json:"-"`
	Err        error             `json:"-"`
	Details    map[string]string `json:"details,omitempty"`
	Fields     map[string]string `json:"fields,omitempty"`
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

// Is enables standard Go errors.Is matching against AppError or sentinel errors
func (e *AppError) Is(target error) bool {
	if target == nil {
		return false
	}

	if t, ok := target.(*AppError); ok {
		return e.Code == t.Code
	}

	switch target {
	case ErrNotFound:
		return e.Code == CodeNotFound
	case ErrUnauthorized:
		return e.Code == CodeUnauthorized
	case ErrForbidden:
		return e.Code == CodeForbidden
	case ErrConflict:
		return e.Code == CodeConflict
	case ErrValidation:
		return e.Code == CodeValidation
	case ErrInvalidInput, ErrBadRequest:
		return e.Code == CodeBadRequest || e.Code == CodeValidation
	case ErrInternal:
		return e.Code == CodeInternal || e.Code == CodeDatabase
	case ErrUnavailable:
		return e.Code == CodeServiceUnavailable
	case ErrTimeout:
		return e.Code == CodeTimeout
	case ErrTooManyRequests:
		return e.Code == CodeTooManyRequests
	}

	if e.Err != nil {
		return errors.Is(e.Err, target)
	}

	return false
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
func NewNotFound(message string, err ...error) *AppError {
	var underlying error
	if len(err) > 0 {
		underlying = err[0]
	}
	return &AppError{
		Code:       CodeNotFound,
		Message:    message,
		HTTPStatus: http.StatusNotFound,
		Err:        underlying,
	}
}

// NewUnauthorized creates a 401 Unauthorized error
func NewUnauthorized(message string, err ...error) *AppError {
	var underlying error
	if len(err) > 0 {
		underlying = err[0]
	}
	return &AppError{
		Code:       CodeUnauthorized,
		Message:    message,
		HTTPStatus: http.StatusUnauthorized,
		Err:        underlying,
	}
}

// NewForbidden creates a 403 Forbidden error
func NewForbidden(message string, err ...error) *AppError {
	var underlying error
	if len(err) > 0 {
		underlying = err[0]
	}
	return &AppError{
		Code:       CodeForbidden,
		Message:    message,
		HTTPStatus: http.StatusForbidden,
		Err:        underlying,
	}
}

// NewValidation creates a 422 Unprocessable Entity validation error
func NewValidation(message string, details map[string]string, err ...error) *AppError {
	var underlying error
	if len(err) > 0 {
		underlying = err[0]
	}
	return &AppError{
		Code:       CodeValidation,
		Message:    message,
		HTTPStatus: http.StatusUnprocessableEntity,
		Details:    details,
		Fields:     details,
		Err:        underlying,
	}
}

// NewConflict creates a 409 Conflict error
func NewConflict(message string, err ...error) *AppError {
	var underlying error
	if len(err) > 0 {
		underlying = err[0]
	}
	return &AppError{
		Code:       CodeConflict,
		Message:    message,
		HTTPStatus: http.StatusConflict,
		Err:        underlying,
	}
}

// NewBadRequest creates a 400 Bad Request error
func NewBadRequest(message string, err ...error) *AppError {
	var underlying error
	if len(err) > 0 {
		underlying = err[0]
	}
	return &AppError{
		Code:       CodeBadRequest,
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
		Err:        underlying,
	}
}

// NewTooManyRequests creates a 429 Too Many Requests rate limit error
func NewTooManyRequests(message string, err ...error) *AppError {
	var underlying error
	if len(err) > 0 {
		underlying = err[0]
	}
	return &AppError{
		Code:       CodeTooManyRequests,
		Message:    message,
		HTTPStatus: http.StatusTooManyRequests,
		Err:        underlying,
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
func NewServiceUnavailable(message string, err ...error) *AppError {
	var underlying error
	if len(err) > 0 {
		underlying = err[0]
	}
	return &AppError{
		Code:       CodeServiceUnavailable,
		Message:    message,
		HTTPStatus: http.StatusServiceUnavailable,
		Err:        underlying,
	}
}

// NewTimeout creates a 504 Gateway Timeout error
func NewTimeout(message string, err ...error) *AppError {
	var underlying error
	if len(err) > 0 {
		underlying = err[0]
	}
	return &AppError{
		Code:       CodeTimeout,
		Message:    message,
		HTTPStatus: http.StatusGatewayTimeout,
		Err:        underlying,
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

	// Match against standard sentinel errors
	if errors.Is(err, ErrNotFound) {
		return NewNotFound(err.Error(), err)
	}
	if errors.Is(err, ErrConflict) {
		return NewConflict(err.Error(), err)
	}
	if errors.Is(err, ErrUnauthorized) {
		return NewUnauthorized(err.Error(), err)
	}
	if errors.Is(err, ErrForbidden) {
		return NewForbidden(err.Error(), err)
	}
	if errors.Is(err, ErrValidation) {
		return NewValidation(err.Error(), nil, err)
	}
	if errors.Is(err, ErrInvalidInput) || errors.Is(err, ErrBadRequest) {
		return NewBadRequest(err.Error(), err)
	}
	if errors.Is(err, ErrTooManyRequests) {
		return NewTooManyRequests(err.Error(), err)
	}
	if errors.Is(err, ErrUnavailable) {
		return NewServiceUnavailable(err.Error(), err)
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrTimeout) {
		return NewTimeout("request timed out", err)
	}
	if errors.Is(err, context.Canceled) {
		return New("REQUEST_CANCELED", "request was canceled", 499, err)
	}

	return NewInternal(err)
}

// IsNotFound checks whether the error represents a not found condition
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeNotFound
	}
	return errors.Is(err, ErrNotFound)
}

// IsConflict checks whether the error represents a conflict condition
func IsConflict(err error) bool {
	if err == nil {
		return false
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeConflict
	}
	return errors.Is(err, ErrConflict)
}

// IsUnauthorized checks whether the error represents an unauthorized condition
func IsUnauthorized(err error) bool {
	if err == nil {
		return false
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeUnauthorized
	}
	return errors.Is(err, ErrUnauthorized)
}

// IsForbidden checks whether the error represents a forbidden condition
func IsForbidden(err error) bool {
	if err == nil {
		return false
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeForbidden
	}
	return errors.Is(err, ErrForbidden)
}

// IsBadRequest checks whether the error represents a bad request condition
func IsBadRequest(err error) bool {
	if err == nil {
		return false
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeBadRequest
	}
	return errors.Is(err, ErrBadRequest) || errors.Is(err, ErrInvalidInput)
}
