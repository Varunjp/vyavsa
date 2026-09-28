package repository

import (
	"context"
	"time"
)

// TokenBlacklistRepository handles invalidation and revocation of JWT tokens
type TokenBlacklistRepository interface {
	RevokeToken(ctx context.Context, tokenID string, ttl time.Duration) error
	IsTokenRevoked(ctx context.Context, tokenID string) (bool, error)
}
