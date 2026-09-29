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

func TestTenantOperations_RBACAndSecurity(t *testing.T) {
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
	tenantAdminID := uuid.New()
	tenantUserID := uuid.New()
	platformAdminID := uuid.New()

	// 1. Generate Tokens
	adminToken, err := jwtManager.GenerateTokenPair(tenantAdminID, &tenantID, "admin@business.com", auth.RoleTenantAdmin, auth.UserTypeTenantUser)
	require.NoError(t, err)

	userToken, err := jwtManager.GenerateTokenPair(tenantUserID, &tenantID, "staff@business.com", auth.RoleTenantUser, auth.UserTypeTenantUser)
	require.NoError(t, err)

	platToken, err := jwtManager.GenerateTokenPair(platformAdminID, nil, "plat@vyavsa.com", auth.RolePlatformAdmin, auth.UserTypePlatformAdmin)
	require.NoError(t, err)

	// Admin-only endpoints to verify RBAC
	adminOnlyEndpoints := []struct {
		method string
		url    string
		body   string
	}{
		{http.MethodGet, "/api/v1/tenant/users", ""},
		{http.MethodPost, "/api/v1/tenant/users", `{"name":"A","email":"a@b.com","password":"pass","role":"user"}`},
		{http.MethodGet, "/api/v1/tenant/employees", ""},
		{http.MethodPost, "/api/v1/tenant/employees", `{"name":"A","salary":100}`},
		{http.MethodGet, "/api/v1/tenant/customers", ""},
		{http.MethodPost, "/api/v1/tenant/customers", `{"customer_name":"A"}`},
		{http.MethodGet, "/api/v1/tenant/banks", ""},
		{http.MethodPost, "/api/v1/tenant/banks", `{"bank_name":"A"}`},
		{http.MethodGet, "/api/v1/tenant/salaries", ""},
		{http.MethodGet, "/api/v1/tenant/salaries/pending", ""},
	}

	for _, ep := range adminOnlyEndpoints {
		t.Run("Unauthenticated "+ep.method+" "+ep.url+" returns 401", func(t *testing.T) {
			var body *bytes.Buffer
			if ep.body != "" {
				body = bytes.NewBufferString(ep.body)
			} else {
				body = bytes.NewBuffer(nil)
			}
			req, _ := http.NewRequest(ep.method, ep.url, body)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})

		t.Run("Tenant User "+ep.method+" "+ep.url+" returns 403 Forbidden", func(t *testing.T) {
			var body *bytes.Buffer
			if ep.body != "" {
				body = bytes.NewBufferString(ep.body)
			} else {
				body = bytes.NewBuffer(nil)
			}
			req, _ := http.NewRequest(ep.method, ep.url, body)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+userToken.AccessToken)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusForbidden, w.Code)
			assert.Contains(t, w.Body.String(), "FORBIDDEN")
		})

		t.Run("Platform Admin "+ep.method+" "+ep.url+" returns 403 (requires tenant context)", func(t *testing.T) {
			var body *bytes.Buffer
			if ep.body != "" {
				body = bytes.NewBufferString(ep.body)
			} else {
				body = bytes.NewBuffer(nil)
			}
			req, _ := http.NewRequest(ep.method, ep.url, body)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+platToken.AccessToken)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusForbidden, w.Code)
		})
	}

	// Shared operations accessible by Tenant User
	sharedEndpoints := []struct {
		method string
		url    string
	}{
		{http.MethodGet, "/api/v1/tenant/line-sales"},
		{http.MethodGet, "/api/v1/tenant/counter-sales"},
		{http.MethodGet, "/api/v1/tenant/attendance"},
		{http.MethodGet, "/api/v1/tenant/purchases"},
		{http.MethodGet, "/api/v1/tenant/expenses"},
		{http.MethodGet, "/api/v1/tenant/daily-stats"},
		{http.MethodGet, "/api/v1/tenant/dashboard"},
	}

	for _, ep := range sharedEndpoints {
		t.Run("Tenant User can access shared "+ep.method+" "+ep.url, func(t *testing.T) {
			req, _ := http.NewRequest(ep.method, ep.url, nil)
			req.Header.Set("Authorization", "Bearer "+userToken.AccessToken)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			// Because mock DB pool is nil in unit router test, handler returns 200 or handles db nil gracefully
			// Crucially: it is NOT 401 or 403!
			assert.NotEqual(t, http.StatusUnauthorized, w.Code)
			assert.NotEqual(t, http.StatusForbidden, w.Code)
		})

		t.Run("Tenant Admin can access shared "+ep.method+" "+ep.url, func(t *testing.T) {
			req, _ := http.NewRequest(ep.method, ep.url, nil)
			req.Header.Set("Authorization", "Bearer "+adminToken.AccessToken)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.NotEqual(t, http.StatusUnauthorized, w.Code)
			assert.NotEqual(t, http.StatusForbidden, w.Code)
		})
	}
}
