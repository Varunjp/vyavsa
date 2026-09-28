package logger

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Recovery(log Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				requestLog := FromContext(c.Request.Context())

				requestLog.Error().
					Interface("panic", recovered).
					Msg("panic recovered")

				c.AbortWithStatusJSON(
					http.StatusInternalServerError,
					gin.H{
						"error": gin.H{
							"code":    "INTERNAL_SERVER_ERROR",
							"message": "internal server error",
						},
					},
				)
			}
		}()

		c.Next()
	}
}
