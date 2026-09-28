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

func TestAuthPipelineIntegration(t *testing.T) {
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
	userID := uuid.New()
	tenantID := uuid.New()

	t.Run("POST /api/v1/auth/platform/login with invalid payload returns 422 Validation Error", func(t *testing.T) {
		body := bytes.NewBufferString(`{"identifier": ""}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/platform/login", body)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "VALIDATION_ERROR")
	})

	t.Run("POST /api/v1/auth/tenant/login with invalid payload returns 422 Validation Error", func(t *testing.T) {
		body := bytes.NewBufferString(`{"email": "not-an-email"}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/tenant/login", body)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "VALIDATION_ERROR")
	})

	t.Run("GET /api/v1/auth/me without token returns 401 Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "UNAUTHORIZED")
	})

	t.Run("GET /api/v1/platform/ping with platform admin token returns 200 OK", func(t *testing.T) {
		tokens, err := jwtManager.GenerateTokenPair(userID, nil, "admin@vyavsa.com", auth.RolePlatformAdmin, auth.UserTypePlatformAdmin)
		require.NoError(t, err)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/platform/ping", nil)
		req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"success":true`)
		assert.Contains(t, w.Body.String(), "platform admin authenticated")
	})

	t.Run("GET /api/v1/platform/ping with tenant user token returns 403 Forbidden", func(t *testing.T) {
		tokens, err := jwtManager.GenerateTokenPair(userID, &tenantID, "user@store.com", auth.RoleTenantAdmin, auth.UserTypeTenantUser)
		require.NoError(t, err)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/platform/ping", nil)
		req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "FORBIDDEN")
	})

	t.Run("GET /api/v1/tenant/ping with tenant user token returns 200 OK", func(t *testing.T) {
		tokens, err := jwtManager.GenerateTokenPair(userID, &tenantID, "user@store.com", auth.RoleTenantAdmin, auth.UserTypeTenantUser)
		require.NoError(t, err)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/ping", nil)
		req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"success":true`)
		assert.Contains(t, w.Body.String(), "tenant user authenticated")
	})

	t.Run("POST /api/v1/auth/logout with valid token returns 200 OK", func(t *testing.T) {
		tokens, err := jwtManager.GenerateTokenPair(userID, &tenantID, "user@store.com", auth.RoleTenantAdmin, auth.UserTypeTenantUser)
		require.NoError(t, err)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))
	})
}
