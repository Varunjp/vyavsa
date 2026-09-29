package web_test

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Varunjp/vyavsa/internal/handler/web"
	webAssets "github.com/Varunjp/vyavsa/web"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestWebRouter(t *testing.T) (*gin.Engine, *web.Handler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	tmpl, err := template.ParseFS(webAssets.WebFS, "templates/*.html")
	require.NoError(t, err, "failed to parse embedded templates")
	router.SetHTMLTemplate(tmpl)

	handler := web.NewHandler()

	router.GET("/", handler.ShowLanding)
	router.GET("/login", handler.ShowLogin)
	router.GET("/register", handler.ShowRegister)
	router.GET("/verify-otp", handler.ShowVerifyOTP)
	router.GET("/forgot-password", handler.ShowForgotPassword)
	router.GET("/reset-password", handler.ShowResetPassword)
	router.GET("/platform/login", handler.ShowPlatformLogin)
	router.GET("/dashboard", handler.ShowDashboard)

	return router, handler
}

func TestWebHandler_ShowLanding(t *testing.T) {
	router, _ := setupTestWebRouter(t)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Vyavsa")
	assert.Contains(t, body, "Simple business management.")
	assert.Contains(t, body, "Built for small businesses.")
	assert.Contains(t, body, "Daily Transactions")
	assert.Contains(t, body, "Customer Dues")
	assert.Contains(t, body, "Employee Management")
	assert.Contains(t, body, "Financial Overview")
	assert.Contains(t, body, "Get Started")
	assert.NotContains(t, body, "Platform Admin")
	assert.NotContains(t, body, "OTP Verification")
}

func TestWebHandler_ShowLogin(t *testing.T) {
	router, _ := setupTestWebRouter(t)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/login", nil)
	require.NoError(t, err)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Sign in to your business account")
	assert.Contains(t, body, "tenant-login-form")
	assert.Contains(t, body, `id="email"`)
	assert.Contains(t, body, `id="password"`)
	assert.Contains(t, body, "/forgot-password")
	assert.Contains(t, body, "/register")
}

func TestWebHandler_ShowPlatformLogin(t *testing.T) {
	router, _ := setupTestWebRouter(t)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/platform/login", nil)
	require.NoError(t, err)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Platform Administration")
	assert.Contains(t, body, "platform-login-form")
	assert.Contains(t, body, "Authenticate Administrator")
	assert.Contains(t, body, "/login")
}

func TestWebHandler_ShowRegister(t *testing.T) {
	router, _ := setupTestWebRouter(t)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/register", nil)
	require.NoError(t, err)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Create your business account")
	assert.Contains(t, body, "register-form")
	assert.Contains(t, body, `id="name"`)
	assert.Contains(t, body, `id="email"`)
	assert.Contains(t, body, `id="password"`)
	assert.Contains(t, body, `id="confirm-password"`)
	assert.Contains(t, body, "plan-select-container")
}

func TestWebHandler_ShowVerifyOTP(t *testing.T) {
	router, _ := setupTestWebRouter(t)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/verify-otp?email=owner@example.com", nil)
	require.NoError(t, err)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Verify your account")
	assert.Contains(t, body, "otp-form")
	assert.Contains(t, body, "otp-digit")
	assert.Contains(t, body, "resend-btn")
	assert.Contains(t, body, "countdown-text")
}

func TestWebHandler_ShowForgotPassword(t *testing.T) {
	router, _ := setupTestWebRouter(t)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/forgot-password", nil)
	require.NoError(t, err)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Forgot password?")
	assert.Contains(t, body, "forgot-password-form")
	assert.Contains(t, body, "Send Verification Code")
	assert.Contains(t, body, "/login")
}

func TestWebHandler_ShowResetPassword(t *testing.T) {
	router, _ := setupTestWebRouter(t)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/reset-password", nil)
	require.NoError(t, err)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Set new password")
	assert.Contains(t, body, "reset-password-form")
	assert.Contains(t, body, `id="new-password"`)
	assert.Contains(t, body, `id="confirm-password"`)
	assert.Contains(t, body, "reset-success-card")
}

func TestWebHandler_ShowDashboard(t *testing.T) {
	router, _ := setupTestWebRouter(t)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/dashboard", nil)
	require.NoError(t, err)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Dashboard")
	assert.Contains(t, body, "Recent Transactions")
	assert.Contains(t, body, "logout-btn")
}
