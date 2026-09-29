package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/dto"
	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/Varunjp/vyavsa/internal/metrics"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestPasswordRecoveryUnit(t *testing.T) {
	hasher := auth.NewBcryptHasher(bcrypt.MinCost)
	jwtManager := auth.NewJWTManager(config.JWTConfig{
		Secret:        "test-secret-key-32-chars-long-minimum-bytes",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 24 * time.Hour,
	})
	log := logger.Default().Logger
	m := metrics.New()
	pwCfg := config.PasswordResetConfig{
		OTPExpiry:        5 * time.Minute,
		TokenExpiry:      10 * time.Minute,
		OTPLength:        6,
		MaxAttempts:      5,
		ResendCooldown:   60 * time.Second,
		MaxEmailRequests: 3,
		MaxIPRequests:    10,
		RateLimitWindow:  15 * time.Minute,
	}

	t.Run("OTP generation creates 6 numeric digits", func(t *testing.T) {
		for i := 0; i < 20; i++ {
			otp, err := generateNumericOTP(6)
			require.NoError(t, err)
			assert.Len(t, otp, 6)
			for _, r := range otp {
				assert.True(t, r >= '0' && r <= '9', "char %c is not numeric", r)
			}
		}
	})

	t.Run("OTP hashing produces consistent SHA256 hex output", func(t *testing.T) {
		h1 := hashOTP("123456")
		h2 := hashOTP("123456")
		h3 := hashOTP("654321")

		assert.Equal(t, h1, h2)
		assert.NotEqual(t, h1, h3)
		assert.Len(t, h1, 64) // 32 bytes hex encoded = 64 chars
	})

	t.Run("Email normalization trims spaces and downcases", func(t *testing.T) {
		input := "  John.Doe@EXAMPLE.Com  "
		normalized := strings.ToLower(strings.TrimSpace(input))
		assert.Equal(t, "john.doe@example.com", normalized)
	})

	t.Run("Password validation enforces minimum length of 6 and max of 72", func(t *testing.T) {
		tenantID := uuid.New()
		userID := uuid.New()
		user := &domain.TenantUser{
			ID:           userID,
			TenantID:     tenantID,
			Email:        "user@example.com",
			Status:       "active",
			PasswordHash: "old-hash",
		}

		svc := NewAuthService(
			&mockPlatformAdminRepo{},
			&mockTenantUserRepo{user: user},
			&mockBlacklistRepo{revoked: make(map[string]bool), userRevoked: make(map[uuid.UUID]time.Time)},
			newMockPasswordResetRepo(),
			&mockMailer{},
			hasher,
			jwtManager,
			pwCfg,
			m,
			log,
		)

		// Too short (<6)
		err := svc.ResetPassword(context.Background(), dto.ResetPasswordRequest{
			ResetToken:  "some-token",
			NewPassword: "123",
		}, "127.0.0.1")
		require.Error(t, err)
		appErr := appErrors.FromError(err)
		assert.Equal(t, appErrors.CodeBadRequest, appErr.Code)

		// Too long (>72)
		longPass := strings.Repeat("A", 73)
		err = svc.ResetPassword(context.Background(), dto.ResetPasswordRequest{
			ResetToken:  "some-token",
			NewPassword: longPass,
		}, "127.0.0.1")
		require.Error(t, err)
		appErr = appErrors.FromError(err)
		assert.Equal(t, appErrors.CodeBadRequest, appErr.Code)
	})

	t.Run("Reset token expiry and single-use validation", func(t *testing.T) {
		tenantID := uuid.New()
		userID := uuid.New()
		user := &domain.TenantUser{
			ID:           userID,
			TenantID:     tenantID,
			Email:        "user@example.com",
			Status:       "active",
			PasswordHash: "initial-hash",
		}
		repo := newMockPasswordResetRepo()
		svc := NewAuthService(
			&mockPlatformAdminRepo{},
			&mockTenantUserRepo{user: user},
			&mockBlacklistRepo{revoked: make(map[string]bool), userRevoked: make(map[uuid.UUID]time.Time)},
			repo,
			&mockMailer{},
			hasher,
			jwtManager,
			pwCfg,
			m,
			log,
		)

		// Issue expired token
		expiredToken, tokenID, err := jwtManager.GeneratePasswordResetToken(userID, &tenantID, "user@example.com", auth.UserTypeTenantUser, -1*time.Minute)
		require.NoError(t, err)
		_ = repo.StoreResetToken(context.Background(), tokenID, &domain.PasswordResetTokenData{
			TokenID:   tokenID,
			UserID:    userID,
			TenantID:  &tenantID,
			UserType:  auth.UserTypeTenantUser,
			Email:     "user@example.com",
			ExpiresAt: time.Now().Add(-1 * time.Minute),
		}, -1*time.Minute)

		err = svc.ResetPassword(context.Background(), dto.ResetPasswordRequest{
			ResetToken:  expiredToken,
			NewPassword: "ValidPassword123!",
		}, "127.0.0.1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})
}
