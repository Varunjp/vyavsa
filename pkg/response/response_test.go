package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestResponseFormat(t *testing.T) {
	t.Run("Success response", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		Success(c, map[string]string{"foo": "bar"}, "custom success message")

		assert.Equal(t, http.StatusOK, w.Code)

		var resp Response
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "custom success message", resp.Message)
		assert.NotNil(t, resp.Data)
	})

	t.Run("Created response", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		Created(c, map[string]int{"id": 42})

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp Response
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("Paginated response", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		items := []string{"item1", "item2"}
		pagination := Pagination{
			Page:       1,
			PageSize:   20,
			TotalItems: 2,
			TotalPages: 1,
		}

		Paginated(c, items, pagination)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp Response
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotNil(t, resp.Pagination)
		assert.Equal(t, int64(2), resp.Pagination.TotalItems)
	})

	t.Run("Error response with AppError", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		appErr := appErrors.NewNotFound("customer not found")
		Error(c, appErr)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var resp Response
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.False(t, resp.Success)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, appErrors.CodeNotFound, resp.Error.Code)
		assert.Equal(t, "customer not found", resp.Error.Message)
	})
}
