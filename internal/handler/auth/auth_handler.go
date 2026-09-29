package auth

import (
	"time"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/dto"
	"github.com/Varunjp/vyavsa/internal/service"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/Varunjp/vyavsa/pkg/response"
	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for authentication
type Handler struct {
	authService service.AuthService
}

// NewHandler creates a new auth HTTP handler
func NewHandler(authService service.AuthService) *Handler {
	return &Handler{
		authService: authService,
	}
}

// PlatformLogin handles platform admin authentication
func (h *Handler) PlatformLogin(c *gin.Context) {
	var req dto.PlatformLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ParseBindingError(err))
		return
	}

	req.Identifier = req.GetIdentifier()
	if req.Identifier == "" {
		response.Error(c, appErrors.NewValidation("invalid request payload", map[string]string{
			"identifier": "username or email identifier is required",
		}))
		return
	}

	resp, err := h.authService.LoginPlatformAdmin(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp, "platform administrator authenticated successfully")
}

// TenantLogin handles tenant user authentication
func (h *Handler) TenantLogin(c *gin.Context) {
	var req dto.TenantLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ParseBindingError(err))
		return
	}

	resp, err := h.authService.LoginTenantUser(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp, "tenant user authenticated successfully")
}

// RefreshToken handles refresh token rotation
func (h *Handler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ParseBindingError(err))
		return
	}

	resp, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp, "token pair refreshed successfully")
}

// Logout invalidates the authenticated token
func (h *Handler) Logout(c *gin.Context) {
	claims, err := auth.GetClaims(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	var remaining time.Duration
	if claims.ExpiresAt != nil {
		remaining = time.Until(claims.ExpiresAt.Time)
	}

	if err := h.authService.Logout(c.Request.Context(), claims.TokenID, remaining); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"logged_out": true}, "logged out successfully")
}

// GetMe returns the authenticated user's profile and tenant context
func (h *Handler) GetMe(c *gin.Context) {
	claims, err := auth.GetClaims(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	profile, err := h.authService.GetProfile(c.Request.Context(), claims)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, profile, "profile retrieved successfully")
}

// ForgotPassword initiates the password recovery flow by generating and emailing an OTP
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ParseBindingError(err))
		return
	}

	if err := h.authService.ForgotPassword(c.Request.Context(), req, c.ClientIP()); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil, "If an account exists with this email, an OTP has been sent.")
}

// VerifyResetOTP validates the submitted OTP and issues a short-lived password reset token
func (h *Handler) VerifyResetOTP(c *gin.Context) {
	var req dto.VerifyResetOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ParseBindingError(err))
		return
	}

	resp, err := h.authService.VerifyResetOTP(c.Request.Context(), req, c.ClientIP())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp, "OTP verified successfully")
}

// ResetPassword finalizes the password reset with a valid token and updates credentials
func (h *Handler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ParseBindingError(err))
		return
	}

	if err := h.authService.ResetPassword(c.Request.Context(), req, c.ClientIP()); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil, "Password reset successfully.")
}
