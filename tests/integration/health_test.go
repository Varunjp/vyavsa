package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/Varunjp/vyavsa/internal/metrics"
	"github.com/Varunjp/vyavsa/internal/middleware"
	"github.com/Varunjp/vyavsa/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFullHTTPPipelineIntegration(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			Name:            "test-app",
			Env:             "test",
			Port:            "8080",
			ShutdownTimeout: 2 * time.Second,
		},
		Metrics: config.MetricsConfig{
			Enabled: true,
			Path:    "/metrics",
		},
		CORS: config.CORSConfig{
			AllowedOrigins: []string{"*"},
		},
	}

	appLogger := logger.Default()
	appMetrics := metrics.New()

	srv := server.New(cfg, appLogger, nil, nil, appMetrics)
	router := srv.Router()

	t.Run("GET /health returns 200 OK and injects X-Request-ID header", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/health", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NotEmpty(t, w.Header().Get(middleware.HeaderXRequestID))
		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))

		var resp map[string]any
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "UP", resp["status"])
	})

	t.Run("GET /ready without dependencies returns 503 Service Unavailable", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/ready", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		var resp map[string]any
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "NOT_READY", resp["status"])
	})

	t.Run("GET /metrics returns Prometheus metric format", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/metrics", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "billbook_http_requests_total")
	})

	t.Run("GET /api/v1/ping returns standard success envelope", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/api/v1/ping", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"success":true`)
		assert.Contains(t, w.Body.String(), `"pong":true`)
	})

	t.Run("Unknown route returns standard 404 error envelope", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/non-existent-path", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), `"success":false`)
		assert.Contains(t, w.Body.String(), `ROUTE_NOT_FOUND`)
	})
}
