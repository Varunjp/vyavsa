package errors

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppError(t *testing.T) {
	t.Run("NotFound error constructor", func(t *testing.T) {
		err := NewNotFound("item not found")
		assert.Equal(t, CodeNotFound, err.Code)
		assert.Equal(t, "item not found", err.Message)
		assert.Equal(t, http.StatusNotFound, err.HTTPStatus)
		assert.Contains(t, err.Error(), CodeNotFound)
	})

	t.Run("Validation error constructor", func(t *testing.T) {
		details := map[string]string{"field": "email", "reason": "invalid format"}
		err := NewValidation("validation failed", details)
		assert.Equal(t, CodeValidation, err.Code)
		assert.Equal(t, http.StatusUnprocessableEntity, err.HTTPStatus)
		assert.Equal(t, details, err.Details)
	})

	t.Run("Unauthorized error constructor", func(t *testing.T) {
		err := NewUnauthorized("invalid credentials")
		assert.Equal(t, CodeUnauthorized, err.Code)
		assert.Equal(t, http.StatusUnauthorized, err.HTTPStatus)
	})

	t.Run("Conflict error constructor", func(t *testing.T) {
		err := NewConflict("already exists")
		assert.Equal(t, CodeConflict, err.Code)
		assert.Equal(t, http.StatusConflict, err.HTTPStatus)
	})

	t.Run("FromError with AppError", func(t *testing.T) {
		orig := NewForbidden("access denied")
		extracted := FromError(orig)
		assert.Equal(t, orig, extracted)
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
