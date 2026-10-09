package config

import (
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	App            AppConfig
	Database       DatabaseConfig
	Redis          RedisConfig
	JWT            JWTConfig
	PasswordReset  PasswordResetConfig
	Mailer         MailerConfig
	Log            LogConfig
	Metrics        MetricsConfig
	CORS           CORSConfig
	BootstrapAdmin BootstrapAdminConfig
	RateLimit      RateLimitConfig
}

// AppConfig holds HTTP server and general application settings
type AppConfig struct {
	Name            string
	Env             string
	Port            string
	Timezone        string
	Location        *time.Location
	ShutdownTimeout time.Duration
}

// DatabaseConfig holds PostgreSQL connection and pool parameters
type DatabaseConfig struct {
	Host            string
	Port            string
	Name            string
	User            string
	Password        string
	SSLMode         string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	AutoMigrate     bool
}

// RedisConfig holds Redis connection parameters
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// JWTConfig holds authentication token settings
type JWTConfig struct {
	Secret        string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

// PasswordResetConfig holds OTP-based password recovery parameters
type PasswordResetConfig struct {
	OTPExpiry        time.Duration
	TokenExpiry      time.Duration
	OTPLength        int
	MaxAttempts      int
	ResendCooldown   time.Duration
	MaxEmailRequests int
	MaxIPRequests    int
	RateLimitWindow  time.Duration
}

// MailerConfig holds email dispatch settings for SMTP and Resend providers
type MailerConfig struct {
	Provider     string
	Host         string
	Port         int
	Username     string
	Password     string
	From         string
	FromName     string
	ResendAPIKey string
}

// LogConfig holds structured logging settings
type LogConfig struct {
	Level  string
	Format string // "json" or "text"
}

// MetricsConfig holds Prometheus metrics settings
type MetricsConfig struct {
	Enabled bool
	Path    string
}

// CORSConfig holds Cross-Origin Resource Sharing settings
type CORSConfig struct {
	AllowedOrigins []string
}

// BootstrapAdminConfig holds initial platform admin credentials from the environment
type BootstrapAdminConfig struct {
	Enabled  bool
	Email    string
	Password string
	Username string
	Phone    string
}

// RateLimitConfig holds distributed Redis-backed rate limiting parameters
type RateLimitConfig struct {
	Enabled          bool
	FailOpen         bool
	AuthFailOpen     bool
	GeneralRequests  int
	GeneralWindow    time.Duration
	TenantRequests   int
	TenantWindow     time.Duration
	PlatformRequests int
	PlatformWindow   time.Duration
	AuthRequests     int
	AuthWindow       time.Duration
	SecurityRequests int
	SecurityWindow   time.Duration
}

// ConnectionString returns the PostgreSQL DSN URL
func (d *DatabaseConfig) ConnectionString() string {
	// If a full DATABASE_URL environment variable was provided, parse and return it
	if envURL := os.Getenv("DATABASE_URL"); envURL != "" {
		return envURL
	}

	userInfo := url.UserPassword(d.User, d.Password)
	hostPort := fmt.Sprintf("%s:%s", d.Host, d.Port)

	query := url.Values{}
	query.Set("sslmode", d.SSLMode)

	u := url.URL{
		Scheme:   "postgres",
		User:     userInfo,
		Host:     hostPort,
		Path:     d.Name,
		RawQuery: query.Encode(),
	}

	return u.String()
}

// Addr returns the Redis Host:Port address
func (r *RedisConfig) Addr() string {
	if envAddr := os.Getenv("REDIS_ADDR"); envAddr != "" {
		return envAddr
	}
	return fmt.Sprintf("%s:%s", r.Host, r.Port)
}

// Load loads configuration from environment variables, optionally reading from .env file
func Load() (*Config, error) {
	// Best-effort load from .env file (does not overwrite existing environment variables)
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")
	_ = godotenv.Load("../../.env")

	tz := getEnv("APP_TIMEZONE", "Asia/Kolkata")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.Local
		if loc == nil {
			loc = time.UTC
		}
	}

	cfg := &Config{
		App: AppConfig{
			Name:            getEnv("APP_NAME", "vyavsa-bill-book-api"),
			Env:             getEnv("APP_ENV", "development"),
			Port:            getEnv("APP_PORT", "8080"),
			Timezone:        tz,
			Location:        loc,
			ShutdownTimeout: getDurationEnv("APP_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DATABASE_HOST", "localhost"),
			Port:            getEnv("DATABASE_PORT", "5432"),
			Name:            getEnv("DATABASE_NAME", "billbook"),
			User:            getEnv("DATABASE_USER", "postgres"),
			Password:        getEnv("DATABASE_PASSWORD", "postgres"),
			SSLMode:         getEnv("DATABASE_SSLMODE", "disable"),
			MaxConns:        getInt32Env("DATABASE_MAX_CONNS", 25),
			MinConns:        getInt32Env("DATABASE_MIN_CONNS", 5),
			MaxConnLifetime: getDurationWithFallbackEnv("DATABASE_MAX_CONN_LIFETIME", "DATABASE_MAX_LIFETIME", 30*time.Minute),
			MaxConnIdleTime: getDurationWithFallbackEnv("DATABASE_MAX_CONN_IDLE_TIME", "DATABASE_MAX_IDLE_TIME", 5*time.Minute),
			AutoMigrate:     getBoolEnv("DATABASE_AUTO_MIGRATE", true),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getIntEnv("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:        getEnv("JWT_SECRET", "super-secret-development-key-change-in-production-min-32-chars"),
			AccessExpiry:  getDurationEnv("JWT_ACCESS_EXPIRY", 15*time.Minute),
			RefreshExpiry: getDurationEnv("JWT_REFRESH_EXPIRY", 7*24*time.Hour),
		},
		PasswordReset: PasswordResetConfig{
			OTPExpiry:        getDurationEnv("PASSWORD_RESET_OTP_EXPIRY", 5*time.Minute),
			TokenExpiry:      getDurationEnv("PASSWORD_RESET_TOKEN_EXPIRY", 10*time.Minute),
			OTPLength:        getIntEnv("PASSWORD_RESET_OTP_LENGTH", 6),
			MaxAttempts:      getIntEnv("PASSWORD_RESET_MAX_ATTEMPTS", 5),
			ResendCooldown:   getDurationEnv("PASSWORD_RESET_RESEND_COOLDOWN", 60*time.Second),
			MaxEmailRequests: getIntEnv("PASSWORD_RESET_MAX_EMAIL_REQUESTS", 3),
			MaxIPRequests:    getIntEnv("PASSWORD_RESET_MAX_IP_REQUESTS", 10),
			RateLimitWindow:  getDurationEnv("PASSWORD_RESET_RATE_LIMIT_WINDOW", 15*time.Minute),
		},
		Mailer: func() MailerConfig {
			provider := strings.ToLower(strings.TrimSpace(getEnv("EMAIL_PROVIDER", "")))
			env := getEnv("APP_ENV", "development")
			if provider == "" && env != "production" {
				if getEnv("SMTP_HOST", "") != "" {
					provider = "smtp"
				} else {
					provider = "log"
				}
			}

			emailFrom := getEnvWithFallback("EMAIL_FROM", "SMTP_FROM", "no-reply@vyavsa.com")
			emailFromName := getEnvWithFallback("EMAIL_FROM_NAME", "SMTP_FROM_NAME", "Vyavsa Support")

			// If emailFrom is formatted as RFC 5322 "Name <email@domain.com>", parse and split components
			if parsedAddr, err := mail.ParseAddress(emailFrom); err == nil && parsedAddr.Address != "" {
				emailFrom = parsedAddr.Address
				if parsedAddr.Name != "" && os.Getenv("EMAIL_FROM_NAME") == "" && os.Getenv("SMTP_FROM_NAME") == "" {
					emailFromName = parsedAddr.Name
				}
			}

			return MailerConfig{
				Provider:     provider,
				Host:         getEnv("SMTP_HOST", ""),
				Port:         getIntEnv("SMTP_PORT", 587),
				Username:     getEnv("SMTP_USERNAME", ""),
				Password:     getEnv("SMTP_PASSWORD", ""),
				From:         emailFrom,
				FromName:     emailFromName,
				ResendAPIKey: getEnv("RESEND_API_KEY", ""),
			}
		}(),
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", ""), // Defaults to json in prod, text in dev
		},
		Metrics: MetricsConfig{
			Enabled: getBoolEnv("PROMETHEUS_ENABLED", true),
			Path:    getEnv("PROMETHEUS_PATH", "/metrics"),
		},
		CORS: CORSConfig{
			AllowedOrigins: getSliceEnv("CORS_ALLOWED_ORIGINS", []string{"*"}),
		},
		BootstrapAdmin: BootstrapAdminConfig{
			Enabled:  getBoolEnv("PLATFORM_ADMIN_BOOTSTRAP_ENABLED", true),
			Email:    getEnv("PLATFORM_ADMIN_EMAIL", "admin@vyavsa.com"),
			Password: getEnv("PLATFORM_ADMIN_PASSWORD", "Admin@12345"),
			Username: getEnv("PLATFORM_ADMIN_USERNAME", "platform_admin"),
			Phone:    getEnv("PLATFORM_ADMIN_PHONE", "+919876543210"),
		},
		RateLimit: RateLimitConfig{
			Enabled:          getBoolEnv("RATE_LIMIT_ENABLED", true),
			FailOpen:         getBoolEnv("RATE_LIMIT_FAIL_OPEN", true),
			AuthFailOpen:     getBoolEnv("RATE_LIMIT_AUTH_FAIL_OPEN", false),
			GeneralRequests:  getIntWithFallbackEnv("RATE_LIMIT_GENERAL_REQUESTS", "RATE_LIMIT_REQUESTS", 100),
			GeneralWindow:    getSecondsOrDurationWithFallbackEnv("RATE_LIMIT_GENERAL_WINDOW", "RATE_LIMIT_WINDOW_SECONDS", 60*time.Second),
			TenantRequests:   getIntEnv("RATE_LIMIT_TENANT_REQUESTS", 300),
			TenantWindow:     getSecondsOrDurationEnv("RATE_LIMIT_TENANT_WINDOW", 60*time.Second),
			PlatformRequests: getIntEnv("RATE_LIMIT_PLATFORM_REQUESTS", 300),
			PlatformWindow:   getSecondsOrDurationEnv("RATE_LIMIT_PLATFORM_WINDOW", 60*time.Second),
			AuthRequests:     getIntEnv("RATE_LIMIT_AUTH_REQUESTS", 20),
			AuthWindow:       getSecondsOrDurationEnv("RATE_LIMIT_AUTH_WINDOW", 60*time.Second),
			SecurityRequests: getIntEnv("RATE_LIMIT_SECURITY_REQUESTS", 5),
			SecurityWindow:   getSecondsOrDurationEnv("RATE_LIMIT_SECURITY_WINDOW", 60*time.Second),
		},
	}

	// Default Log Format based on environment if not explicitly set
	if cfg.Log.Format == "" {
		if cfg.App.Env == "production" {
			cfg.Log.Format = "json"
		} else {
			cfg.Log.Format = "text"
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// Validate performs fail-fast validation on configuration
func (c *Config) Validate() error {
	if c.App.Name == "" {
		return fmt.Errorf("APP_NAME cannot be empty")
	}
	if c.App.Port == "" {
		return fmt.Errorf("APP_PORT cannot be empty")
	}
	if c.Database.Name == "" {
		return fmt.Errorf("DATABASE_NAME cannot be empty")
	}
	if c.Database.User == "" {
		return fmt.Errorf("DATABASE_USER cannot be empty")
	}
	if c.Database.MaxConns <= 0 {
		return fmt.Errorf("DATABASE_MAX_CONNS must be greater than 0")
	}
	if c.Database.MinConns < 0 {
		return fmt.Errorf("DATABASE_MIN_CONNS cannot be negative")
	}
	if c.Database.MinConns > c.Database.MaxConns {
		return fmt.Errorf("DATABASE_MIN_CONNS cannot exceed DATABASE_MAX_CONNS")
	}
	if c.App.Env == "production" && len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters in production")
	}
	if c.BootstrapAdmin.Enabled {
		if c.BootstrapAdmin.Email == "" {
			return fmt.Errorf("PLATFORM_ADMIN_EMAIL cannot be empty when bootstrap is enabled")
		}
		if c.BootstrapAdmin.Password == "" {
			return fmt.Errorf("PLATFORM_ADMIN_PASSWORD cannot be empty when bootstrap is enabled")
		}
	}
	if c.RateLimit.Enabled {
		if c.RateLimit.GeneralRequests <= 0 || c.RateLimit.GeneralWindow <= 0 {
			return fmt.Errorf("RATE_LIMIT_GENERAL_REQUESTS and RATE_LIMIT_GENERAL_WINDOW must be positive")
		}
		if c.RateLimit.AuthRequests <= 0 || c.RateLimit.AuthWindow <= 0 {
			return fmt.Errorf("RATE_LIMIT_AUTH_REQUESTS and RATE_LIMIT_AUTH_WINDOW must be positive")
		}
		if c.RateLimit.SecurityRequests <= 0 || c.RateLimit.SecurityWindow <= 0 {
			return fmt.Errorf("RATE_LIMIT_SECURITY_REQUESTS and RATE_LIMIT_SECURITY_WINDOW must be positive")
		}
		if c.RateLimit.TenantRequests <= 0 || c.RateLimit.TenantWindow <= 0 {
			return fmt.Errorf("RATE_LIMIT_TENANT_REQUESTS and RATE_LIMIT_TENANT_WINDOW must be positive")
		}
		if c.RateLimit.PlatformRequests <= 0 || c.RateLimit.PlatformWindow <= 0 {
			return fmt.Errorf("RATE_LIMIT_PLATFORM_REQUESTS and RATE_LIMIT_PLATFORM_WINDOW must be positive")
		}
	}

	// Email provider validation
	switch strings.ToLower(strings.TrimSpace(c.Mailer.Provider)) {
	case "":
		if c.App.Env == "production" {
			return fmt.Errorf("EMAIL_PROVIDER must be explicitly configured in production (e.g. 'resend')")
		}
	case "resend":
		if strings.TrimSpace(c.Mailer.ResendAPIKey) == "" {
			return fmt.Errorf("RESEND_API_KEY cannot be empty when EMAIL_PROVIDER=resend")
		}
		if strings.TrimSpace(c.Mailer.From) == "" {
			return fmt.Errorf("EMAIL_FROM cannot be empty when EMAIL_PROVIDER=resend")
		}
	case "smtp":
		if c.App.Env == "production" {
			return fmt.Errorf("EMAIL_PROVIDER 'smtp' conflicts with production environment policy; 'resend' is required for production email delivery")
		}
		if strings.TrimSpace(c.Mailer.Host) == "" {
			return fmt.Errorf("SMTP_HOST cannot be empty when EMAIL_PROVIDER=smtp")
		}
		if c.Mailer.Port <= 0 {
			return fmt.Errorf("SMTP_PORT must be greater than 0 when EMAIL_PROVIDER=smtp")
		}
		if strings.TrimSpace(c.Mailer.Username) == "" || strings.TrimSpace(c.Mailer.Password) == "" {
			return fmt.Errorf("SMTP_USERNAME and SMTP_PASSWORD cannot be empty when EMAIL_PROVIDER=smtp")
		}
		if strings.TrimSpace(c.Mailer.From) == "" {
			return fmt.Errorf("EMAIL_FROM cannot be empty when EMAIL_PROVIDER=smtp")
		}
	case "log":
		if c.App.Env == "production" {
			return fmt.Errorf("EMAIL_PROVIDER 'log' is not permitted in production")
		}
	default:
		return fmt.Errorf("unsupported EMAIL_PROVIDER %q; valid options are 'resend', 'smtp', or 'log'", c.Mailer.Provider)
	}

	return nil
}

// Helper utilities for environment variable parsing

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return strings.TrimSpace(val)
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
			return i
		}
	}
	return fallback
}

func getInt32Env(key string, fallback int32) int32 {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.ParseInt(strings.TrimSpace(val), 10, 32); err == nil {
			return int32(i)
		}
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(strings.TrimSpace(val)); err == nil {
			return b
		}
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(strings.TrimSpace(val)); err == nil {
			return d
		}
	}
	return fallback
}

func getDurationWithFallbackEnv(primaryKey, secondaryKey string, fallback time.Duration) time.Duration {
	if val := os.Getenv(primaryKey); val != "" {
		if d, err := time.ParseDuration(strings.TrimSpace(val)); err == nil {
			return d
		}
	}
	if val := os.Getenv(secondaryKey); val != "" {
		if d, err := time.ParseDuration(strings.TrimSpace(val)); err == nil {
			return d
		}
	}
	return fallback
}

func getSliceEnv(key string, fallback []string) []string {
	if val := os.Getenv(key); val != "" {
		parts := strings.Split(val, ",")
		res := make([]string, 0, len(parts))
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				res = append(res, trimmed)
			}
		}
		if len(res) > 0 {
			return res
		}
	}
	return fallback
}

func getIntWithFallbackEnv(primaryKey, secondaryKey string, fallback int) int {
	if val := os.Getenv(primaryKey); val != "" {
		if i, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
			return i
		}
	}
	if val := os.Getenv(secondaryKey); val != "" {
		if i, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
			return i
		}
	}
	return fallback
}

func getSecondsOrDurationEnv(key string, fallback time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		trimmed := strings.TrimSpace(val)
		if sec, err := strconv.Atoi(trimmed); err == nil {
			return time.Duration(sec) * time.Second
		}
		if d, err := time.ParseDuration(trimmed); err == nil {
			return d
		}
	}
	return fallback
}

func getSecondsOrDurationWithFallbackEnv(primaryKey, secondaryKey string, fallback time.Duration) time.Duration {
	if val := os.Getenv(primaryKey); val != "" {
		trimmed := strings.TrimSpace(val)
		if sec, err := strconv.Atoi(trimmed); err == nil {
			return time.Duration(sec) * time.Second
		}
		if d, err := time.ParseDuration(trimmed); err == nil {
			return d
		}
	}
	if val := os.Getenv(secondaryKey); val != "" {
		trimmed := strings.TrimSpace(val)
		if sec, err := strconv.Atoi(trimmed); err == nil {
			return time.Duration(sec) * time.Second
		}
		if d, err := time.ParseDuration(trimmed); err == nil {
			return d
		}
	}
	return fallback
}

func getEnvWithFallback(primaryKey, secondaryKey, fallback string) string {
	if val := os.Getenv(primaryKey); val != "" {
		return strings.TrimSpace(val)
	}
	if val := os.Getenv(secondaryKey); val != "" {
		return strings.TrimSpace(val)
	}
	return fallback
}
