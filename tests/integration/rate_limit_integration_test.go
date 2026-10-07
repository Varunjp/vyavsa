package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/cache"
	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/Varunjp/vyavsa/internal/metrics"
	"github.com/Varunjp/vyavsa/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRateLimitIntegrationServer(t *testing.T, redisClient *cache.Redis) (*server.Server, *config.Config) {
	cfg := &config.Config{
		App: config.AppConfig{
			Name:            "vyavsa-test-api",
			Env:             "test",
			Port:            "8080",
			ShutdownTimeout: 2 * time.Second,
		},
		Metrics: config.MetricsConfig{
			Enabled: true,
			Path:    "/metrics",
		},
		CORS: config.CORSConfig{
			AllowedOrigins: []string{"*"},
		},
		JWT: config.JWTConfig{
			Secret:        "integration-test-secret-minimum-32-chars-long",
			AccessExpiry:  15 * time.Minute,
			RefreshExpiry: 24 * time.Hour,
		},
		RateLimit: config.RateLimitConfig{
			Enabled:          true,
			FailOpen:         true,
			AuthFailOpen:     false,
			GeneralRequests:  3,
			GeneralWindow:    5 * time.Second,
			AuthRequests:     2,
			AuthWindow:       5 * time.Second,
			SecurityRequests: 1,
			SecurityWindow:   5 * time.Second,
			TenantRequests:   5,
			TenantWindow:     5 * time.Second,
			PlatformRequests: 5,
			PlatformWindow:   5 * time.Second,
		},
	}

	appLogger := logger.Default()
	appMetrics := metrics.New()

	srv := server.New(cfg, appLogger, nil, redisClient, appMetrics)
	return srv, cfg
}

func getLiveTestRedis(t *testing.T) *cache.Redis {
	cfg := config.RedisConfig{
		Host: "localhost",
		Port: "6379",
		DB:   2, // Use isolated DB 2 for integration tests
	}
	log := logger.Default().Logger
	c, err := cache.NewRedis(context.Background(), cfg, log)
	if err != nil {
		t.Skip("redis is not accessible, skipping live integration test")
	}
	t.Cleanup(func() {
		_ = c.Client.FlushDB(context.Background()).Err()
		_ = c.Close()
	})
	return c
}

func TestRateLimitHTTPPipelineIntegration(t *testing.T) {
	redisClient := getLiveTestRedis(t)
	srv, _ := setupRateLimitIntegrationServer(t, redisClient)
	router := srv.Router()

	t.Run("Health and Metrics endpoints are NEVER blocked by rate limiter", func(t *testing.T) {
		for i := 0; i < 20; i++ {
			req, _ := http.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}

		for i := 0; i < 10; i++ {
			req, _ := http.NewRequest(http.MethodGet, "/metrics", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}
	})

	t.Run("General endpoint /api/v1/ping enforces limit of 3 requests", func(t *testing.T) {
		clientIP := "192.0.2.10"

		sendPing := func() *httptest.ResponseRecorder {
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/ping", nil)
			req.RemoteAddr = clientIP + ":12345"
			req.Header.Set("X-Forwarded-For", clientIP)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			return w
		}

		// 1st request -> 200 OK
		w1 := sendPing()
		assert.Equal(t, http.StatusOK, w1.Code)
		assert.Equal(t, "3", w1.Header().Get("X-RateLimit-Limit"))
		assert.Equal(t, "2", w1.Header().Get("X-RateLimit-Remaining"))

		// 2nd request -> 200 OK
		w2 := sendPing()
		assert.Equal(t, http.StatusOK, w2.Code)
		assert.Equal(t, "1", w2.Header().Get("X-RateLimit-Remaining"))

		// 3rd request -> 200 OK (reaches limit)
		w3 := sendPing()
		assert.Equal(t, http.StatusOK, w3.Code)
		assert.Equal(t, "0", w3.Header().Get("X-RateLimit-Remaining"))

		// 4th request -> 429 Too Many Requests
		w4 := sendPing()
		assert.Equal(t, http.StatusTooManyRequests, w4.Code)
		assert.NotEmpty(t, w4.Header().Get("Retry-After"))

		var resp map[string]any
		err := json.Unmarshal(w4.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.False(t, resp["success"].(bool))
		errObj := resp["error"].(map[string]any)
		assert.Equal(t, "TOO_MANY_REQUESTS", errObj["code"])
	})

	t.Run("Auth endpoint /api/v1/auth/platform/login enforces strict limit of 2 requests", func(t *testing.T) {
		clientIP := "192.0.2.20"

		sendLogin := func() *httptest.ResponseRecorder {
			body := bytes.NewBufferString(`{"identifier":"admin@vyavsa.com","password":"secret"}`)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/platform/login", body)
			req.Header.Set("Content-Type", "application/json")
			req.RemoteAddr = clientIP + ":12345"
			req.Header.Set("X-Forwarded-For", clientIP)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			return w
		}

		// 1st request -> processes (auth fails or 401/422/etc, but NOT 429)
		w1 := sendLogin()
		assert.NotEqual(t, http.StatusTooManyRequests, w1.Code)
		assert.Equal(t, "2", w1.Header().Get("X-RateLimit-Limit"))
		assert.Equal(t, "1", w1.Header().Get("X-RateLimit-Remaining"))

		// 2nd request -> processes (reaches limit)
		w2 := sendLogin()
		assert.NotEqual(t, http.StatusTooManyRequests, w2.Code)
		assert.Equal(t, "0", w2.Header().Get("X-RateLimit-Remaining"))

		// 3rd request -> 429 Too Many Requests
		w3 := sendLogin()
		assert.Equal(t, http.StatusTooManyRequests, w3.Code)
		assert.NotEmpty(t, w3.Header().Get("Retry-After"))
	})

	t.Run("Security endpoint /api/v1/auth/forgot-password enforces very strict limit of 1 request", func(t *testing.T) {
		clientIP := "192.0.2.30"

		sendForgot := func() *httptest.ResponseRecorder {
			body := bytes.NewBufferString(`{"email":"user@test.com"}`)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", body)
			req.Header.Set("Content-Type", "application/json")
			req.RemoteAddr = clientIP + ":12345"
			req.Header.Set("X-Forwarded-For", clientIP)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			return w
		}

		// 1st request -> allowed (returns 200 or 500 depending on DB, not 429)
		w1 := sendForgot()
		assert.NotEqual(t, http.StatusTooManyRequests, w1.Code)
		assert.Equal(t, "1", w1.Header().Get("X-RateLimit-Limit"))
		assert.Equal(t, "0", w1.Header().Get("X-RateLimit-Remaining"))

		// 2nd request -> 429 Too Many Requests immediately
		w2 := sendForgot()
		assert.Equal(t, http.StatusTooManyRequests, w2.Code)
		assert.NotEmpty(t, w2.Header().Get("Retry-After"))
	})
}
