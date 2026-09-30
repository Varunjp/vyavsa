package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/metrics"
	"github.com/Varunjp/vyavsa/internal/repository"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
)

const (
	// TenantPlanCacheTTL defines the duration tenant plan statuses remain cached in Redis (24 hours)
	TenantPlanCacheTTL = 24 * time.Hour
)

// TenantPlanService defines contracts for tenant subscription and plan validation
type TenantPlanService interface {
	HasActivePlan(ctx context.Context, tenantID uuid.UUID) (bool, error)
	InvalidateTenantPlanCache(ctx context.Context, tenantID uuid.UUID) error
}

type tenantPlanService struct {
	subscriptionRepo repository.PlatformSubscriptionRepository
	tenantRepo       repository.TenantRepository
	planCache        repository.TenantPlanCacheRepository
	metrics          *metrics.Metrics
	log              *slog.Logger
}

// NewTenantPlanService creates a new TenantPlanService instance
func NewTenantPlanService(
	subscriptionRepo repository.PlatformSubscriptionRepository,
	tenantRepo repository.TenantRepository,
	planCache repository.TenantPlanCacheRepository,
	m *metrics.Metrics,
	log *slog.Logger,
) TenantPlanService {
	return &tenantPlanService{
		subscriptionRepo: subscriptionRepo,
		tenantRepo:       tenantRepo,
		planCache:        planCache,
		metrics:          m,
		log:              log,
	}
}

// HasActivePlan verifies if the tenant currently possesses an active, unexpired subscription plan.
// It checks Redis first; on miss or Redis unavailability, it safely falls back to PostgreSQL
// and populates Redis with a 24-hour TTL.
func (s *tenantPlanService) HasActivePlan(ctx context.Context, tenantID uuid.UUID) (bool, error) {
	if tenantID == uuid.Nil {
		return false, appErrors.NewForbidden("invalid tenant identity")
	}

	// 1. Check Redis Cache
	var cached *domain.CachedTenantPlan
	if s.planCache != nil {
		c, err := s.planCache.Get(ctx, tenantID)
		if err != nil {
			// Redis failure: log warning and proceed to PostgreSQL fallback (do not bypass security)
			s.log.WarnContext(ctx, "tenant plan redis cache unavailable, falling back to database",
				slog.String("tenant_id", tenantID.String()),
				slog.String("error", err.Error()),
			)
		} else {
			cached = c
		}
	}

	// 2. Cache Hit Evaluation
	if cached != nil {
		s.log.InfoContext(ctx, "tenant_plan_cache_hit",
			slog.String("tenant_id", tenantID.String()),
			slog.Bool("active", cached.Active),
			slog.String("status", cached.Status),
		)
		if s.metrics != nil {
			s.metrics.IncTenantPlanCacheHits()
		}

		if !cached.Active {
			s.log.WarnContext(ctx, "tenant_plan_validation_failed",
				slog.String("tenant_id", tenantID.String()),
				slog.String("reason", "cached plan is inactive or expired"),
				slog.String("status", cached.Status),
			)
			if s.metrics != nil {
				s.metrics.IncTenantPlanValidationFailures()
			}
			return false, nil
		}

		return true, nil
	}

	// 3. Cache Miss Evaluation -> Query PostgreSQL
	s.log.InfoContext(ctx, "tenant_plan_cache_miss",
		slog.String("tenant_id", tenantID.String()),
	)
	if s.metrics != nil {
		s.metrics.IncTenantPlanCacheMisses()
	}

	if s.subscriptionRepo == nil {
		s.log.WarnContext(ctx, "tenant_plan_validation_failed",
			slog.String("tenant_id", tenantID.String()),
			slog.String("reason", "subscription repository not configured"),
		)
		if s.metrics != nil {
			s.metrics.IncTenantPlanValidationFailures()
		}
		return false, nil
	}

	sub, err := s.subscriptionRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		if appErrors.IsNotFound(err) {
			// Tenant has no subscription recorded
			s.log.WarnContext(ctx, "tenant_plan_validation_failed",
				slog.String("tenant_id", tenantID.String()),
				slog.String("reason", "no subscription found"),
			)
			if s.metrics != nil {
				s.metrics.IncTenantPlanValidationFailures()
			}

			// Cache negative result in Redis so subsequent requests avoid querying PostgreSQL
			if s.planCache != nil {
				_ = s.planCache.Set(ctx, tenantID, &domain.CachedTenantPlan{
					Active: false,
					Status: "none",
				}, TenantPlanCacheTTL)
			}

			return false, nil
		}

		// Database error occurred
		s.log.ErrorContext(ctx, "failed to query tenant subscription from database",
			slog.String("tenant_id", tenantID.String()),
			slog.String("error", err.Error()),
		)
		return false, appErrors.NewDatabase(fmt.Errorf("failed to query subscription: %w", err))
	}

	// 4. Determine Active Plan Validity
	now := time.Now().UTC()
	var active bool
	var status = sub.Status

	if sub.EndDate != nil && sub.EndDate.Before(now) {
		status = "expired"
		active = false
	} else if sub.Status == "active" {
		active = true
	} else {
		active = false
	}

	// Check tenant organization status if tenantRepo is provided
	if s.tenantRepo != nil {
		tenant, tErr := s.tenantRepo.GetByID(ctx, tenantID)
		if tErr == nil && tenant != nil && tenant.Status != "active" {
			active = false
			status = tenant.Status
		}
	}

	// 5. Store Result in Redis Cache with 24-hour TTL
	if s.planCache != nil {
		cachedPlan := &domain.CachedTenantPlan{
			Active: active,
			PlanID: &sub.CurrentPlanID,
			Status: status,
		}
		if setErr := s.planCache.Set(ctx, tenantID, cachedPlan, TenantPlanCacheTTL); setErr != nil {
			s.log.WarnContext(ctx, "failed to cache tenant plan in redis",
				slog.String("tenant_id", tenantID.String()),
				slog.String("error", setErr.Error()),
			)
		}
	}

	// 6. Handle Inactive / Expired Plan
	if !active {
		s.log.WarnContext(ctx, "tenant_plan_validation_failed",
			slog.String("tenant_id", tenantID.String()),
			slog.String("reason", "subscription inactive or expired"),
			slog.String("status", status),
		)
		if s.metrics != nil {
			s.metrics.IncTenantPlanValidationFailures()
		}
		return false, nil
	}

	return true, nil
}

// InvalidateTenantPlanCache deletes the tenant's cached plan status from Redis
func (s *tenantPlanService) InvalidateTenantPlanCache(ctx context.Context, tenantID uuid.UUID) error {
	if tenantID == uuid.Nil {
		return nil
	}

	if s.planCache != nil {
		if err := s.planCache.Delete(ctx, tenantID); err != nil {
			s.log.WarnContext(ctx, "failed to invalidate tenant plan cache in redis",
				slog.String("tenant_id", tenantID.String()),
				slog.String("error", err.Error()),
			)
			return err
		}
	}

	s.log.InfoContext(ctx, "tenant_plan_cache_invalidated",
		slog.String("tenant_id", tenantID.String()),
	)

	return nil
}
