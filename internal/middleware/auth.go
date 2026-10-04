package middleware

import (
	"strings"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/Varunjp/vyavsa/internal/repository"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/Varunjp/vyavsa/pkg/response"
	"github.com/gin-gonic/gin"
)

const (
	authorizationHeader = "Authorization"
	bearerPrefix        = "Bearer "
)

// Authenticate verifies the incoming Bearer JWT and sets claims in the context
func Authenticate(jwtManager auth.JWTManager, blacklist repository.TokenBlacklistRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(authorizationHeader)
		if authHeader == "" {
			response.Error(c, appErrors.NewUnauthorized("authorization header is required"))
			c.Abort()
			return
		}

		if !strings.HasPrefix(authHeader, bearerPrefix) {
			response.Error(c, appErrors.NewUnauthorized("authorization header must start with Bearer"))
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, bearerPrefix)
		if tokenString == "" {
			response.Error(c, appErrors.NewUnauthorized("token cannot be empty"))
			c.Abort()
			return
		}

		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			response.Error(c, appErrors.NewUnauthorized("invalid or expired access token"))
			c.Abort()
			return
		}

		// Ensure that only access tokens can be used for authenticated endpoints
		if claims.TokenType != "" && claims.TokenType != auth.TokenTypeAccess {
			response.Error(c, appErrors.NewUnauthorized("provided token is not an access token"))
			c.Abort()
			return
		}

		// Check if token has been revoked individually or via user-wide revocation (password reset)
		if blacklist != nil {
			if claims.TokenID != "" {
				revoked, err := blacklist.IsTokenRevoked(c.Request.Context(), claims.TokenID)
				if err != nil {
					logger.FromContext(c.Request.Context()).Warn("failed to check token revocation status", "error", err.Error(), "token_id", claims.TokenID)
				} else if revoked {
					response.Error(c, appErrors.NewUnauthorized("access token has been revoked"))
					c.Abort()
					return
				}
			}
			if claims.IssuedAt != nil {
				userRevoked, err := blacklist.IsUserTokenRevoked(c.Request.Context(), claims.UserID, claims.IssuedAt.Time)
				if err != nil {
					logger.FromContext(c.Request.Context()).Warn("failed to check user revocation status", "error", err.Error(), "user_id", claims.UserID.String())
				} else if userRevoked {
					response.Error(c, appErrors.NewUnauthorized("access token has been revoked"))
					c.Abort()
					return
				}
			}
		}

		// Inject claims into Gin context and standard context
		auth.SetClaims(c, claims)

		// Also enrich request context with user and tenant IDs for structured logging
		ctx := logger.WithUserID(c.Request.Context(), claims.UserID.String())
		if claims.TenantID != nil {
			ctx = logger.WithTenantID(ctx, claims.TenantID.String())
		}
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
