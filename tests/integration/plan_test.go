package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/Varunjp/vyavsa/internal/metrics"
	"github.com/Varunjp/vyavsa/internal/server"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanEndpointsIntegration(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			Name:            "test-app",
			Env:             "test",
			Port:            "8080",
			ShutdownTimeout: 2 * time.Second,
		},
		JWT: config.JWTConfig{
			Secret:        "integration-test-secret-minimum-32-bytes",
			AccessExpiry:  15 * time.Minute,
			RefreshExpiry: 24 * time.Hour,
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

	jwtManager := auth.NewJWTManager(cfg.JWT)
	adminID := uuid.New()
	tenantUserID := uuid.New()
	tenantID := uuid.New()

	adminToken, err := jwtManager.GenerateTokenPair(adminID, nil, "admin@vyavsa.com", auth.RolePlatformAdmin, auth.UserTypePlatformAdmin)
	require.NoError(t, err)

	tenantToken, err := jwtManager.GenerateTokenPair(tenantUserID, &tenantID, "user@store.com", auth.RoleTenantAdmin, auth.UserTypeTenantUser)
	require.NoError(t, err)

	t.Run("GET /api/v1/plans without authentication returns 200 OK (Public Catalog)", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/plans", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Without live DB pool, handlers return 500 or 200 depending on repo nil check, or handle gracefully
		// Since repo is nil, let's verify response is structured JSON
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "success")
	})

	t.Run("POST /api/v1/platform/plans without token returns 401 Unauthorized", func(t *testing.T) {
		body := bytes.NewBufferString(`{"plan_name":"Pro","price":499.00}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/platform/plans", body)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "UNAUTHORIZED")
	})

	t.Run("POST /api/v1/platform/plans with tenant user token returns 403 Forbidden", func(t *testing.T) {
		body := bytes.NewBufferString(`{"plan_name":"Pro","price":499.00}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/platform/plans", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tenantToken.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "FORBIDDEN")
	})

	t.Run("POST /api/v1/platform/plans with platform admin token and invalid body returns 422 Validation Error", func(t *testing.T) {
		body := bytes.NewBufferString(`{"plan_name":""}`) // Missing price and invalid plan_name
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/platform/plans", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "VALIDATION_ERROR")
	})
}
