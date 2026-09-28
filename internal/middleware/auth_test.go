package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dummyBlacklist struct {
	revoked map[string]bool
}

func (d *dummyBlacklist) RevokeToken(ctx context.Context, tokenID string, ttl time.Duration) error {
	d.revoked[tokenID] = true
	return nil
}

func (d *dummyBlacklist) IsTokenRevoked(ctx context.Context, tokenID string) (bool, error) {
	return d.revoked[tokenID], nil
}

func TestAuthAndRBACMiddleware(t *testing.T) {
	jwtManager := auth.NewJWTManager(config.JWTConfig{
		Secret:        "test-secret-must-be-at-least-32-bytes-long",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 24 * time.Hour,
	})
	blacklist := &dummyBlacklist{revoked: make(map[string]bool)}

	r := gin.New()
	api := r.Group("/test")
	api.Use(Authenticate(jwtManager, blacklist))
	{
		api.GET("/protected", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		platform := api.Group("/platform")
		platform.Use(RequirePlatformAdmin())
		platform.GET("/dashboard", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		tenant := api.Group("/tenant")
		tenant.Use(RequireRole(auth.RoleTenantAdmin))
		tenant.GET("/settings", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
	}

	userID := uuid.New()
	tenantID := uuid.New()

	t.Run("Rejects missing Authorization header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test/protected", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "UNAUTHORIZED")
	})

	t.Run("Rejects invalid Bearer format", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test/protected", nil)
		req.Header.Set("Authorization", "Basic 12345")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Allows valid tenant user token on protected endpoint", func(t *testing.T) {
		tokens, err := jwtManager.GenerateTokenPair(userID, &tenantID, "user@tenant.com", auth.RoleTenantUser, auth.UserTypeTenantUser)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Blocks tenant user from platform admin endpoint with 403 Forbidden", func(t *testing.T) {
		tokens, err := jwtManager.GenerateTokenPair(userID, &tenantID, "user@tenant.com", auth.RoleTenantUser, auth.UserTypeTenantUser)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test/platform/dashboard", nil)
		req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "FORBIDDEN")
	})

	t.Run("Allows platform admin on platform admin endpoint", func(t *testing.T) {
		adminTokens, err := jwtManager.GenerateTokenPair(userID, nil, "admin@platform.com", auth.RolePlatformAdmin, auth.UserTypePlatformAdmin)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test/platform/dashboard", nil)
		req.Header.Set("Authorization", "Bearer "+adminTokens.AccessToken)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Enforces role requirement", func(t *testing.T) {
		// Normal tenant user (not admin) attempting admin settings
		userTokens, err := jwtManager.GenerateTokenPair(userID, &tenantID, "user@tenant.com", auth.RoleTenantUser, auth.UserTypeTenantUser)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test/tenant/settings", nil)
		req.Header.Set("Authorization", "Bearer "+userTokens.AccessToken)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		// Tenant admin attempting admin settings
		adminTokens, err := jwtManager.GenerateTokenPair(userID, &tenantID, "admin@tenant.com", auth.RoleTenantAdmin, auth.UserTypeTenantUser)
		require.NoError(t, err)

		w2 := httptest.NewRecorder()
		req2, _ := http.NewRequest(http.MethodGet, "/test/tenant/settings", nil)
		req2.Header.Set("Authorization", "Bearer "+adminTokens.AccessToken)
		r.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusOK, w2.Code)
	})
}
