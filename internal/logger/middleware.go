package logger

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func RequestLogger(log Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		requestID := c.GetHeader("X-Request-ID")

		if requestID == "" {
			requestID = uuid.NewString()
		}

		c.Header("X-Request-ID", requestID)

		requestLog := WithRequestID(
			log.Zerolog,
			requestID,
		)

		requestLog = requestLog.With().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("remote_ip", c.ClientIP()).
			Logger()

		c.Request = c.Request.WithContext(
			WithContext(
				c.Request.Context(),
				requestLog,
			),
		)

		requestLog.Info().
			Msg("request started")

		c.Next()

		duration := time.Since(start)

		status := c.Writer.Status()

		event := requestLog.Info()

		if status >= 500 {
			event = requestLog.Error()
		} else if status >= 400 {
			event = requestLog.Warn()
		}

		event.
			Int("status", status).
			Int64("duration_ms", duration.Milliseconds()).
			Msg("request completed")
	}
}
