package errors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppError(t *testing.T) {
	t.Run("NotFound error constructor", func(t *testing.T) {
		err := NewNotFound("item not found")
		assert.Equal(t, CodeNotFound, err.Code)
		assert.Equal(t, "item not found", err.Message)
		assert.Equal(t, http.StatusNotFound, err.HTTPStatus)
		assert.Contains(t, err.Error(), CodeNotFound)
		assert.True(t, errors.Is(err, ErrNotFound))
		assert.True(t, IsNotFound(err))
	})

	t.Run("Validation error constructor", func(t *testing.T) {
		details := map[string]string{"field": "email", "reason": "invalid format"}
		err := NewValidation("validation failed", details)
		assert.Equal(t, CodeValidation, err.Code)
		assert.Equal(t, http.StatusUnprocessableEntity, err.HTTPStatus)
		assert.Equal(t, details, err.Details)
		assert.Equal(t, details, err.Fields)
		assert.True(t, errors.Is(err, ErrValidation))
	})

	t.Run("Unauthorized error constructor", func(t *testing.T) {
		err := NewUnauthorized("invalid credentials")
		assert.Equal(t, CodeUnauthorized, err.Code)
		assert.Equal(t, http.StatusUnauthorized, err.HTTPStatus)
		assert.True(t, errors.Is(err, ErrUnauthorized))
		assert.True(t, IsUnauthorized(err))
	})

	t.Run("Conflict error constructor", func(t *testing.T) {
		err := NewConflict("already exists")
		assert.Equal(t, CodeConflict, err.Code)
		assert.Equal(t, http.StatusConflict, err.HTTPStatus)
		assert.True(t, errors.Is(err, ErrConflict))
		assert.True(t, IsConflict(err))
	})

	t.Run("BadRequest error constructor", func(t *testing.T) {
		err := NewBadRequest("bad input")
		assert.Equal(t, CodeBadRequest, err.Code)
		assert.Equal(t, http.StatusBadRequest, err.HTTPStatus)
		assert.True(t, errors.Is(err, ErrBadRequest))
		assert.True(t, IsBadRequest(err))
	})

	t.Run("TooManyRequests error constructor", func(t *testing.T) {
		err := NewTooManyRequests("rate limit reached")
		assert.Equal(t, CodeTooManyRequests, err.Code)
		assert.Equal(t, http.StatusTooManyRequests, err.HTTPStatus)
		assert.True(t, errors.Is(err, ErrTooManyRequests))
	})

	t.Run("ServiceUnavailable error constructor", func(t *testing.T) {
		err := NewServiceUnavailable("service down")
		assert.Equal(t, CodeServiceUnavailable, err.Code)
		assert.Equal(t, http.StatusServiceUnavailable, err.HTTPStatus)
		assert.True(t, errors.Is(err, ErrUnavailable))
	})

	t.Run("Timeout error constructor", func(t *testing.T) {
		err := NewTimeout("timed out")
		assert.Equal(t, CodeTimeout, err.Code)
		assert.Equal(t, http.StatusGatewayTimeout, err.HTTPStatus)
		assert.True(t, errors.Is(err, ErrTimeout))
	})

	t.Run("Error wrapping and unwrapping", func(t *testing.T) {
		cause := errors.New("underlying failure")
		wrapped := NewNotFound("tenant missing", cause)
		assert.Equal(t, cause, errors.Unwrap(wrapped))
		assert.True(t, errors.Is(wrapped, cause))
		assert.True(t, errors.Is(wrapped, ErrNotFound))

		// Multi-level wrapping with fmt.Errorf
		higherLevel := fmt.Errorf("use-case execution: %w", wrapped)
		assert.True(t, errors.Is(higherLevel, ErrNotFound))
		assert.True(t, errors.Is(higherLevel, cause))

		fromErr := FromError(higherLevel)
		require.NotNil(t, fromErr)
		assert.Equal(t, CodeNotFound, fromErr.Code)
		assert.Equal(t, http.StatusNotFound, fromErr.HTTPStatus)
	})

	t.Run("FromError with AppError", func(t *testing.T) {
		orig := NewForbidden("access denied")
		extracted := FromError(orig)
		assert.Equal(t, orig, extracted)
	})

	t.Run("FromError with sentinel errors", func(t *testing.T) {
		errNotFound := FromError(ErrNotFound)
		assert.Equal(t, CodeNotFound, errNotFound.Code)
		assert.Equal(t, http.StatusNotFound, errNotFound.HTTPStatus)

		errConflict := FromError(ErrConflict)
		assert.Equal(t, CodeConflict, errConflict.Code)
		assert.Equal(t, http.StatusConflict, errConflict.HTTPStatus)

		errUnauthorized := FromError(ErrUnauthorized)
		assert.Equal(t, CodeUnauthorized, errUnauthorized.Code)
		assert.Equal(t, http.StatusUnauthorized, errUnauthorized.HTTPStatus)

		errForbidden := FromError(ErrForbidden)
		assert.Equal(t, CodeForbidden, errForbidden.Code)
		assert.Equal(t, http.StatusForbidden, errForbidden.HTTPStatus)

		errTimeout := FromError(context.DeadlineExceeded)
		assert.Equal(t, CodeTimeout, errTimeout.Code)
		assert.Equal(t, http.StatusGatewayTimeout, errTimeout.HTTPStatus)
	})

	t.Run("FromError with standard error", func(t *testing.T) {
		stdErr := errors.New("raw failure")
		converted := FromError(stdErr)
		assert.Equal(t, CodeInternal, converted.Code)
		assert.Equal(t, http.StatusInternalServerError, converted.HTTPStatus)
		assert.True(t, errors.Is(converted, stdErr))
	})

	t.Run("FromError with nil", func(t *testing.T) {
		assert.Nil(t, FromError(nil))
	})
}

func TestParseBindingError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		assert.Nil(t, ParseBindingError(nil))
	})

	t.Run("io.EOF empty body", func(t *testing.T) {
		err := ParseBindingError(io.EOF)
		require.NotNil(t, err)
		assert.Equal(t, CodeBadRequest, err.Code)
		assert.Equal(t, http.StatusBadRequest, err.HTTPStatus)
		assert.Equal(t, "request body cannot be empty", err.Message)
	})

	t.Run("json.SyntaxError malformed json", func(t *testing.T) {
		var target struct{ Name string }
		decodeErr := json.Unmarshal([]byte(`{"name": `), &target)
		err := ParseBindingError(decodeErr)
		require.NotNil(t, err)
		assert.Equal(t, CodeBadRequest, err.Code)
		assert.Equal(t, http.StatusBadRequest, err.HTTPStatus)
		assert.Contains(t, err.Message, "malformed JSON")
	})

	t.Run("json.UnmarshalTypeError invalid field type", func(t *testing.T) {
		var target struct {
			Age int `json:"age"`
		}
		decodeErr := json.Unmarshal([]byte(`{"age": "twenty"}`), &target)
		err := ParseBindingError(decodeErr)
		require.NotNil(t, err)
		assert.Equal(t, CodeValidation, err.Code)
		assert.Equal(t, http.StatusUnprocessableEntity, err.HTTPStatus)
		assert.Contains(t, err.Fields["age"], "invalid type")
	})

	t.Run("validator.ValidationErrors struct validation", func(t *testing.T) {
		type testReq struct {
			Email string `validate:"required,email"`
			Name  string `validate:"required,min=3"`
			Role  string `validate:"required,oneof=admin user"`
		}

		validate := validator.New()
		valErr := validate.Struct(testReq{
			Email: "not-an-email",
			Name:  "ab",
			Role:  "superadmin",
		})
		require.Error(t, valErr)

		err := ParseBindingError(valErr)
		require.NotNil(t, err)
		assert.Equal(t, CodeValidation, err.Code)
		assert.Equal(t, http.StatusUnprocessableEntity, err.HTTPStatus)
		assert.NotEmpty(t, err.Fields["email"])
		assert.NotEmpty(t, err.Fields["name"])
		assert.NotEmpty(t, err.Fields["role"])
	})
}
