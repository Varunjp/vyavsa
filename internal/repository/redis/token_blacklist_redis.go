package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Varunjp/vyavsa/internal/cache"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	blacklistKeyPrefix      = "token:revoked:"
	userRevokedBeforePrefix = "user:revoked_before:"
)

// TokenBlacklistRedis implements repository.TokenBlacklistRepository
type TokenBlacklistRedis struct {
	cache *cache.Redis
}

// NewTokenBlacklistRedis creates a new TokenBlacklistRedis repository
func NewTokenBlacklistRedis(c *cache.Redis) *TokenBlacklistRedis {
	return &TokenBlacklistRedis{cache: c}
}

// RevokeToken marks a JWT token ID as revoked in Redis with an expiration TTL
func (r *TokenBlacklistRedis) RevokeToken(ctx context.Context, tokenID string, ttl time.Duration) error {
	if r.cache == nil || r.cache.Client == nil {
		return nil // Degraded mode: non-blocking if Redis is disabled
	}

	key := blacklistKeyPrefix + tokenID
	err := r.cache.Client.Set(ctx, key, "1", ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to blacklist token in redis: %w", err)
	}

	return nil
}

// IsTokenRevoked checks whether a JWT token ID is blacklisted
func (r *TokenBlacklistRedis) IsTokenRevoked(ctx context.Context, tokenID string) (bool, error) {
	if r.cache == nil || r.cache.Client == nil {
		return false, nil // Degraded mode
	}

	key := blacklistKeyPrefix + tokenID
	exists, err := r.cache.Client.Exists(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		// If Redis is unreachable, fail safe by logging and allowing request to avoid complete outage
		return false, fmt.Errorf("redis check failed: %w", err)
	}

	return exists > 0, nil
}

// RevokeAllUserTokens records a revocation timestamp for the user, invalidating any tokens issued prior
func (r *TokenBlacklistRedis) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID, ttl time.Duration) error {
	if r.cache == nil || r.cache.Client == nil {
		return nil
	}

	key := userRevokedBeforePrefix + userID.String()
	nowUnix := strconv.FormatInt(time.Now().Unix(), 10)
	err := r.cache.Client.Set(ctx, key, nowUnix, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set user revocation timestamp in redis: %w", err)
	}

	return nil
}

// IsUserTokenRevoked checks whether a token's issued-at timestamp is before the user's latest revocation timestamp
func (r *TokenBlacklistRedis) IsUserTokenRevoked(ctx context.Context, userID uuid.UUID, issuedAt time.Time) (bool, error) {
	if r.cache == nil || r.cache.Client == nil {
		return false, nil
	}

	key := userRevokedBeforePrefix + userID.String()
	val, err := r.cache.Client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, fmt.Errorf("redis check failed: %w", err)
	}

	revokedBeforeUnix, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return false, nil
	}

	return issuedAt.Unix() <= revokedBeforeUnix, nil
}
