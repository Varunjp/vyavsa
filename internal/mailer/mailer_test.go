package mailer

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMailer(t *testing.T) {
	log := logger.Default().Logger

	t.Run("Creates LogMailer when SMTP Host is empty", func(t *testing.T) {
		cfg := config.MailerConfig{
			Host: "",
			Port: 0,
		}
		m := NewMailer(cfg, log)
		require.NotNil(t, m)

		err := m.SendPasswordResetOTP(context.Background(), "user@example.com", "123456", 5*time.Minute)
		assert.NoError(t, err)
	})

	t.Run("Creates SMTPMailer when Host and Port are configured", func(t *testing.T) {
		cfg := config.MailerConfig{
			Host: "smtp.example.com",
			Port: 587,
		}
		m := NewMailer(cfg, log)
		require.NotNil(t, m)
		_, ok := m.(*SMTPMailer)
		assert.True(t, ok)
	})

	t.Run("Live SMTP check from env", func(t *testing.T) {
		if os.Getenv("RUN_LIVE_SMTP_TEST") != "true" {
			t.Skip("skipping live SMTP check unless RUN_LIVE_SMTP_TEST=true")
		}
		_ = godotenv.Load("../../.env")
		cfg, err := config.Load()
		if err != nil {
			t.Skip("config load failed")
		}
		if cfg.Mailer.Host == "" || cfg.Mailer.Username == "" {
			t.Skip("no SMTP config in env")
		}
		m := NewMailer(cfg.Mailer, log)
		err = m.SendPasswordResetOTP(context.Background(), cfg.Mailer.From, "123456", 5*time.Minute)
		require.NoError(t, err)
	})
}
