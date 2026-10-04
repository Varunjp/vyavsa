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

	t.Run("POST /api/v1/auth/refresh with valid refresh token returns 200 and new tokens", func(t *testing.T) {
		tokens, err := jwtManager.GenerateTokenPair(userID, &tenantID, "user@store.com", auth.RoleTenantAdmin, auth.UserTypeTenantUser)
		require.NoError(t, err)

		body, _ := json.Marshal(map[string]string{
			"refresh_token": tokens.RefreshToken,
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
				ExpiresIn    int64  `json:"expires_in"`
			} `json:"data"`
		}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Data.AccessToken)
		assert.NotEmpty(t, resp.Data.RefreshToken)
		assert.NotEqual(t, tokens.AccessToken, resp.Data.AccessToken)

		// Test that new access token works on protected route
		reqNew, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/ping", nil)
		reqNew.Header.Set("Authorization", "Bearer "+resp.Data.AccessToken)
		wNew := httptest.NewRecorder()
		router.ServeHTTP(wNew, reqNew)
		assert.Equal(t, http.StatusOK, wNew.Code)
	})

	t.Run("POST /api/v1/auth/refresh with expired refresh token returns 401 Unauthorized", func(t *testing.T) {
		expiredCfg := config.JWTConfig{
			Secret:        cfg.JWT.Secret,
			AccessExpiry:  -1 * time.Minute,
			RefreshExpiry: -1 * time.Minute,
		}
		expiredMgr := auth.NewJWTManager(expiredCfg)
		tokens, err := expiredMgr.GenerateTokenPair(userID, &tenantID, "user@store.com", auth.RoleTenantAdmin, auth.UserTypeTenantUser)
		require.NoError(t, err)

		body, _ := json.Marshal(map[string]string{
			"refresh_token": tokens.RefreshToken,
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "UNAUTHORIZED")
	})

	t.Run("POST /api/v1/auth/refresh with invalid token returns 401 Unauthorized", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"refresh_token": "completely-invalid-malformed-jwt",
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "UNAUTHORIZED")
	})

	t.Run("POST /api/v1/auth/refresh rejecting access token returns 401 Unauthorized", func(t *testing.T) {
		tokens, err := jwtManager.GenerateTokenPair(userID, &tenantID, "user@store.com", auth.RoleTenantAdmin, auth.UserTypeTenantUser)
		require.NoError(t, err)

		body, _ := json.Marshal(map[string]string{
			"refresh_token": tokens.AccessToken, // Passing access token instead of refresh token
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "provided token is not a refresh token")
	})

	t.Run("GET /api/v1/tenant/ping rejecting refresh token returns 401 Unauthorized", func(t *testing.T) {
		tokens, err := jwtManager.GenerateTokenPair(userID, &tenantID, "user@store.com", auth.RoleTenantAdmin, auth.UserTypeTenantUser)
		require.NoError(t, err)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/ping", nil)
		req.Header.Set("Authorization", "Bearer "+tokens.RefreshToken) // Passing refresh token as bearer

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "provided token is not an access token")
	})

	t.Run("POST /auth/refresh on root router works and returns 200 OK", func(t *testing.T) {
		tokens, err := jwtManager.GenerateTokenPair(userID, &tenantID, "user@store.com", auth.RoleTenantAdmin, auth.UserTypeTenantUser)
		require.NoError(t, err)

		body, _ := json.Marshal(map[string]string{
			"refresh_token": tokens.RefreshToken,
		})
		req, _ := http.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"success":true`)
	})

	t.Run("POST /auth/revoke on root router works and returns 200 OK", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"refresh_token": "some-token",
		})
		req, _ := http.NewRequest(http.MethodPost, "/auth/revoke", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"success":true`)
	})

	t.Run("End-to-End expired access token recovery flow", func(t *testing.T) {
		// 1. Initial valid refresh token, expired access token
		expiredCfg := config.JWTConfig{
			Secret:        cfg.JWT.Secret,
			AccessExpiry:  -1 * time.Minute, // Expired
			RefreshExpiry: 24 * time.Hour,   // Still valid
		}
		mgrForSetup := auth.NewJWTManager(expiredCfg)
		tokens, err := mgrForSetup.GenerateTokenPair(userID, &tenantID, "user@store.com", auth.RoleTenantAdmin, auth.UserTypeTenantUser)
		require.NoError(t, err)

		// 2. Request with expired access token fails with 401
		reqExpired, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/ping", nil)
		reqExpired.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
		wExpired := httptest.NewRecorder()
		router.ServeHTTP(wExpired, reqExpired)
		assert.Equal(t, http.StatusUnauthorized, wExpired.Code)

		// 3. Frontend intercepts 401, calls refresh endpoint
		refBody, _ := json.Marshal(map[string]string{
			"refresh_token": tokens.RefreshToken,
		})
		reqRef, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(refBody))
		reqRef.Header.Set("Content-Type", "application/json")
		wRef := httptest.NewRecorder()
		router.ServeHTTP(wRef, reqRef)
		assert.Equal(t, http.StatusOK, wRef.Code)

		var refResp struct {
			Data struct {
				AccessToken string `json:"access_token"`
			} `json:"data"`
		}
		err = json.Unmarshal(wRef.Body.Bytes(), &refResp)
		require.NoError(t, err)
		assert.NotEmpty(t, refResp.Data.AccessToken)

		// 4. Frontend retries original request with new access token and succeeds
		reqRetry, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/ping", nil)
		reqRetry.Header.Set("Authorization", "Bearer "+refResp.Data.AccessToken)
		wRetry := httptest.NewRecorder()
		router.ServeHTTP(wRetry, reqRetry)
		assert.Equal(t, http.StatusOK, wRetry.Code)
		assert.Contains(t, wRetry.Body.String(), `"success":true`)
	})

	t.Run("Concurrent requests with expired access token and single refresh retry", func(t *testing.T) {
		expiredCfg := config.JWTConfig{
			Secret:        cfg.JWT.Secret,
			AccessExpiry:  -1 * time.Minute,
			RefreshExpiry: 24 * time.Hour,
		}
		mgrForSetup := auth.NewJWTManager(expiredCfg)
		tokens, err := mgrForSetup.GenerateTokenPair(userID, &tenantID, "user@store.com", auth.RoleTenantAdmin, auth.UserTypeTenantUser)
		require.NoError(t, err)

		// 10 concurrent requests with expired token all return 401
		const numRequests = 10
		results := make(chan int, numRequests)
		for i := 0; i < numRequests; i++ {
			go func() {
				req, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/ping", nil)
				req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				results <- w.Code
			}()
		}

		for i := 0; i < numRequests; i++ {
			code := <-results
			assert.Equal(t, http.StatusUnauthorized, code)
		}

		// Single-flight refresh executes once
		refBody, _ := json.Marshal(map[string]string{
			"refresh_token": tokens.RefreshToken,
		})
		reqRef, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(refBody))
		reqRef.Header.Set("Content-Type", "application/json")
		wRef := httptest.NewRecorder()
		router.ServeHTTP(wRef, reqRef)
		assert.Equal(t, http.StatusOK, wRef.Code)

		var refResp struct {
			Data struct {
				AccessToken string `json:"access_token"`
			} `json:"data"`
		}
		err = json.Unmarshal(wRef.Body.Bytes(), &refResp)
		require.NoError(t, err)
		assert.NotEmpty(t, refResp.Data.AccessToken)

		// All 10 pending requests retry concurrently with new access token and succeed
		retryResults := make(chan int, numRequests)
		for i := 0; i < numRequests; i++ {
			go func() {
				req, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/ping", nil)
				req.Header.Set("Authorization", "Bearer "+refResp.Data.AccessToken)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				retryResults <- w.Code
			}()
		}

		for i := 0; i < numRequests; i++ {
			code := <-retryResults
			assert.Equal(t, http.StatusOK, code)
		}
	})
}
