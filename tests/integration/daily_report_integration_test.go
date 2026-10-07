package integration

import (
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

func TestDailyReport_IntegrationRoutes(t *testing.T) {
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
	tenantID := uuid.New()
	tenantUserID := uuid.New()
	platformAdminID := uuid.New()

	userToken, err := jwtManager.GenerateTokenPair(tenantUserID, &tenantID, "user@tenant.com", auth.RoleTenantUser, auth.UserTypeTenantUser)
	require.NoError(t, err)

	platformToken, err := jwtManager.GenerateTokenPair(platformAdminID, nil, "admin@platform.com", auth.RolePlatformAdmin, auth.UserTypePlatformAdmin)
	require.NoError(t, err)

	// 1. Unauthenticated requests rejected
	t.Run("Unauthenticated JSON and PDF requests rejected with 401", func(t *testing.T) {
		w1 := httptest.NewRecorder()
		req1, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/reports/daily?date=2026-10-07", nil)
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusUnauthorized, w1.Code)

		w2 := httptest.NewRecorder()
		req2, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/reports/daily/pdf?date=2026-10-07", nil)
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusUnauthorized, w2.Code)

		w3 := httptest.NewRecorder()
		req3, _ := http.NewRequest(http.MethodGet, "/api/v1/reports/daily?date=2026-10-07", nil)
		router.ServeHTTP(w3, req3)
		assert.Equal(t, http.StatusUnauthorized, w3.Code)
	})

	// 2. Platform admin token rejected for tenant-scoped reports
	t.Run("Platform admin token rejected for tenant report endpoints", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/reports/daily?date=2026-10-07", nil)
		req.Header.Set("Authorization", "Bearer "+platformToken.AccessToken)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	// 3. Invalid date parameter format returns 400 Bad Request
	t.Run("Invalid date returns 400 Bad Request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/reports/daily?date=invalid-date", nil)
		req.Header.Set("Authorization", "Bearer "+userToken.AccessToken)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// 4. Web view /reports/daily returns 200 OK
	t.Run("Web Page GET /reports/daily returns 200 OK with report template", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/reports/daily", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()
		assert.Contains(t, body, "Daily Business Report")
		assert.Contains(t, body, "Report Date:")
		assert.Contains(t, body, "Download PDF")
	})
}
