package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// TokenBlacklistRepository handles invalidation and revocation of JWT tokens
type TokenBlacklistRepository interface {
	RevokeToken(ctx context.Context, tokenID string, ttl time.Duration) error
	IsTokenRevoked(ctx context.Context, tokenID string) (bool, error)
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID, ttl time.Duration) error
	IsUserTokenRevoked(ctx context.Context, userID uuid.UUID, issuedAt time.Time) (bool, error)
}
