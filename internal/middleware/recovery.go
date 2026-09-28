package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/Varunjp/vyavsa/pkg/response"
	"github.com/gin-gonic/gin"
)

// Recovery handles panics cleanly by logging the panic and returning standard error response
func Recovery(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				reqID := logger.RequestIDFromContext(c.Request.Context())

				log.ErrorContext(c.Request.Context(), "panic recovered in HTTP handler",
					slog.Any("panic", r),
					slog.String("stack", stack),
					slog.String("request_id", reqID),
					slog.String("path", c.Request.URL.Path),
					slog.String("method", c.Request.Method),
				)

				c.Abort()
				response.CustomError(
					c,
					http.StatusInternalServerError,
					"INTERNAL_SERVER_ERROR",
					fmt.Sprintf("Internal server error: %v", r),
				)
			}
		}()

		c.Next()
	}
}
