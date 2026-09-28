package middleware

import (
	"log/slog"
	"time"

	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/Varunjp/vyavsa/internal/metrics"
	"github.com/gin-gonic/gin"
)

// RequestLogger logs incoming HTTP requests and updates Prometheus metrics
func RequestLogger(log *logger.Logger, m *metrics.Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		if m != nil {
			m.IncInFlight()
			defer m.DecInFlight()
		}

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()
		size := int64(c.Writer.Size())
		if size < 0 {
			size = 0
		}

		// Route pattern for Prometheus (avoiding high cardinality)
		route := c.FullPath()
		if route == "" {
			if status == 404 {
				route = "not_found"
			} else {
				route = "unmatched"
			}
		}

		if m != nil {
			m.RecordHTTPRequest(c.Request.Method, route, status, duration, size)
		}

		// Skip info log for frequent liveness checks unless there is an error
		if (path == "/health" || path == "/metrics") && status < 400 {
			return
		}

		ctx := c.Request.Context()
		attrs := []slog.Attr{
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("route", route),
			slog.Int("status", status),
			slog.Int64("latency_ms", duration.Milliseconds()),
			slog.Float64("duration_s", duration.Seconds()),
			slog.String("client_ip", c.ClientIP()),
			slog.Int64("bytes_out", size),
		}

		msg := "http request completed"
		switch {
		case status >= 500:
			log.LogAttrs(ctx, slog.LevelError, msg, attrs...)
		case status >= 400:
			log.LogAttrs(ctx, slog.LevelWarn, msg, attrs...)
		default:
			log.LogAttrs(ctx, slog.LevelInfo, msg, attrs...)
		}
	}
}
