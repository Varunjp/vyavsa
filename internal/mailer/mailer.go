package mailer

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/metrics"
)

// Mailer defines high-level business email sending contracts
type Mailer interface {
	SendPasswordResetOTP(ctx context.Context, toEmail, otp string, expiry time.Duration) error
}

// NewSender initializes an EmailSender based on configuration
func NewSender(cfg config.MailerConfig, log *slog.Logger) (EmailSender, error) {
	return NewSenderWithMetrics(cfg, nil, log)
}

// NewSenderWithMetrics initializes an EmailSender with Prometheus metrics recording
func NewSenderWithMetrics(cfg config.MailerConfig, m *metrics.Metrics, log *slog.Logger) (EmailSender, error) {
	if log == nil {
		log = slog.Default()
	}

	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.Username = strings.TrimSpace(cfg.Username)
	cfg.Password = strings.TrimSpace(cfg.Password)
	cfg.From = strings.Trim(strings.TrimSpace(cfg.From), "\"")
	cfg.FromName = strings.Trim(strings.TrimSpace(cfg.FromName), "\"")
	cfg.ResendAPIKey = strings.TrimSpace(cfg.ResendAPIKey)

	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	switch provider {
	case "resend":
		if cfg.ResendAPIKey == "" {
			return nil, fmt.Errorf("RESEND_API_KEY is required when EMAIL_PROVIDER=resend")
		}
		log.Info("initializing Resend email provider",
			slog.String("from", cfg.From),
			slog.String("from_name", cfg.FromName),
		)
		return NewResendSenderWithMetrics(cfg, m, log), nil

	case "smtp":
		if cfg.Host == "" || cfg.Port <= 0 {
			return nil, fmt.Errorf("SMTP_HOST and valid SMTP_PORT are required when EMAIL_PROVIDER=smtp")
		}
		log.Info("initializing SMTP email provider",
			slog.String("host", cfg.Host),
			slog.Int("port", cfg.Port),
			slog.String("from", cfg.From),
			slog.String("user", cfg.Username),
		)
		return NewSMTPSenderWithMetrics(cfg, m, log), nil

	case "log":
		log.Info("initializing Log email provider (simulated delivery)")
		return NewLogSenderWithMetrics(m, log), nil

	case "":
		// Fallback for tests or legacy callers: check Host/Port
		if cfg.Host != "" && cfg.Port > 0 {
			log.Info("SMTP host configured without explicit provider, defaulting to SMTP",
				slog.String("host", cfg.Host),
				slog.Int("port", cfg.Port),
			)
			return NewSMTPSenderWithMetrics(cfg, m, log), nil
		}
		log.Info("email provider unconfigured; defaulting to simulated Log provider")
		return NewLogSenderWithMetrics(m, log), nil

	default:
		return nil, fmt.Errorf("unsupported email provider %q; valid options are 'resend', 'smtp', or 'log'", cfg.Provider)
	}
}

// NewMailer creates an appropriate Mailer implementation based on configuration
func NewMailer(cfg config.MailerConfig, log *slog.Logger) Mailer {
	return NewMailerWithMetrics(cfg, nil, log)
}

// NewMailerWithMetrics creates a Mailer with metrics recording
func NewMailerWithMetrics(cfg config.MailerConfig, m *metrics.Metrics, log *slog.Logger) Mailer {
	if log == nil {
		log = slog.Default()
	}

	sender, err := NewSenderWithMetrics(cfg, m, log)
	if err != nil {
		log.Error("failed to initialize configured email sender, email sending will fail",
			slog.String("provider", cfg.Provider),
			slog.String("error", err.Error()),
		)
		return NewEmailService(newErrorSender(err), cfg, m, log)
	}

	return NewEmailService(sender, cfg, m, log)
}

// EmailService implements Mailer by rendering business email templates and dispatching via EmailSender
type EmailService struct {
	sender  EmailSender
	cfg     config.MailerConfig
	metrics *metrics.Metrics
	log     *slog.Logger
}

// NewEmailService constructs a new business email service
func NewEmailService(
	sender EmailSender,
	cfg config.MailerConfig,
	m *metrics.Metrics,
	log *slog.Logger,
) *EmailService {
	if log == nil {
		log = slog.Default()
	}
	return &EmailService{
		sender:  sender,
		cfg:     cfg,
		metrics: m,
		log:     log,
	}
}

// Sender returns the underlying EmailSender
func (s *EmailService) Sender() EmailSender {
	return s.sender
}

// SendPasswordResetOTP renders and dispatches a branded password reset OTP email
func (s *EmailService) SendPasswordResetOTP(ctx context.Context, toEmail, otp string, expiry time.Duration) error {
	subject := "Reset your Vyavsa password"

	expiryMinutes := int(expiry.Minutes())
	if expiryMinutes <= 0 {
		expiryMinutes = 5
	}

	plainBody := fmt.Sprintf(
		"Hello,\n\n"+
			"Your Vyavsa password reset OTP is %s.\n\n"+
			"This OTP expires in %d minutes.\n\n"+
			"Security Notice: Do NOT share this OTP with anyone. Vyavsa staff will never ask for your OTP.\n"+
			"If you did not request a password reset, you can safely ignore this email.\n\n"+
			"— The Vyavsa Team",
		otp, expiryMinutes,
	)

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #f9fafb; margin: 0; padding: 24px; }
    .container { max-width: 540px; margin: 0 auto; background: #ffffff; border-radius: 8px; border: 1px solid #e5e7eb; padding: 32px; }
    .header { text-align: center; margin-bottom: 24px; }
    .title { font-size: 22px; font-weight: 700; color: #111827; margin: 0; }
    .badge { display: inline-block; background-color: #f3f4f6; color: #4f46e5; font-size: 14px; font-weight: 600; padding: 4px 12px; border-radius: 9999px; margin-top: 8px; }
    .content { color: #374151; font-size: 15px; line-height: 1.6; margin-bottom: 24px; }
    .otp-box { background: #f0fdf4; border: 1px solid #86efac; border-radius: 6px; text-align: center; padding: 18px; margin: 24px 0; }
    .otp-code { font-size: 32px; font-weight: 800; letter-spacing: 8px; color: #15803d; font-family: monospace; }
    .warning { background-color: #fef2f2; border-left: 4px solid #ef4444; padding: 12px 16px; margin: 20px 0; font-size: 13px; color: #991b1b; }
    .footer { font-size: 12px; color: #9ca3af; text-align: center; border-top: 1px solid #e5e7eb; padding-top: 16px; margin-top: 24px; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1 class="title">Vyavsa</h1>
      <span class="badge">Password Recovery</span>
    </div>
    <div class="content">
      <p>Hello,</p>
      <p>We received a request to reset your password for your Vyavsa account. Use the one-time password (OTP) below to proceed:</p>
      <div class="otp-box">
        <div class="otp-code">%s</div>
      </div>
      <p>This OTP will expire in <strong>%d minutes</strong>.</p>
      <div class="warning">
        <strong>Security Warning:</strong> Never share this code with anyone. Vyavsa employees will never ask for your verification code.
      </div>
      <p>If you did not request a password reset, please ignore this email or contact support if you suspect unauthorized access.</p>
    </div>
    <div class="footer">
      &copy; Vyavsa Bill Book SaaS. All rights reserved.
    </div>
  </div>
</body>
</html>`, otp, expiryMinutes)

	return s.sender.SendEmail(ctx, toEmail, subject, htmlBody, plainBody)
}

// -----------------------------------------------------------------------------
// Backward-Compatibility Adapters
// -----------------------------------------------------------------------------

// LogMailer preserves the legacy LogMailer type for existing tests and callers
type LogMailer struct {
	service *EmailService
}

// NewLogMailer creates a development/test logger-based mailer
func NewLogMailer(log *slog.Logger) *LogMailer {
	sender := NewLogSender(log)
	return &LogMailer{
		service: NewEmailService(sender, config.MailerConfig{Provider: "log"}, nil, log),
	}
}

// SendPasswordResetOTP simulates sending an email by logging recipient metadata
func (m *LogMailer) SendPasswordResetOTP(ctx context.Context, toEmail, otp string, expiry time.Duration) error {
	return m.service.SendPasswordResetOTP(ctx, toEmail, otp, expiry)
}

// SMTPMailer preserves the legacy SMTPMailer type for existing tests and callers
type SMTPMailer struct {
	service *EmailService
}

// NewSMTPMailer creates an SMTP mailer
func NewSMTPMailer(cfg config.MailerConfig, log *slog.Logger) *SMTPMailer {
	sender := NewSMTPSender(cfg, log)
	return &SMTPMailer{
		service: NewEmailService(sender, cfg, nil, log),
	}
}

// SendPasswordResetOTP sends a branded password reset email via SMTP
func (m *SMTPMailer) SendPasswordResetOTP(ctx context.Context, toEmail, otp string, expiry time.Duration) error {
	return m.service.SendPasswordResetOTP(ctx, toEmail, otp, expiry)
}
