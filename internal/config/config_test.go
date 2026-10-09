package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigValidation(t *testing.T) {
	t.Run("Valid default configuration", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Name:            "billbook-api",
				Env:             "development",
				Port:            "8080",
				ShutdownTimeout: 10 * time.Second,
			},
			Database: DatabaseConfig{
				Host:     "localhost",
				Port:     "5432",
				Name:     "billbook",
				User:     "postgres",
				Password: "secret",
				MaxConns: 20,
				MinConns: 2,
			},
			JWT: JWTConfig{
				Secret: "dev-secret-key-12345",
			},
		}

		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("Fail fast on missing App Name", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Name: "",
				Port: "8080",
			},
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "APP_NAME")
	})

	t.Run("Fail fast on invalid connection pool config", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Name: "test",
				Port: "8080",
			},
			Database: DatabaseConfig{
				Name:     "db",
				User:     "user",
				MaxConns: 5,
				MinConns: 10, // MinConns > MaxConns is invalid
			},
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DATABASE_MIN_CONNS cannot exceed DATABASE_MAX_CONNS")
	})

	t.Run("Fail fast on short JWT secret in production", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Name: "test",
				Port: "8080",
				Env:  "production",
			},
			Database: DatabaseConfig{
				Name:     "db",
				User:     "user",
				MaxConns: 10,
				MinConns: 2,
			},
			JWT: JWTConfig{
				Secret: "short",
			},
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "JWT_SECRET must be at least 32 characters")
	})

	t.Run("ConnectionString generation", func(t *testing.T) {
		os.Unsetenv("DATABASE_URL")
		cfg := DatabaseConfig{
			Host:     "127.0.0.1",
			Port:     "5433",
			Name:     "testdb",
			User:     "myuser",
			Password: "mypassword",
			SSLMode:  "disable",
		}

		connStr := cfg.ConnectionString()
		assert.Contains(t, connStr, "postgres://myuser:mypassword@127.0.0.1:5433/testdb")
		assert.Contains(t, connStr, "sslmode=disable")
	})

	t.Run("Fail fast on invalid RateLimit config when enabled", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Name: "test",
				Port: "8080",
			},
			Database: DatabaseConfig{
				Name:     "db",
				User:     "user",
				MaxConns: 10,
				MinConns: 2,
			},
			RateLimit: RateLimitConfig{
				Enabled:         true,
				GeneralRequests: 0, // Invalid: must be positive
				GeneralWindow:   time.Minute,
			},
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "RATE_LIMIT_GENERAL_REQUESTS")
	})

	t.Run("Pass RateLimit validation when properly configured", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Name: "test",
				Port: "8080",
			},
			Database: DatabaseConfig{
				Name:     "db",
				User:     "user",
				MaxConns: 10,
				MinConns: 2,
			},
			RateLimit: RateLimitConfig{
				Enabled:          true,
				GeneralRequests:  100,
				GeneralWindow:    time.Minute,
				AuthRequests:     20,
				AuthWindow:       time.Minute,
				SecurityRequests: 5,
				SecurityWindow:   time.Minute,
				TenantRequests:   300,
				TenantWindow:     time.Minute,
				PlatformRequests: 300,
				PlatformWindow:   time.Minute,
			},
		}
		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("Email Provider - Resend validates successfully in production", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Name: "test",
				Port: "8080",
				Env:  "production",
			},
			Database: DatabaseConfig{
				Name:     "db",
				User:     "user",
				MaxConns: 10,
			},
			JWT: JWTConfig{
				Secret: "super-secret-key-that-is-at-least-32-bytes-long",
			},
			Mailer: MailerConfig{
				Provider:     "resend",
				ResendAPIKey: "re_prod_test_key_12345",
				From:         "noreply@vyavsa.com",
			},
		}
		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("Email Provider - Missing Resend API key fails fast", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Name: "test",
				Port: "8080",
				Env:  "production",
			},
			Database: DatabaseConfig{
				Name:     "db",
				User:     "user",
				MaxConns: 10,
			},
			JWT: JWTConfig{
				Secret: "super-secret-key-that-is-at-least-32-bytes-long",
			},
			Mailer: MailerConfig{
				Provider:     "resend",
				ResendAPIKey: "", // missing
				From:         "noreply@vyavsa.com",
			},
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "RESEND_API_KEY cannot be empty")
	})

	t.Run("Email Provider - SMTP validates successfully in development", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Name: "test",
				Port: "8080",
				Env:  "development",
			},
			Database: DatabaseConfig{
				Name:     "db",
				User:     "user",
				MaxConns: 10,
			},
			Mailer: MailerConfig{
				Provider: "smtp",
				Host:     "smtp.gmail.com",
				Port:     587,
				Username: "user@gmail.com",
				Password: "app-password",
				From:     "user@gmail.com",
			},
		}
		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("Email Provider - Missing SMTP credentials fails fast", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Name: "test",
				Port: "8080",
				Env:  "development",
			},
			Database: DatabaseConfig{
				Name:     "db",
				User:     "user",
				MaxConns: 10,
			},
			Mailer: MailerConfig{
				Provider: "smtp",
				Host:     "smtp.gmail.com",
				Port:     587,
				Username: "", // missing credentials
				Password: "",
				From:     "user@gmail.com",
			},
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SMTP_USERNAME and SMTP_PASSWORD cannot be empty")
	})

	t.Run("Email Provider - Missing SMTP host fails fast", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Name: "test",
				Port: "8080",
				Env:  "development",
			},
			Database: DatabaseConfig{
				Name:     "db",
				User:     "user",
				MaxConns: 10,
			},
			Mailer: MailerConfig{
				Provider: "smtp",
				Host:     "", // missing
				Port:     587,
				Username: "user",
				Password: "password",
				From:     "user@gmail.com",
			},
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SMTP_HOST cannot be empty")
	})

	t.Run("Email Provider - Unsupported provider fails clearly", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Name: "test",
				Port: "8080",
				Env:  "development",
			},
			Database: DatabaseConfig{
				Name:     "db",
				User:     "user",
				MaxConns: 10,
			},
			Mailer: MailerConfig{
				Provider: "sendgrid",
			},
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported EMAIL_PROVIDER \"sendgrid\"")
	})

	t.Run("Email Provider - Production cannot silently fall back to development SMTP", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Name: "test",
				Port: "8080",
				Env:  "production",
			},
			Database: DatabaseConfig{
				Name:     "db",
				User:     "user",
				MaxConns: 10,
			},
			JWT: JWTConfig{
				Secret: "super-secret-key-that-is-at-least-32-bytes-long",
			},
			Mailer: MailerConfig{
				Provider: "", // unconfigured in production
			},
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "EMAIL_PROVIDER must be explicitly configured in production")
	})

	t.Run("Email Provider - Production rejects development SMTP provider", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Name: "test",
				Port: "8080",
				Env:  "production",
			},
			Database: DatabaseConfig{
				Name:     "db",
				User:     "user",
				MaxConns: 10,
			},
			JWT: JWTConfig{
				Secret: "super-secret-key-that-is-at-least-32-bytes-long",
			},
			Mailer: MailerConfig{
				Provider: "smtp",
				Host:     "smtp.gmail.com",
				Port:     587,
				Username: "user",
				Password: "password",
				From:     "user@gmail.com",
			},
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "conflicts with production environment policy")
	})

	t.Run("Email Provider - Production rejects simulated log provider", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Name: "test",
				Port: "8080",
				Env:  "production",
			},
			Database: DatabaseConfig{
				Name:     "db",
				User:     "user",
				MaxConns: 10,
			},
			JWT: JWTConfig{
				Secret: "super-secret-key-that-is-at-least-32-bytes-long",
			},
			Mailer: MailerConfig{
				Provider: "log",
			},
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "EMAIL_PROVIDER 'log' is not permitted in production")
	})
}
