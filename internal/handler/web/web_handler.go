package web

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for public web pages and user-facing HTML views
type Handler struct{}

// NewHandler creates a new instance of the web Handler
func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) defaultData() gin.H {
	return gin.H{
		"CurrentYear": time.Now().Year(),
	}
}

// setNoCacheHeaders instructs browsers, intermediaries, and bfcache never to cache authenticated views
func setNoCacheHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	c.Header("Surrogate-Control", "no-store")
}

// ShowLanding renders the public landing page (GET /)
func (h *Handler) ShowLanding(c *gin.Context) {
	c.HTML(http.StatusOK, "landing.html", h.defaultData())
}

// ShowLogin renders the tenant user login page (GET /login)
func (h *Handler) ShowLogin(c *gin.Context) {
	setNoCacheHeaders(c)
	c.HTML(http.StatusOK, "login.html", h.defaultData())
}

// ShowPlatformLogin renders the platform administrator login page (GET /platform/login)
func (h *Handler) ShowPlatformLogin(c *gin.Context) {
	setNoCacheHeaders(c)
	c.HTML(http.StatusOK, "platform_login.html", h.defaultData())
}

// ShowRegister renders the tenant self-registration page (GET /register)
func (h *Handler) ShowRegister(c *gin.Context) {
	setNoCacheHeaders(c)
	c.HTML(http.StatusOK, "register.html", h.defaultData())
}

// ShowVerifyOTP renders the OTP verification page (GET /verify-otp)
func (h *Handler) ShowVerifyOTP(c *gin.Context) {
	setNoCacheHeaders(c)
	c.HTML(http.StatusOK, "verify_otp.html", h.defaultData())
}

// ShowForgotPassword renders the password recovery initiation page (GET /forgot-password)
func (h *Handler) ShowForgotPassword(c *gin.Context) {
	setNoCacheHeaders(c)
	c.HTML(http.StatusOK, "forgot_password.html", h.defaultData())
}

// ShowResetPassword renders the set new password page (GET /reset-password)
func (h *Handler) ShowResetPassword(c *gin.Context) {
	setNoCacheHeaders(c)
	c.HTML(http.StatusOK, "reset_password.html", h.defaultData())
}

// ShowDashboard renders the tenant dashboard preview page (GET /dashboard)
func (h *Handler) ShowDashboard(c *gin.Context) {
	setNoCacheHeaders(c)
	c.HTML(http.StatusOK, "dashboard.html", h.defaultData())
}

// ShowPlatformDashboard renders the platform administrator dashboard portal view
func (h *Handler) ShowPlatformDashboard(c *gin.Context) {
	setNoCacheHeaders(c)
	c.HTML(http.StatusOK, "platform_dashboard.html", h.defaultData())
}
