package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all application Prometheus metrics
type Metrics struct {
	Registry *prometheus.Registry

	// HTTP Metrics
	httpRequestsTotal     *prometheus.CounterVec
	httpRequestDuration   *prometheus.HistogramVec
	httpRequestsInFlight  prometheus.Gauge
	httpResponseSizeBytes *prometheus.HistogramVec

	// Business Metrics Counters
	salesTotal          *prometheus.CounterVec
	purchasesTotal      *prometheus.CounterVec
	expensesTotal       *prometheus.CounterVec
	tenantsCreatedTotal prometheus.Counter

	// Password Recovery Metrics
	passwordResetRequestsTotal         prometheus.Counter
	passwordResetOTPSentTotal          prometheus.Counter
	passwordResetOTPVerifySuccessTotal prometheus.Counter
	passwordResetOTPVerifyFailedTotal  prometheus.Counter
	passwordResetSuccessTotal          prometheus.Counter
	passwordResetFailedTotal           prometheus.Counter
	passwordResetRateLimitedTotal      prometheus.Counter
}

// New initializes application metrics and registers them with a custom Prometheus registry
func New() *Metrics {
	reg := prometheus.NewRegistry()

	// Register standard process and Go runtime collectors
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	reg.MustRegister(prometheus.NewGoCollector())

	m := &Metrics{
		Registry: reg,

		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "billbook",
				Subsystem: "http",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests processed",
			},
			[]string{"method", "route", "status_code"},
		),

		httpRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "billbook",
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help:      "HTTP request latency distributions in seconds",
				Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"method", "route", "status_code"},
		),

		httpRequestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "billbook",
				Subsystem: "http",
				Name:      "requests_in_flight",
				Help:      "Current number of HTTP requests being served",
			},
		),

		httpResponseSizeBytes: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "billbook",
				Subsystem: "http",
				Name:      "response_size_bytes",
				Help:      "HTTP response size distributions in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 6), // 100B to 10MB
			},
			[]string{"method", "route", "status_code"},
		),

		salesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "billbook",
				Subsystem: "business",
				Name:      "sales_total",
				Help:      "Total count of recorded sales transactions",
			},
			[]string{"type"}, // line or counter
		),

		purchasesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "billbook",
				Subsystem: "business",
				Name:      "purchases_total",
				Help:      "Total count of recorded purchases",
			},
			[]string{"status"},
		),

		expensesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "billbook",
				Subsystem: "business",
				Name:      "expenses_total",
				Help:      "Total count of recorded expenses",
			},
			[]string{"category"},
		),

		tenantsCreatedTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: "billbook",
				Subsystem: "business",
				Name:      "tenants_created_total",
				Help:      "Total number of tenants created",
			},
		),

		passwordResetRequestsTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: "billbook",
				Subsystem: "auth",
				Name:      "password_reset_requests_total",
				Help:      "Total count of password reset requests initiated",
			},
		),
		passwordResetOTPSentTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: "billbook",
				Subsystem: "auth",
				Name:      "password_reset_otp_sent_total",
				Help:      "Total count of password reset OTPs dispatched",
			},
		),
		passwordResetOTPVerifySuccessTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: "billbook",
				Subsystem: "auth",
				Name:      "password_reset_otp_verification_success_total",
				Help:      "Total count of successful OTP verifications",
			},
		),
		passwordResetOTPVerifyFailedTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: "billbook",
				Subsystem: "auth",
				Name:      "password_reset_otp_verification_failed_total",
				Help:      "Total count of failed OTP verification attempts",
			},
		),
		passwordResetSuccessTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: "billbook",
				Subsystem: "auth",
				Name:      "password_reset_success_total",
				Help:      "Total count of successfully completed password resets",
			},
		),
		passwordResetFailedTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: "billbook",
				Subsystem: "auth",
				Name:      "password_reset_failed_total",
				Help:      "Total count of failed password reset finalizations",
			},
		),
		passwordResetRateLimitedTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: "billbook",
				Subsystem: "auth",
				Name:      "password_reset_rate_limited_total",
				Help:      "Total count of rate-limited password reset requests",
			},
		),
	}

	reg.MustRegister(
		m.httpRequestsTotal,
		m.httpRequestDuration,
		m.httpRequestsInFlight,
		m.httpResponseSizeBytes,
		m.salesTotal,
		m.purchasesTotal,
		m.expensesTotal,
		m.tenantsCreatedTotal,
		m.passwordResetRequestsTotal,
		m.passwordResetOTPSentTotal,
		m.passwordResetOTPVerifySuccessTotal,
		m.passwordResetOTPVerifyFailedTotal,
		m.passwordResetSuccessTotal,
		m.passwordResetFailedTotal,
		m.passwordResetRateLimitedTotal,
	)

	return m
}

// RecordHTTPRequest records latency, status code, and payload size for an HTTP request
func (m *Metrics) RecordHTTPRequest(method, route string, statusCode int, duration time.Duration, size int64) {
	statusStr := strconv.Itoa(statusCode)
	m.httpRequestsTotal.WithLabelValues(method, route, statusStr).Inc()
	m.httpRequestDuration.WithLabelValues(method, route, statusStr).Observe(duration.Seconds())
	if size > 0 {
		m.httpResponseSizeBytes.WithLabelValues(method, route, statusStr).Observe(float64(size))
	}
}

// IncInFlight increments currently in-flight HTTP requests
func (m *Metrics) IncInFlight() {
	m.httpRequestsInFlight.Inc()
}

// DecInFlight decrements currently in-flight HTTP requests
func (m *Metrics) DecInFlight() {
	m.httpRequestsInFlight.Dec()
}

// IncTenantsCreated increments the total tenants created counter
func (m *Metrics) IncTenantsCreated() {
	if m != nil && m.tenantsCreatedTotal != nil {
		m.tenantsCreatedTotal.Inc()
	}
}

// IncPasswordResetRequests increments password reset request counter
func (m *Metrics) IncPasswordResetRequests() {
	if m != nil && m.passwordResetRequestsTotal != nil {
		m.passwordResetRequestsTotal.Inc()
	}
}

// IncPasswordResetOTPSent increments sent recovery OTP counter
func (m *Metrics) IncPasswordResetOTPSent() {
	if m != nil && m.passwordResetOTPSentTotal != nil {
		m.passwordResetOTPSentTotal.Inc()
	}
}

// IncPasswordResetOTPVerifySuccess increments successful OTP verifications counter
func (m *Metrics) IncPasswordResetOTPVerifySuccess() {
	if m != nil && m.passwordResetOTPVerifySuccessTotal != nil {
		m.passwordResetOTPVerifySuccessTotal.Inc()
	}
}

// IncPasswordResetOTPVerifyFailed increments failed OTP verifications counter
func (m *Metrics) IncPasswordResetOTPVerifyFailed() {
	if m != nil && m.passwordResetOTPVerifyFailedTotal != nil {
		m.passwordResetOTPVerifyFailedTotal.Inc()
	}
}

// IncPasswordResetSuccess increments completed password reset counter
func (m *Metrics) IncPasswordResetSuccess() {
	if m != nil && m.passwordResetSuccessTotal != nil {
		m.passwordResetSuccessTotal.Inc()
	}
}

// IncPasswordResetFailed increments failed password reset counter
func (m *Metrics) IncPasswordResetFailed() {
	if m != nil && m.passwordResetFailedTotal != nil {
		m.passwordResetFailedTotal.Inc()
	}
}

// IncPasswordResetRateLimited increments rate limited password recovery counter
func (m *Metrics) IncPasswordResetRateLimited() {
	if m != nil && m.passwordResetRateLimitedTotal != nil {
		m.passwordResetRateLimitedTotal.Inc()
	}
}

// RegisterDBPoolMetrics registers dynamic PostgreSQL pool metrics collectors
func (m *Metrics) RegisterDBPoolMetrics(pool *pgxpool.Pool) {
	collector := &dbPoolCollector{pool: pool}
	m.Registry.MustRegister(collector)
}

// Handler returns the Prometheus HTTP metrics handler
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// dbPoolCollector is a custom Prometheus collector for pgxpool.Stat
type dbPoolCollector struct {
	pool *pgxpool.Pool
}

var (
	descDBTotalConns = prometheus.NewDesc(
		"billbook_db_pool_total_connections",
		"Total number of connections in the PostgreSQL pool",
		nil, nil,
	)
	descDBAcquiredConns = prometheus.NewDesc(
		"billbook_db_pool_acquired_connections",
		"Current number of actively acquired connections in the PostgreSQL pool",
		nil, nil,
	)
	descDBIdleConns = prometheus.NewDesc(
		"billbook_db_pool_idle_connections",
		"Current number of idle connections in the PostgreSQL pool",
		nil, nil,
	)
	descDBMaxConns = prometheus.NewDesc(
		"billbook_db_pool_max_connections",
		"Maximum allowed connections in the PostgreSQL pool",
		nil, nil,
	)
	descDBEmptyAcquires = prometheus.NewDesc(
		"billbook_db_pool_empty_acquire_total",
		"Total times a connection was requested when none were available in the pool",
		nil, nil,
	)
)

func (c *dbPoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descDBTotalConns
	ch <- descDBAcquiredConns
	ch <- descDBIdleConns
	ch <- descDBMaxConns
	ch <- descDBEmptyAcquires
}

func (c *dbPoolCollector) Collect(ch chan<- prometheus.Metric) {
	if c.pool == nil {
		return
	}
	stat := c.pool.Stat()

	ch <- prometheus.MustNewConstMetric(descDBTotalConns, prometheus.GaugeValue, float64(stat.TotalConns()))
	ch <- prometheus.MustNewConstMetric(descDBAcquiredConns, prometheus.GaugeValue, float64(stat.AcquiredConns()))
	ch <- prometheus.MustNewConstMetric(descDBIdleConns, prometheus.GaugeValue, float64(stat.IdleConns()))
	ch <- prometheus.MustNewConstMetric(descDBMaxConns, prometheus.GaugeValue, float64(stat.MaxConns()))
	ch <- prometheus.MustNewConstMetric(descDBEmptyAcquires, prometheus.CounterValue, float64(stat.EmptyAcquireCount()))
}
