package integration

import (
	"bytes"
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

func TestTenantEndpointsIntegration(t *testing.T) {
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

	t.Run("POST /api/v1/platform/tenants without token returns 401 Unauthorized", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name":"New Store"}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/platform/tenants", body)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "UNAUTHORIZED")
	})

	t.Run("POST /api/v1/platform/tenants with tenant user token returns 403 Forbidden", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name":"New Store"}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/platform/tenants", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tenantToken.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "FORBIDDEN")
	})

	t.Run("POST /api/v1/platform/tenants with platform admin and invalid payload returns 422 Validation Error", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name":""}`) // Missing admin_email, plan_id, etc.
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/platform/tenants", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "VALIDATION_ERROR")
	})

	t.Run("GET /api/v1/tenant/profile without token returns 401 Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/profile", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "UNAUTHORIZED")
	})

	t.Run("GET /api/v1/tenant/profile with platform admin token returns 403 Forbidden", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/profile", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "FORBIDDEN")
	})

	t.Run("PATCH /api/v1/platform/tenants/:id/status with invalid UUID returns 400 Bad Request", func(t *testing.T) {
		body := bytes.NewBufferString(`{"status":"suspended"}`)
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/platform/tenants/invalid-uuid/status", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "BAD_REQUEST")
	})

	t.Run("POST /api/v1/auth/tenant/register with invalid payload returns 422 Validation Error", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name":"My Store","email":"invalid-email"}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/tenant/register", body)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "VALIDATION_ERROR")
	})

	t.Run("POST /api/v1/tenants/register alias with invalid payload returns 422 Validation Error", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name":""}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/tenants/register", body)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "VALIDATION_ERROR")
	})

	t.Run("GET /api/v1/platform/dashboard/metrics without token returns 401 Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/platform/dashboard/metrics", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "UNAUTHORIZED")
	})

	t.Run("GET /api/v1/platform/dashboard/metrics with tenant token returns 403 Forbidden", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/platform/dashboard/metrics", nil)
		req.Header.Set("Authorization", "Bearer "+tenantToken.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "FORBIDDEN")
	})

	t.Run("GET /api/v1/platform/subscriptions without token returns 401 Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/platform/subscriptions", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "UNAUTHORIZED")
	})

	t.Run("GET /api/v1/platform/subscriptions with tenant token returns 403 Forbidden", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/platform/subscriptions", nil)
		req.Header.Set("Authorization", "Bearer "+tenantToken.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "FORBIDDEN")
	})

	t.Run("GET /api/v1/platform/transactions without token returns 401 Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/platform/transactions", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "UNAUTHORIZED")
	})

	t.Run("GET /api/v1/platform/transactions with tenant token returns 403 Forbidden", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/platform/transactions", nil)
		req.Header.Set("Authorization", "Bearer "+tenantToken.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "FORBIDDEN")
	})

	t.Run("PUT /api/v1/platform/tenants/:id without token returns 401 Unauthorized", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name":"Updated Name","email":"updated@test.com"}`)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/platform/tenants/00000000-0000-0000-0000-000000000001", body)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "UNAUTHORIZED")
	})

	t.Run("PUT /api/v1/platform/tenants/:id with tenant token returns 403 Forbidden", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name":"Updated Name","email":"updated@test.com"}`)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/platform/tenants/00000000-0000-0000-0000-000000000001", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tenantToken.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "FORBIDDEN")
	})

	t.Run("PUT /api/v1/platform/tenants/:id with invalid UUID returns 400 Bad Request", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name":"Updated Name","email":"updated@test.com"}`)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/platform/tenants/invalid-uuid", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "BAD_REQUEST")
	})
}
