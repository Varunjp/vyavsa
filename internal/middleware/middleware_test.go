package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRequestIDMiddleware(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test-id", func(c *gin.Context) {
		reqID := logger.RequestIDFromContext(c.Request.Context())
		c.String(http.StatusOK, reqID)
	})

	t.Run("Generates Request ID when not provided", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test-id", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		headerID := w.Header().Get(HeaderXRequestID)
		assert.NotEmpty(t, headerID)
		assert.Equal(t, headerID, w.Body.String())
	})

	t.Run("Preserves provided Request ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test-id", nil)
		req.Header.Set(HeaderXRequestID, "custom-client-id-999")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "custom-client-id-999", w.Header().Get(HeaderXRequestID))
		assert.Equal(t, "custom-client-id-999", w.Body.String())
	})
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	r := gin.New()
	r.Use(SecurityHeaders(true))
	r.GET("/test-security", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test-security", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
	assert.Contains(t, w.Header().Get("Strict-Transport-Security"), "max-age=")
}

func TestRecoveryMiddleware(t *testing.T) {
	buf := &bytes.Buffer{}
	log := logger.New(logger.Config{
		Environment: "test",
		ServiceName: "test",
		Output:      buf,
	})

	r := gin.New()
	r.Use(Recovery(log))
	r.GET("/panic", func(c *gin.Context) {
		panic("simulated explosion")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/panic", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "INTERNAL_SERVER_ERROR")
}

func TestCORSMiddleware(t *testing.T) {
	r := gin.New()
	r.Use(CORS([]string{"https://example.com"}))
	r.GET("/api/data", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	t.Run("Preflight OPTIONS returns 204", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodOptions, "/api/data", nil)
		req.Header.Set("Origin", "https://example.com")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
	})
}
