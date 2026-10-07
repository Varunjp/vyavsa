package repository

import (
	"context"
	"time"
)

// RateLimitResult contains the outcome of an atomic rate limit check
type RateLimitResult struct {
	Allowed   bool          // Whether the request is permitted
	Current   int64         // Current request count in the window
	Limit     int64         // Maximum permitted requests
	Remaining int64         // Remaining permitted requests
	TTL       time.Duration // Remaining TTL duration until window resets
	ResetAt   time.Time     // Absolute time when window resets
}

// RateLimitRepository defines the interface for distributed Redis-backed rate limiting persistence
type RateLimitRepository interface {
	// Allow evaluates and increments the rate limit counter for a specific key atomically.
	// Returns the rate limit evaluation result or an error if Redis is unreachable.
	Allow(ctx context.Context, key string, limit int, window time.Duration) (*RateLimitResult, error)

	// Reset deletes the rate limit key (useful for tests and manual operational resets)
	Reset(ctx context.Context, key string) error
}
