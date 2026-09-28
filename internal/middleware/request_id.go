package middleware

import (
	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const HeaderXRequestID = "X-Request-ID"

// RequestID ensures every incoming HTTP request has a unique request ID
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader(HeaderXRequestID)
		if reqID == "" {
			reqID = uuid.NewString()
		}

		// Set header in response
		c.Header(HeaderXRequestID, reqID)
		c.Set("request_id", reqID)

		// Inject into request context for downstream logger and service usage
		ctx := logger.WithRequestID(c.Request.Context(), reqID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
