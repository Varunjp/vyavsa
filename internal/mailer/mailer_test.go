package mailer

import (
	"context"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/logger"
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
}
