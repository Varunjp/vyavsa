package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/Varunjp/vyavsa/internal/config"
)

// Mailer defines email sending contracts
type Mailer interface {
	SendPasswordResetOTP(ctx context.Context, toEmail, otp string, expiry time.Duration) error
}

// NewMailer creates an appropriate Mailer implementation based on configuration
func NewMailer(cfg config.MailerConfig, log *slog.Logger) Mailer {
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.Username = strings.TrimSpace(cfg.Username)
	cfg.Password = strings.TrimSpace(cfg.Password)
	cfg.From = strings.Trim(strings.TrimSpace(cfg.From), "\"")
	cfg.FromName = strings.Trim(strings.TrimSpace(cfg.FromName), "\"")

	if cfg.Host != "" && cfg.Port > 0 {
		log.Info("SMTP mailer active",
			slog.String("host", cfg.Host),
			slog.Int("port", cfg.Port),
			slog.String("from", cfg.From),
			slog.String("user", cfg.Username),
		)
		return NewSMTPMailer(cfg, log)
	}
	log.Warn("SMTP host or port not configured; fallback to simulated LogMailer (emails logged, not delivered)")
	return NewLogMailer(log)
}

// LogMailer logs email dispatch without printing sensitive OTP values
type LogMailer struct {
	log *slog.Logger
}

// NewLogMailer creates a development/test logger-based mailer
func NewLogMailer(log *slog.Logger) *LogMailer {
	return &LogMailer{log: log}
}

// SendPasswordResetOTP simulates sending an email by logging recipient metadata
func (m *LogMailer) SendPasswordResetOTP(ctx context.Context, toEmail, otp string, expiry time.Duration) error {
	m.log.InfoContext(ctx, "simulating password recovery email dispatch",
		slog.String("to", toEmail),
		slog.Duration("expiry", expiry),
		slog.String("service", "Vyavsa Small Business Bill Book"),
	)
	return nil
}

// SMTPMailer delivers emails over standard SMTP
type SMTPMailer struct {
	cfg config.MailerConfig
	log *slog.Logger
}

// NewSMTPMailer creates a production SMTP mailer
func NewSMTPMailer(cfg config.MailerConfig, log *slog.Logger) *SMTPMailer {
	return &SMTPMailer{
		cfg: cfg,
		log: log,
	}
}

// SendPasswordResetOTP sends a branded password reset email containing the single-use OTP
func (m *SMTPMailer) SendPasswordResetOTP(ctx context.Context, toEmail, otp string, expiry time.Duration) error {
	subject := "Reset your Vyavsa password"
	from := m.cfg.From
	if from == "" {
		from = "no-reply@vyavsa.com"
	}

	fromName := m.cfg.FromName
	if fromName == "" {
		fromName = "Vyavsa Support"
	}

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

	msg := strings.Builder{}
	msg.WriteString(fmt.Sprintf("From: %s <%s>\r\n", fromName, from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", toEmail))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	boundary := "vyavsa_boundary_part"
	msg.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", boundary))

	msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msg.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n")
	msg.WriteString(plainBody)
	msg.WriteString("\r\n\r\n")

	msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msg.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n")
	msg.WriteString(htmlBody)
	msg.WriteString("\r\n\r\n")
	msg.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	err := m.sendMail(ctx, from, toEmail, []byte(msg.String()))
	if err != nil {
		m.log.ErrorContext(ctx, "failed to send password reset email via SMTP",
			slog.String("to", toEmail),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to send password recovery email: %w", err)
	}

	m.log.InfoContext(ctx, "password reset email sent successfully via SMTP",
		slog.String("to", toEmail),
	)

	return nil
}

func (m *SMTPMailer) sendMail(ctx context.Context, from, toEmail string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)

	var auth smtp.Auth
	if m.cfg.Username != "" && m.cfg.Password != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}

	dialer := &net.Dialer{
		Timeout: 15 * time.Second,
	}

	var conn net.Conn
	var err error

	if m.cfg.Port == 465 {
		tlsConfig := &tls.Config{
			ServerName: m.cfg.Host,
			MinVersion: tls.VersionTLS12,
		}
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server %s: %w", addr, err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	if m.cfg.Port != 465 {
		if ok, _ := client.Extension("STARTTLS"); ok {
			tlsConfig := &tls.Config{
				ServerName: m.cfg.Host,
				MinVersion: tls.VersionTLS12,
			}
			if err := client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("failed to start TLS: %w", err)
			}
		}
	}

	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("SMTP authentication failed: %w", err)
			}
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL command failed: %w", err)
	}

	if err := client.Rcpt(toEmail); err != nil {
		return fmt.Errorf("SMTP RCPT command failed for %s: %w", toEmail, err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA command failed: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		_ = w.Close()
		return fmt.Errorf("failed to write email body: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to finalize email message: %w", err)
	}

	_ = client.Quit()
	return nil
}
