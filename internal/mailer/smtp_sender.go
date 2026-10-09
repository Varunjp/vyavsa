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
	"github.com/Varunjp/vyavsa/internal/metrics"
)

// SMTPSender delivers emails over SMTP (e.g. Gmail SMTP in development)
type SMTPSender struct {
	cfg     config.MailerConfig
	metrics *metrics.Metrics
	log     *slog.Logger
}

// NewSMTPSender creates a standard SMTP email sender
func NewSMTPSender(cfg config.MailerConfig, log *slog.Logger) *SMTPSender {
	return NewSMTPSenderWithMetrics(cfg, nil, log)
}

// NewSMTPSenderWithMetrics creates an SMTP sender with Prometheus metrics recording
func NewSMTPSenderWithMetrics(cfg config.MailerConfig, m *metrics.Metrics, log *slog.Logger) *SMTPSender {
	if log == nil {
		log = slog.Default()
	}
	return &SMTPSender{
		cfg:     cfg,
		metrics: m,
		log:     log,
	}
}

// SendEmail constructs a multipart MIME email and transmits it over SMTP
func (s *SMTPSender) SendEmail(
	ctx context.Context,
	to string,
	subject string,
	htmlBody string,
	textBody string,
) error {
	to = strings.TrimSpace(to)
	if to == "" {
		return &deliveryError{
			err:       fmt.Errorf("recipient email cannot be empty"),
			permanent: true,
		}
	}

	from := s.cfg.From
	if from == "" {
		from = "no-reply@vyavsa.com"
	}

	fromName := s.cfg.FromName
	if fromName == "" {
		fromName = "Vyavsa Support"
	}

	msg := strings.Builder{}
	msg.WriteString(fmt.Sprintf("From: %s <%s>\r\n", fromName, from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	boundary := "vyavsa_boundary_part"
	msg.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", boundary))

	if textBody != "" {
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n")
		msg.WriteString(textBody)
		msg.WriteString("\r\n\r\n")
	}

	if htmlBody != "" {
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n")
		msg.WriteString(htmlBody)
		msg.WriteString("\r\n\r\n")
	}

	msg.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	start := time.Now()
	err := s.sendMail(ctx, from, to, []byte(msg.String()))
	duration := time.Since(start).Seconds()

	if err != nil {
		if s.metrics != nil {
			s.metrics.IncEmailDelivery("smtp", "failure")
			s.metrics.ObserveEmailDeliveryDuration("smtp", duration)
		}

		cleanErr := sanitizeErrorString(err.Error())
		isPermanent := strings.Contains(cleanErr, "550") || strings.Contains(cleanErr, "553") || strings.Contains(cleanErr, "501")

		s.log.ErrorContext(ctx, "failed to send email via SMTP",
			slog.String("to", to),
			slog.String("subject", subject),
			slog.String("error", cleanErr),
		)
		return &deliveryError{
			err:       fmt.Errorf("failed to send email via SMTP: %s", cleanErr),
			permanent: isPermanent,
		}
	}

	if s.metrics != nil {
		s.metrics.IncEmailDelivery("smtp", "success")
		s.metrics.ObserveEmailDeliveryDuration("smtp", duration)
	}

	s.log.InfoContext(ctx, "email sent successfully via SMTP",
		slog.String("to", to),
		slog.String("subject", subject),
	)

	return nil
}

func (s *SMTPSender) sendMail(ctx context.Context, from, toEmail string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	var auth smtp.Auth
	if s.cfg.Username != "" && s.cfg.Password != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}

	dialer := &net.Dialer{
		Timeout: 15 * time.Second,
	}

	var conn net.Conn
	var err error

	if s.cfg.Port == 465 {
		tlsConfig := &tls.Config{
			ServerName: s.cfg.Host,
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

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	if s.cfg.Port != 465 {
		if ok, _ := client.Extension("STARTTLS"); ok {
			tlsConfig := &tls.Config{
				ServerName: s.cfg.Host,
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
