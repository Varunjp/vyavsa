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
	router.GET("/reports/daily", handler.ShowDailyReport)
	router.GET("/platform/dashboard", handler.ShowPlatformDashboard)
	router.GET("/platform/tenants", handler.ShowPlatformDashboard)
	router.GET("/platform/tenants/:id", handler.ShowPlatformDashboard)
	router.GET("/platform/plans", handler.ShowPlatformDashboard)
	router.GET("/platform/subscriptions", handler.ShowPlatformDashboard)
	router.GET("/platform/transactions", handler.ShowPlatformDashboard)
	router.GET("/platform/metrics", handler.ShowPlatformDashboard)

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

func TestWebHandler_ShowPlatformDashboard(t *testing.T) {
	router, _ := setupTestWebRouter(t)

	paths := []string{
		"/platform/dashboard",
		"/platform/tenants",
		"/platform/tenants/00000000-0000-0000-0000-000000000001",
		"/platform/plans",
		"/platform/subscriptions",
		"/platform/transactions",
		"/platform/metrics",
	}

	for _, p := range paths {
		t.Run("GET "+p, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, p, nil)
			require.NoError(t, err)

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			body := w.Body.String()
			assert.Contains(t, body, "Platform Dashboard")
			assert.Contains(t, body, "Superadmin Portal")
			assert.Contains(t, body, "Active Tenants")
			assert.Contains(t, body, "Monthly Received Income")
			assert.Contains(t, body, "New Registrations")
			assert.Contains(t, body, "Monthly Income & Revenue Trend")
			assert.Contains(t, body, "platform-sidebar")
		})
	}
}

func TestWebHandler_ShowDailyReport(t *testing.T) {
	router, _ := setupTestWebRouter(t)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/reports/daily", nil)
	require.NoError(t, err)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Daily Business Report")
	assert.Contains(t, body, "Report Date:")
	assert.Contains(t, body, "Download PDF")
	assert.Contains(t, body, "Total Available Funds")
}
