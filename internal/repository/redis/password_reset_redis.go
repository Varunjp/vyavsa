package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Varunjp/vyavsa/internal/cache"
	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/redis/go-redis/v9"
)

const (
	otpKeyPrefix      = "password_reset:otp:"
	tokenKeyPrefix    = "password_reset:token:"
	cooldownKeyPrefix = "password_reset:cooldown:"
)

// PasswordResetRedis implements repository.PasswordResetRepository
type PasswordResetRedis struct {
	cache *cache.Redis
}

// NewPasswordResetRedis creates a new PasswordResetRedis repository
func NewPasswordResetRedis(c *cache.Redis) *PasswordResetRedis {
	return &PasswordResetRedis{cache: c}
}

// CheckRateLimit checks and increments a rate-limit counter using fixed window expiration.
// Returns true if allowed, false if limit exceeded.
func (r *PasswordResetRedis) CheckRateLimit(ctx context.Context, key string, maxRequests int, window time.Duration) (bool, error) {
	if r.cache == nil || r.cache.Client == nil {
		return true, nil // Degraded mode: allow if Redis is disabled
	}

	count, err := r.cache.Client.Incr(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to increment rate limit counter: %w", err)
	}

	if count == 1 {
		if err := r.cache.Client.Expire(ctx, key, window).Err(); err != nil {
			return false, fmt.Errorf("failed to set rate limit expiration: %w", err)
		}
	}

	return count <= int64(maxRequests), nil
}

// CheckCooldown checks if a resend cooldown is active for the given email
func (r *PasswordResetRedis) CheckCooldown(ctx context.Context, email string) (bool, error) {
	if r.cache == nil || r.cache.Client == nil {
		return false, nil
	}

	key := cooldownKeyPrefix + email
	exists, err := r.cache.Client.Exists(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, fmt.Errorf("failed to check cooldown: %w", err)
	}

	return exists > 0, nil
}

// SetCooldown sets a cooldown timer for the given email
func (r *PasswordResetRedis) SetCooldown(ctx context.Context, email string, cooldown time.Duration) error {
	if r.cache == nil || r.cache.Client == nil {
		return nil
	}

	key := cooldownKeyPrefix + email
	err := r.cache.Client.Set(ctx, key, "1", cooldown).Err()
	if err != nil {
		return fmt.Errorf("failed to set cooldown: %w", err)
	}

	return nil
}

// StoreOTP stores an OTP record with TTL
func (r *PasswordResetRedis) StoreOTP(ctx context.Context, email string, otpData *domain.PasswordResetOTP, ttl time.Duration) error {
	if r.cache == nil || r.cache.Client == nil {
		return errors.New("redis cache is unavailable")
	}

	data, err := json.Marshal(otpData)
	if err != nil {
		return fmt.Errorf("failed to marshal otp data: %w", err)
	}

	key := otpKeyPrefix + email
	if err := r.cache.Client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store otp in redis: %w", err)
	}

	return nil
}

// GetOTP retrieves an active OTP record by email
func (r *PasswordResetRedis) GetOTP(ctx context.Context, email string) (*domain.PasswordResetOTP, error) {
	if r.cache == nil || r.cache.Client == nil {
		return nil, errors.New("redis cache is unavailable")
	}

	key := otpKeyPrefix + email
	data, err := r.cache.Client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to retrieve otp from redis: %w", err)
	}

	var otpData domain.PasswordResetOTP
	if err := json.Unmarshal(data, &otpData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal otp data: %w", err)
	}

	return &otpData, nil
}

// IncrementOTPAttempts increments the attempt counter for an OTP while preserving its remaining TTL
func (r *PasswordResetRedis) IncrementOTPAttempts(ctx context.Context, email string) (int, error) {
	if r.cache == nil || r.cache.Client == nil {
		return 0, errors.New("redis cache is unavailable")
	}

	key := otpKeyPrefix + email
	ttl, err := r.cache.Client.TTL(ctx, key).Result()
	if err != nil || ttl <= 0 {
		return 0, fmt.Errorf("otp expired or not found")
	}

	data, err := r.cache.Client.Get(ctx, key).Bytes()
	if err != nil {
		return 0, fmt.Errorf("failed to read otp data: %w", err)
	}

	var otpData domain.PasswordResetOTP
	if err := json.Unmarshal(data, &otpData); err != nil {
		return 0, fmt.Errorf("failed to unmarshal otp data: %w", err)
	}

	otpData.Attempts++

	updated, err := json.Marshal(&otpData)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal updated otp data: %w", err)
	}

	if err := r.cache.Client.Set(ctx, key, updated, ttl).Err(); err != nil {
		return 0, fmt.Errorf("failed to save updated otp data: %w", err)
	}

	return otpData.Attempts, nil
}

// DeleteOTP invalidates and removes an OTP record
func (r *PasswordResetRedis) DeleteOTP(ctx context.Context, email string) error {
	if r.cache == nil || r.cache.Client == nil {
		return nil
	}

	key := otpKeyPrefix + email
	if err := r.cache.Client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete otp from redis: %w", err)
	}

	return nil
}

// StoreResetToken stores a password reset token record with TTL
func (r *PasswordResetRedis) StoreResetToken(ctx context.Context, tokenID string, tokenData *domain.PasswordResetTokenData, ttl time.Duration) error {
	if r.cache == nil || r.cache.Client == nil {
		return errors.New("redis cache is unavailable")
	}

	data, err := json.Marshal(tokenData)
	if err != nil {
		return fmt.Errorf("failed to marshal reset token data: %w", err)
	}

	key := tokenKeyPrefix + tokenID
	if err := r.cache.Client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store reset token in redis: %w", err)
	}

	return nil
}

// GetResetToken retrieves a stored reset token record by tokenID
func (r *PasswordResetRedis) GetResetToken(ctx context.Context, tokenID string) (*domain.PasswordResetTokenData, error) {
	if r.cache == nil || r.cache.Client == nil {
		return nil, errors.New("redis cache is unavailable")
	}

	key := tokenKeyPrefix + tokenID
	data, err := r.cache.Client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to retrieve reset token from redis: %w", err)
	}

	var tokenData domain.PasswordResetTokenData
	if err := json.Unmarshal(data, &tokenData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reset token data: %w", err)
	}

	return &tokenData, nil
}

// DeleteResetToken removes a reset token, ensuring single-use consumption
func (r *PasswordResetRedis) DeleteResetToken(ctx context.Context, tokenID string) error {
	if r.cache == nil || r.cache.Client == nil {
		return nil
	}

	key := tokenKeyPrefix + tokenID
	if err := r.cache.Client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete reset token from redis: %w", err)
	}

	return nil
}
