package middleware

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/Varunjp/vyavsa/internal/metrics"
	"github.com/Varunjp/vyavsa/internal/service"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/Varunjp/vyavsa/pkg/response"
	"github.com/gin-gonic/gin"
)

// Rate limit tier definitions
const (
	TierGeneral  = "general"
	TierAuth     = "auth"
	TierSecurity = "security"
	TierTenant   = "tenant"
	TierPlatform = "platform"
)

// RateLimiterMiddleware provides Gin middlewares for distributed rate limiting across API tiers
type RateLimiterMiddleware struct {
	svc     service.RateLimiterService
	cfg     config.RateLimitConfig
	metrics *metrics.Metrics
	log     *logger.Logger
}

// NewRateLimiterMiddleware creates a new RateLimiterMiddleware instance
func NewRateLimiterMiddleware(svc service.RateLimiterService, m *metrics.Metrics, log *logger.Logger) *RateLimiterMiddleware {
	var cfg config.RateLimitConfig
	if svc != nil {
		cfg = svc.GetConfig()
	}
	return &RateLimiterMiddleware{
		svc:     svc,
		cfg:     cfg,
		metrics: m,
		log:     log,
	}
}

// RequireGeneralLimit applies general API rate limiting keyed by client IP
func (rl *RateLimiterMiddleware) RequireGeneralLimit() gin.HandlerFunc {
	return rl.handle(TierGeneral, func(c *gin.Context) string {
		return c.ClientIP()
	}, rl.cfg.GeneralRequests, rl.cfg.GeneralWindow, rl.cfg.FailOpen)
}

// RequireAuthLimit applies strict authentication rate limiting (login, register, token refresh) keyed by client IP
func (rl *RateLimiterMiddleware) RequireAuthLimit() gin.HandlerFunc {
	return rl.handle(TierAuth, func(c *gin.Context) string {
		return c.ClientIP()
	}, rl.cfg.AuthRequests, rl.cfg.AuthWindow, rl.cfg.AuthFailOpen)
}

// RequireSecurityLimit applies very strict security rate limiting (OTP generation/verification, password recovery) keyed by client IP
func (rl *RateLimiterMiddleware) RequireSecurityLimit() gin.HandlerFunc {
	return rl.handle(TierSecurity, func(c *gin.Context) string {
		return c.ClientIP()
	}, rl.cfg.SecurityRequests, rl.cfg.SecurityWindow, rl.cfg.AuthFailOpen)
}

// RequireTenantLimit applies tenant-scoped rate limiting keyed by authoritative TenantID
// This ensures that all users of a tenant share the tenant quota and cannot bypass limits by spawning multiple users.
func (rl *RateLimiterMiddleware) RequireTenantLimit() gin.HandlerFunc {
	return rl.handle(TierTenant, func(c *gin.Context) string {
		tenantID, err := auth.GetTenantID(c)
		if err == nil && tenantID.String() != "" {
			return tenantID.String()
		}
		// Fallback to client IP if tenant identity is not established
		return c.ClientIP()
	}, rl.cfg.TenantRequests, rl.cfg.TenantWindow, rl.cfg.FailOpen)
}

// RequirePlatformAdminLimit applies platform admin rate limiting keyed by authenticated Admin UserID
func (rl *RateLimiterMiddleware) RequirePlatformAdminLimit() gin.HandlerFunc {
	return rl.handle(TierPlatform, func(c *gin.Context) string {
		userID, err := auth.GetUserID(c)
		if err == nil && userID.String() != "" {
			return userID.String()
		}
		return c.ClientIP()
	}, rl.cfg.PlatformRequests, rl.cfg.PlatformWindow, rl.cfg.FailOpen)
}

// handle provides core evaluation, header population, metric recording, and error response handling
func (rl *RateLimiterMiddleware) handle(
	tier string,
	identifierFn func(c *gin.Context) string,
	limit int,
	window time.Duration,
	failOpen bool,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. If rate limiting service is nil or disabled, pass through immediately
		if rl == nil || rl.svc == nil || !rl.cfg.Enabled {
			c.Next()
			return
		}

		identifier := identifierFn(c)
		if identifier == "" {
			identifier = c.ClientIP()
		}

		ctx := c.Request.Context()
		decision, err := rl.svc.Check(ctx, tier, identifier, limit, window, failOpen)
		if err != nil {
			// Redis failed and failOpen is false -> service unavailable
			if rl.log != nil {
				rl.log.WarnContext(ctx, "rate limit evaluation failed in fail-closed mode",
					slog.String("tier", tier),
					slog.String("route", c.FullPath()),
					slog.String("error", err.Error()),
				)
			}
			response.Error(c, appErrors.NewServiceUnavailable("rate limit service temporarily unavailable"))
			c.Abort()
			return
		}

		// 2. Set standard rate limit headers
		c.Header("X-RateLimit-Limit", strconv.FormatInt(decision.Limit, 10))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(decision.Remaining, 10))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(decision.ResetAt.Unix(), 10))

		// 3. Handle rate limit exceeded
		if !decision.Allowed {
			retrySec := int64(decision.RetryAfter.Seconds())
			if retrySec < 1 {
				retrySec = 1
			}

			route := c.FullPath()
			if route == "" {
				route = c.Request.URL.Path
			}

			// Update observability metrics
			if rl.metrics != nil {
				rl.metrics.IncRateLimitRejected(tier, route)
			}

			// Log rate limit rejection (avoid logging raw identifier if sensitive)
			if rl.log != nil {
				rl.log.WarnContext(ctx, "rate limit exceeded",
					slog.String("tier", tier),
					slog.String("route", route),
					slog.String("method", c.Request.Method),
					slog.Int64("retry_after_seconds", retrySec),
				)
			}

			c.Header("Retry-After", strconv.FormatInt(retrySec, 10))
			response.Error(c, appErrors.NewTooManyRequests(
				fmt.Sprintf("rate limit exceeded, please retry after %d seconds", retrySec),
			))
			c.Abort()
			return
		}

		// 4. Request is permitted
		if rl.metrics != nil {
			rl.metrics.IncRateLimitAllowed(tier)
		}

		c.Next()
	}
}
