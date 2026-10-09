package mailer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/Varunjp/vyavsa/internal/metrics"
	"github.com/joho/godotenv"
	"github.com/resend/resend-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockResendClient struct {
	sendFunc func(ctx context.Context, params *resend.SendEmailRequest) (*resend.SendEmailResponse, error)
}

func (m *mockResendClient) SendWithContext(ctx context.Context, params *resend.SendEmailRequest) (*resend.SendEmailResponse, error) {
	if m.sendFunc != nil {
		return m.sendFunc(ctx, params)
	}
	return &resend.SendEmailResponse{Id: "re_test_msg_98765"}, nil
}

type mockSender struct {
	lastTo       string
	lastSubject  string
	lastHTMLBody string
	lastTextBody string
	errToReturn  error
}

func (m *mockSender) SendEmail(ctx context.Context, to, subject, htmlBody, textBody string) error {
	m.lastTo = to
	m.lastSubject = subject
	m.lastHTMLBody = htmlBody
	m.lastTextBody = textBody
	return m.errToReturn
}

func TestMailer_ProviderSelection(t *testing.T) {
	log := logger.Default().Logger

	t.Run("Creates LogSender when provider is log", func(t *testing.T) {
		cfg := config.MailerConfig{
			Provider: "log",
		}
		sender, err := NewSender(cfg, log)
		require.NoError(t, err)
		_, ok := sender.(*LogSender)
		assert.True(t, ok)

		m := NewMailer(cfg, log)
		require.NotNil(t, m)
		err = m.SendPasswordResetOTP(context.Background(), "user@example.com", "123456", 5*time.Minute)
		assert.NoError(t, err)
	})

	t.Run("Creates SMTPSender when provider is smtp and host/port configured", func(t *testing.T) {
		cfg := config.MailerConfig{
			Provider: "smtp",
			Host:     "smtp.example.com",
			Port:     587,
			Username: "user",
			Password: "password",
		}
		sender, err := NewSender(cfg, log)
		require.NoError(t, err)
		_, ok := sender.(*SMTPSender)
		assert.True(t, ok)

		m := NewMailer(cfg, log)
		require.NotNil(t, m)
		svc, ok := m.(*EmailService)
		require.True(t, ok)
		_, ok = svc.Sender().(*SMTPSender)
		assert.True(t, ok)
	})

	t.Run("Creates ResendSender when provider is resend and API key configured", func(t *testing.T) {
		cfg := config.MailerConfig{
			Provider:     "resend",
			ResendAPIKey: "re_test_key_12345",
			From:         "noreply@example.com",
			FromName:     "Vyavsa",
		}
		sender, err := NewSender(cfg, log)
		require.NoError(t, err)
		resendSender, ok := sender.(*ResendSender)
		require.True(t, ok)
		assert.Equal(t, "Vyavsa <noreply@example.com>", resendSender.from)

		m := NewMailer(cfg, log)
		require.NotNil(t, m)
		svc, ok := m.(*EmailService)
		require.True(t, ok)
		_, ok = svc.Sender().(*ResendSender)
		assert.True(t, ok)
	})

	t.Run("Unsupported provider returns error and does not silently fall back", func(t *testing.T) {
		cfg := config.MailerConfig{
			Provider: "sendgrid",
		}
		sender, err := NewSender(cfg, log)
		assert.Error(t, err)
		assert.Nil(t, sender)
		assert.Contains(t, err.Error(), "unsupported email provider \"sendgrid\"")

		// NewMailer with unsupported provider returns an error-returning mailer
		m := NewMailer(cfg, log)
		require.NotNil(t, m)
		sendErr := m.SendPasswordResetOTP(context.Background(), "user@example.com", "123456", 5*time.Minute)
		assert.Error(t, sendErr)
		assert.True(t, IsPermanentError(sendErr))
		assert.Contains(t, sendErr.Error(), "email provider is not properly configured")
	})

	t.Run("Missing Resend API key returns error on NewSender", func(t *testing.T) {
		cfg := config.MailerConfig{
			Provider:     "resend",
			ResendAPIKey: "",
		}
		_, err := NewSender(cfg, log)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "RESEND_API_KEY is required")
	})

	t.Run("Missing SMTP host returns error on NewSender", func(t *testing.T) {
		cfg := config.MailerConfig{
			Provider: "smtp",
			Host:     "",
			Port:     587,
		}
		_, err := NewSender(cfg, log)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SMTP_HOST and valid SMTP_PORT are required")
	})

	t.Run("Legacy fallback: Host and Port default to SMTPSender when provider is omitted", func(t *testing.T) {
		cfg := config.MailerConfig{
			Host: "smtp.example.com",
			Port: 587,
		}
		m := NewMailer(cfg, log)
		require.NotNil(t, m)
		svc, ok := m.(*EmailService)
		require.True(t, ok)
		_, ok = svc.Sender().(*SMTPSender)
		assert.True(t, ok)
	})

	t.Run("Legacy fallback: empty Host defaults to LogSender when provider is omitted", func(t *testing.T) {
		cfg := config.MailerConfig{
			Host: "",
			Port: 0,
		}
		m := NewMailer(cfg, log)
		require.NotNil(t, m)
		svc, ok := m.(*EmailService)
		require.True(t, ok)
		_, ok = svc.Sender().(*LogSender)
		assert.True(t, ok)
	})

	t.Run("Backward-compatible NewSMTPMailer and NewLogMailer wrappers work", func(t *testing.T) {
		smtpM := NewSMTPMailer(config.MailerConfig{Host: "smtp.example.com", Port: 587}, log)
		require.NotNil(t, smtpM)

		logM := NewLogMailer(log)
		require.NotNil(t, logM)
		err := logM.SendPasswordResetOTP(context.Background(), "user@example.com", "123456", 5*time.Minute)
		assert.NoError(t, err)
	})
}

func TestResendSender_DeliveryAndErrorHandling(t *testing.T) {
	log := logger.Default().Logger
	appMetrics := metrics.New()

	t.Run("Transmits correctly formatted request to Resend API", func(t *testing.T) {
		var capturedReq *resend.SendEmailRequest
		mockClient := &mockResendClient{
			sendFunc: func(ctx context.Context, params *resend.SendEmailRequest) (*resend.SendEmailResponse, error) {
				capturedReq = params
				return &resend.SendEmailResponse{Id: "re_msg_success_123"}, nil
			},
		}

		cfg := config.MailerConfig{
			From:     "support@vyavsa.com",
			FromName: "Vyavsa Support",
		}
		sender := NewResendSenderWithClient(mockClient, cfg, appMetrics, log)

		err := sender.SendEmail(
			context.Background(),
			"customer@example.com",
			"Your Verification Code",
			"<p>HTML body</p>",
			"Plain text body",
		)
		require.NoError(t, err)

		require.NotNil(t, capturedReq)
		assert.Equal(t, "Vyavsa Support <support@vyavsa.com>", capturedReq.From)
		assert.Equal(t, []string{"customer@example.com"}, capturedReq.To)
		assert.Equal(t, "Your Verification Code", capturedReq.Subject)
		assert.Equal(t, "<p>HTML body</p>", capturedReq.Html)
		assert.Equal(t, "Plain text body", capturedReq.Text)
	})

	t.Run("Empty recipient returns permanent error immediately", func(t *testing.T) {
		mockClient := &mockResendClient{}
		sender := NewResendSenderWithClient(mockClient, config.MailerConfig{}, appMetrics, log)

		err := sender.SendEmail(context.Background(), "", "Subject", "html", "text")
		require.Error(t, err)
		assert.True(t, IsPermanentError(err))
		assert.Contains(t, err.Error(), "recipient email cannot be empty")
	})

	t.Run("Respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately

		mockClient := &mockResendClient{
			sendFunc: func(ctx context.Context, params *resend.SendEmailRequest) (*resend.SendEmailResponse, error) {
				return nil, ctx.Err()
			},
		}
		sender := NewResendSenderWithClient(mockClient, config.MailerConfig{}, appMetrics, log)

		err := sender.SendEmail(ctx, "user@example.com", "Subject", "html", "text")
		require.Error(t, err)
		assert.False(t, IsPermanentError(err))
		assert.Contains(t, err.Error(), "canceled")
	})

	t.Run("Handles 401 Unauthorized as permanent error without exposing API key", func(t *testing.T) {
		secretKey := "re_super_secret_live_api_key_abcdef123456"
		mockClient := &mockResendClient{
			sendFunc: func(ctx context.Context, params *resend.SendEmailRequest) (*resend.SendEmailResponse, error) {
				return nil, fmt.Errorf("[ERROR]: 401 Unauthorized with key %s", secretKey)
			},
		}
		sender := NewResendSenderWithClient(mockClient, config.MailerConfig{}, appMetrics, log)

		err := sender.SendEmail(context.Background(), "user@example.com", "Subject", "html", "text")
		require.Error(t, err)
		assert.True(t, IsPermanentError(err))
		assert.False(t, strings.Contains(err.Error(), secretKey), "API key should be sanitized from error message")
		assert.Contains(t, err.Error(), "[REDACTED_KEY]")
	})

	t.Run("Handles 422 Unprocessable Entity as permanent error", func(t *testing.T) {
		mockClient := &mockResendClient{
			sendFunc: func(ctx context.Context, params *resend.SendEmailRequest) (*resend.SendEmailResponse, error) {
				return nil, errors.New("[ERROR]: 422 Unprocessable Entity: Domain not verified")
			},
		}
		sender := NewResendSenderWithClient(mockClient, config.MailerConfig{}, appMetrics, log)

		err := sender.SendEmail(context.Background(), "user@example.com", "Subject", "html", "text")
		require.Error(t, err)
		assert.True(t, IsPermanentError(err))
		assert.Contains(t, err.Error(), "422")
	})

	t.Run("Handles RateLimit error as transient retryable error", func(t *testing.T) {
		mockClient := &mockResendClient{
			sendFunc: func(ctx context.Context, params *resend.SendEmailRequest) (*resend.SendEmailResponse, error) {
				return nil, &resend.RateLimitError{
					Message: "Too many requests",
				}
			},
		}
		sender := NewResendSenderWithClient(mockClient, config.MailerConfig{}, appMetrics, log)

		err := sender.SendEmail(context.Background(), "user@example.com", "Subject", "html", "text")
		require.Error(t, err)
		assert.False(t, IsPermanentError(err), "Rate limit errors should be transient and retryable")
		assert.Contains(t, err.Error(), "rate limit exceeded")
	})
}

func TestEmailService_SendPasswordResetOTP(t *testing.T) {
	log := logger.Default().Logger
	mock := &mockSender{}

	svc := NewEmailService(mock, config.MailerConfig{From: "support@vyavsa.com"}, nil, log)

	err := svc.SendPasswordResetOTP(context.Background(), "target@example.com", "987654", 10*time.Minute)
	require.NoError(t, err)

	assert.Equal(t, "target@example.com", mock.lastTo)
	assert.Equal(t, "Reset your Vyavsa password", mock.lastSubject)
	assert.Contains(t, mock.lastTextBody, "987654")
	assert.Contains(t, mock.lastTextBody, "10 minutes")
	assert.Contains(t, mock.lastHTMLBody, "987654")
	assert.Contains(t, mock.lastHTMLBody, "10 minutes")
	assert.Contains(t, mock.lastHTMLBody, "<!DOCTYPE html>")
}

func TestLiveSMTPCheck(t *testing.T) {
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
	log := logger.Default().Logger
	m := NewMailer(cfg.Mailer, log)
	err = m.SendPasswordResetOTP(context.Background(), cfg.Mailer.From, "123456", 5*time.Minute)
	require.NoError(t, err)
}
