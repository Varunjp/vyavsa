package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/Varunjp/vyavsa/internal/metrics"
	"github.com/Varunjp/vyavsa/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebRoutesIntegration(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			Name:            "vyavsa-test",
			Env:             "test",
			Port:            "8080",
			ShutdownTimeout: 2 * time.Second,
		},
		Metrics: config.MetricsConfig{
			Enabled: false,
		},
		CORS: config.CORSConfig{
			AllowedOrigins: []string{"*"},
		},
	}

	appLogger := logger.Default()
	appMetrics := metrics.New()

	srv := server.New(cfg, appLogger, nil, nil, appMetrics)
	router := srv.Router()

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectInBody   []string
	}{
		{
			name:           "Landing Page GET /",
			path:           "/",
			expectedStatus: http.StatusOK,
			expectInBody: []string{
				"Vyavsa",
				"Simple business management.",
				"Built for small businesses.",
				"Daily Transactions",
				"Customer Dues",
				"Employee Management",
				"Financial Overview",
				"Get Started",
				"Login",
			},
		},
		{
			name:           "Tenant Login GET /login",
			path:           "/login",
			expectedStatus: http.StatusOK,
			expectInBody: []string{
				"Sign in to your business account",
				"tenant-login-form",
				"email",
				"password",
				"Forgot password?",
				"Create an account",
			},
		},
		{
			name:           "Platform Admin Login GET /platform/login",
			path:           "/platform/login",
			expectedStatus: http.StatusOK,
			expectInBody: []string{
				"Platform Administration",
				"Superadmin Portal",
				"platform-login-form",
				"Authenticate Administrator",
			},
		},
		{
			name:           "Tenant Registration GET /register",
			path:           "/register",
			expectedStatus: http.StatusOK,
			expectInBody: []string{
				"Create your business account",
				"register-form",
				"Business / Shop Name",
				"Create Password",
				"Confirm Password",
				"plan-select-container",
			},
		},
		{
			name:           "OTP Verification GET /verify-otp",
			path:           "/verify-otp?email=user@test.com",
			expectedStatus: http.StatusOK,
			expectInBody: []string{
				"Verify your account",
				"otp-form",
				"otp-digit",
				"resend-btn",
			},
		},
		{
			name:           "Forgot Password GET /forgot-password",
			path:           "/forgot-password",
			expectedStatus: http.StatusOK,
			expectInBody: []string{
				"Forgot password?",
				"forgot-password-form",
				"Send Verification Code",
			},
		},
		{
			name:           "Reset Password GET /reset-password",
			path:           "/reset-password",
			expectedStatus: http.StatusOK,
			expectInBody: []string{
				"Set new password",
				"reset-password-form",
				"Update Password",
			},
		},
		{
			name:           "Dashboard GET /dashboard",
			path:           "/dashboard",
			expectedStatus: http.StatusOK,
			expectInBody: []string{
				"Dashboard",
				"Today's Sales",
				"Pending Receivables",
				"Sign Out",
			},
		},
		{
			name:           "Static CSS main.css",
			path:           "/static/css/main.css",
			expectedStatus: http.StatusOK,
			expectInBody: []string{
				"--color-primary",
				"Vyavsa Design System",
			},
		},
		{
			name:           "Static CSS landing.css",
			path:           "/static/css/landing.css",
			expectedStatus: http.StatusOK,
			expectInBody: []string{
				"hero-section",
				"features-grid",
			},
		},
		{
			name:           "Static CSS auth.css",
			path:           "/static/css/auth.css",
			expectedStatus: http.StatusOK,
			expectInBody: []string{
				"auth-card",
				"otp-digit",
			},
		},
		{
			name:           "Static JS api.js",
			path:           "/static/js/api.js",
			expectedStatus: http.StatusOK,
			expectInBody: []string{
				"Vyavsa API Client",
				"BASE_URL",
			},
		},
		{
			name:           "Static JS auth.js",
			path:           "/static/js/auth.js",
			expectedStatus: http.StatusOK,
			expectInBody: []string{
				"Vyavsa Authentication & Session Manager",
				"setTenantAuth",
			},
		},
		{
			name:           "Static Image logo.svg",
			path:           "/static/images/logo.svg",
			expectedStatus: http.StatusOK,
			expectInBody: []string{
				"<svg",
			},
		},
		{
			name:           "Static Image favicon.svg",
			path:           "/static/images/favicon.svg",
			expectedStatus: http.StatusOK,
			expectInBody: []string{
				"<svg",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, tc.path, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			body := w.Body.String()
			for _, expectedStr := range tc.expectInBody {
				assert.Contains(t, body, expectedStr)
			}
		})
	}

	t.Run("GET /favicon.ico redirects to SVG favicon", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/favicon.ico", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusMovedPermanently, w.Code)
		assert.Equal(t, "/static/images/favicon.svg", w.Header().Get("Location"))
	})

	t.Run("GET /dashboard enforces strict no-cache headers to prevent bfcache leaks", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/dashboard", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Cache-Control"), "no-store")
		assert.Contains(t, w.Header().Get("Cache-Control"), "no-cache")
		assert.Equal(t, "no-cache", w.Header().Get("Pragma"))
		assert.Equal(t, "0", w.Header().Get("Expires"))
	})

	t.Run("Landing page excludes platform admin, reset password, and OTP links from footer", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()
		assert.NotContains(t, body, "Platform Admin")
		assert.NotContains(t, body, "OTP Verification")
		assert.NotContains(t, body, "/platform/login")
		assert.NotContains(t, body, "/verify-otp")
	})
}
