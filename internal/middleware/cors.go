package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS provides cross-origin resource sharing middleware
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowAll := len(allowedOrigins) == 0 || (len(allowedOrigins) == 1 && allowedOrigins[0] == "*")

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if origin != "" {
			if allowAll {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
			} else {
				for _, allowed := range allowedOrigins {
					if strings.EqualFold(allowed, origin) {
						c.Header("Access-Control-Allow-Origin", origin)
						c.Header("Vary", "Origin")
						break
					}
				}
			}

			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept, Authorization, X-Request-ID, Idempotency-Key")
			c.Header("Access-Control-Expose-Headers", "Content-Length, X-Request-ID")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
