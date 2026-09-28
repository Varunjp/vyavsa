package health

import (
	"context"
	"net/http"
	"time"

	"github.com/Varunjp/vyavsa/internal/cache"
	"github.com/Varunjp/vyavsa/internal/database"
	"github.com/gin-gonic/gin"
)

// Handler handles health and readiness checks
type Handler struct {
	db    *database.Postgres
	redis *cache.Redis
}

// NewHandler creates a new health check handler
func NewHandler(db *database.Postgres, redis *cache.Redis) *Handler {
	return &Handler{
		db:    db,
		redis: redis,
	}
}

// Health performs process liveness check (does NOT check database)
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "UP",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// Ready performs readiness check against dependencies (PostgreSQL and Redis)
func (h *Handler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	checks := make(map[string]string)
	isReady := true

	// Check PostgreSQL
	if h.db != nil {
		if err := h.db.Ping(ctx); err != nil {
			checks["postgres"] = "DOWN: " + err.Error()
			isReady = false
		} else {
			checks["postgres"] = "UP"
		}
	} else {
		checks["postgres"] = "UNINITIALIZED"
		isReady = false
	}

	// Check Redis
	if h.redis != nil {
		if err := h.redis.Ping(ctx); err != nil {
			checks["redis"] = "DOWN: " + err.Error()
			// Note: If Redis is configured as non-critical cache, you could choose not to fail readiness,
			// but for a strict check we flag it.
			isReady = false
		} else {
			checks["redis"] = "UP"
		}
	} else {
		checks["redis"] = "NOT_CONFIGURED"
	}

	statusCode := http.StatusOK
	overallStatus := "READY"
	if !isReady {
		statusCode = http.StatusServiceUnavailable
		overallStatus = "NOT_READY"
	}

	c.JSON(statusCode, gin.H{
		"status":    overallStatus,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"checks":    checks,
	})
}
