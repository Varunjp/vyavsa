package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/metrics"
)

func TestMetrics_InitializationAndRecording(t *testing.T) {
	m := metrics.New()
	if m == nil {
		t.Fatal("expected non-nil metrics instance")
	}

	if m.Registry == nil {
		t.Fatal("expected non-nil prometheus registry")
	}

	// Test recording HTTP request
	m.IncInFlight()
	m.RecordHTTPRequest("GET", "/api/v1/plans", 200, 15*time.Millisecond, 1024)
	m.DecInFlight()

	// Test business and recovery metrics
	m.IncTenantsCreated()
	m.IncPasswordResetRequests()
	m.IncPasswordResetOTPSent()
	m.IncPasswordResetOTPVerifySuccess()
	m.IncPasswordResetOTPVerifyFailed()
	m.IncPasswordResetSuccess()
	m.IncPasswordResetFailed()
	m.IncPasswordResetRateLimited()
	m.IncTenantPlanCacheHits()
	m.IncTenantPlanCacheMisses()
	m.IncTenantPlanValidationFailures()

	// Test registering nil pool/client collectors does not panic
	m.RegisterDBPoolMetrics(nil)
	m.RegisterRedisPoolMetrics(nil)

	// Test HTTP Handler
	handler := m.Handler()
	if handler == nil {
		t.Fatal("expected non-nil http handler")
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 from /metrics handler, got %d", rec.Code)
	}

	body := rec.Body.String()
	if body == "" {
		t.Fatal("expected non-empty metrics output")
	}

	// Verify our latency and request metrics exist in scrape output
	expectedMetrics := []string{
		"billbook_http_requests_total",
		"billbook_http_request_duration_seconds_bucket",
		"billbook_http_request_duration_seconds_count",
		"billbook_http_request_duration_seconds_sum",
		"billbook_http_requests_in_flight",
	}

	for _, metricName := range expectedMetrics {
		if !contains(body, metricName) {
			t.Errorf("expected metrics output to contain %q", metricName)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
