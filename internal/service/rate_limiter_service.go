package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/metrics"
	"github.com/Varunjp/vyavsa/internal/repository"
)

// RateLimitDecision represents the decision returned by RateLimiterService
type RateLimitDecision struct {
	Allowed    bool
	Current    int64
	Limit      int64
	Remaining  int64
	RetryAfter time.Duration
	ResetAt    time.Time
}

// RateLimiterService defines the rate limiting service contract
type RateLimiterService interface {
	Check(ctx context.Context, tier string, identifier string, limit int, window time.Duration, failOpen bool) (*RateLimitDecision, error)
	Reset(ctx context.Context, tier string, identifier string) error
	GetConfig() config.RateLimitConfig
}

type rateLimiterService struct {
	repo    repository.RateLimitRepository
	cfg     config.RateLimitConfig
	metrics *metrics.Metrics
	log     *slog.Logger
}

// NewRateLimiterService constructs a new RateLimiterService instance
func NewRateLimiterService(
	repo repository.RateLimitRepository,
	cfg config.RateLimitConfig,
	m *metrics.Metrics,
	log *slog.Logger,
) RateLimiterService {
	return &rateLimiterService{
		repo:    repo,
		cfg:     cfg,
		metrics: m,
		log:     log,
	}
}

// Check evaluates the rate limit for a specific tier and identifier
func (s *rateLimiterService) Check(ctx context.Context, tier string, identifier string, limit int, window time.Duration, failOpen bool) (*RateLimitDecision, error) {
	now := time.Now()

	// Short-circuit if rate limiting is globally disabled
	if !s.cfg.Enabled {
		return &RateLimitDecision{
			Allowed:   true,
			Current:   0,
			Limit:     int64(limit),
			Remaining: int64(limit),
			ResetAt:   now.Add(window),
		}, nil
	}

	key := fmt.Sprintf("ratelimit:%s:%s", tier, identifier)

	// If repository is nil (e.g. Redis unconfigured in test environment)
	if s.repo == nil {
		if failOpen {
			return &RateLimitDecision{
				Allowed:   true,
				Current:   0,
				Limit:     int64(limit),
				Remaining: int64(limit),
				ResetAt:   now.Add(window),
			}, nil
		}
		return nil, fmt.Errorf("rate limit repository is not available")
	}

	res, err := s.repo.Allow(ctx, key, limit, window)
	if err != nil {
		if s.metrics != nil {
			s.metrics.IncRateLimitRedisError(tier)
		}
		if s.log != nil {
			s.log.WarnContext(ctx, "rate limit redis evaluation error",
				slog.String("tier", tier),
				slog.Bool("fail_open", failOpen),
				slog.String("error", err.Error()),
			)
		}

		if failOpen {
			return &RateLimitDecision{
				Allowed:   true,
				Current:   0,
				Limit:     int64(limit),
				Remaining: int64(limit),
				ResetAt:   now.Add(window),
			}, nil
		}

		return nil, err
	}

	return &RateLimitDecision{
		Allowed:    res.Allowed,
		Current:    res.Current,
		Limit:      res.Limit,
		Remaining:  res.Remaining,
		RetryAfter: res.TTL,
		ResetAt:    res.ResetAt,
	}, nil
}

// Reset clears the rate limit state for a tier and identifier
func (s *rateLimiterService) Reset(ctx context.Context, tier string, identifier string) error {
	if s.repo == nil {
		return nil
	}
	key := fmt.Sprintf("ratelimit:%s:%s", tier, identifier)
	return s.repo.Reset(ctx, key)
}

// GetConfig returns the active rate limit configuration
func (s *rateLimiterService) GetConfig() config.RateLimitConfig {
	return s.cfg
}
