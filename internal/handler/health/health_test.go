package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestHealthHandler(t *testing.T) {
	t.Run("Health check liveness returns UP without DB", func(t *testing.T) {
		h := NewHandler(nil, nil)

		r := gin.New()
		r.GET("/health", h.Health)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/health", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "UP", resp["status"])
		assert.NotEmpty(t, resp["timestamp"])
	})

	t.Run("Ready check reports uninitialized dependencies", func(t *testing.T) {
		h := NewHandler(nil, nil)

		r := gin.New()
		r.GET("/ready", h.Ready)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/ready", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "NOT_READY", resp["status"])
	})
}
