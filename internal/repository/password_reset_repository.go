package repository

import (
	"context"
	"time"

	"github.com/Varunjp/vyavsa/internal/domain"
)

// PasswordResetRepository defines caching, state management, and rate-limiting contracts for password recovery
type PasswordResetRepository interface {
	// Rate Limiting & Cooldowns
	CheckRateLimit(ctx context.Context, key string, maxRequests int, window time.Duration) (bool, error)
	CheckCooldown(ctx context.Context, email string) (bool, error)
	SetCooldown(ctx context.Context, email string, cooldown time.Duration) error

	// OTP Management
	StoreOTP(ctx context.Context, email string, otpData *domain.PasswordResetOTP, ttl time.Duration) error
	GetOTP(ctx context.Context, email string) (*domain.PasswordResetOTP, error)
	IncrementOTPAttempts(ctx context.Context, email string) (int, error)
	DeleteOTP(ctx context.Context, email string) error

	// Reset Token Management
	StoreResetToken(ctx context.Context, tokenID string, tokenData *domain.PasswordResetTokenData, ttl time.Duration) error
	GetResetToken(ctx context.Context, tokenID string) (*domain.PasswordResetTokenData, error)
	DeleteResetToken(ctx context.Context, tokenID string) error
}
