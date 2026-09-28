package auth

import (
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTManager(t *testing.T) {
	cfg := config.JWTConfig{
		Secret:        "test-secret-key-at-least-32-characters-long",
		AccessExpiry:  1 * time.Minute,
		RefreshExpiry: 5 * time.Minute,
	}

	mgr := NewJWTManager(cfg)
	userID := uuid.New()
	tenantID := uuid.New()

	t.Run("Generates and validates token pair", func(t *testing.T) {
		tokens, err := mgr.GenerateTokenPair(userID, &tenantID, "user@example.com", RoleTenantAdmin, UserTypeTenantUser)
		require.NoError(t, err)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
		assert.Equal(t, "Bearer", tokens.TokenType)
		assert.Equal(t, int64(60), tokens.ExpiresIn)

		claims, err := mgr.ValidateToken(tokens.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, &tenantID, claims.TenantID)
		assert.Equal(t, "user@example.com", claims.Email)
		assert.Equal(t, RoleTenantAdmin, claims.Role)
		assert.Equal(t, UserTypeTenantUser, claims.UserType)
	})

	t.Run("Rejects expired token", func(t *testing.T) {
		expiredCfg := config.JWTConfig{
			Secret:        cfg.Secret,
			AccessExpiry:  -1 * time.Minute, // already expired
			RefreshExpiry: -1 * time.Minute,
		}
		expiredMgr := NewJWTManager(expiredCfg)
		tokens, err := expiredMgr.GenerateTokenPair(userID, nil, "admin@platform.com", RolePlatformAdmin, UserTypePlatformAdmin)
		require.NoError(t, err)

		_, err = mgr.ValidateToken(tokens.AccessToken)
		assert.ErrorIs(t, err, ErrExpiredToken)
	})

	t.Run("Rejects token signed with wrong secret", func(t *testing.T) {
		otherMgr := NewJWTManager(config.JWTConfig{
			Secret:        "different-secret-key-minimum-32-bytes-long",
			AccessExpiry:  1 * time.Minute,
			RefreshExpiry: 5 * time.Minute,
		})

		tokens, err := otherMgr.GenerateTokenPair(userID, nil, "admin@platform.com", RolePlatformAdmin, UserTypePlatformAdmin)
		require.NoError(t, err)

		_, err = mgr.ValidateToken(tokens.AccessToken)
		assert.ErrorIs(t, err, ErrInvalidToken)
	})
}
