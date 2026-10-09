package mailer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/metrics"
	"github.com/resend/resend-go/v2"
)

// ResendEmailsClient abstracts the Resend Emails service API for testability
type ResendEmailsClient interface {
	SendWithContext(ctx context.Context, params *resend.SendEmailRequest) (*resend.SendEmailResponse, error)
}

// ResendSender delivers transactional emails using the official Resend API
type ResendSender struct {
	client  ResendEmailsClient
	from    string
	metrics *metrics.Metrics
	log     *slog.Logger
}

// NewResendSender creates a production Resend email sender
func NewResendSender(cfg config.MailerConfig, log *slog.Logger) *ResendSender {
	return NewResendSenderWithMetrics(cfg, nil, log)
}

// NewResendSenderWithMetrics creates a Resend sender with Prometheus metrics recording
func NewResendSenderWithMetrics(cfg config.MailerConfig, m *metrics.Metrics, log *slog.Logger) *ResendSender {
	if log == nil {
		log = slog.Default()
	}

	httpClient := &http.Client{
		Timeout: 15 * time.Second,
	}
	resendClient := resend.NewCustomClient(httpClient, cfg.ResendAPIKey)

	return NewResendSenderWithClient(resendClient.Emails, cfg, m, log)
}

// NewResendSenderWithClient creates a Resend sender with an injected client (ideal for unit testing)
func NewResendSenderWithClient(
	client ResendEmailsClient,
	cfg config.MailerConfig,
	m *metrics.Metrics,
	log *slog.Logger,
) *ResendSender {
	if log == nil {
		log = slog.Default()
	}

	from := cfg.From
	if from == "" {
		from = "no-reply@vyavsa.com"
	}
	if cfg.FromName != "" && !strings.Contains(from, "<") {
		from = fmt.Sprintf("%s <%s>", cfg.FromName, from)
	}

	return &ResendSender{
		client:  client,
		from:    from,
		metrics: m,
		log:     log,
	}
}

// SendEmail transmits an email message through the Resend API
func (s *ResendSender) SendEmail(
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

	start := time.Now()
	s.log.InfoContext(ctx, "dispatching email via Resend",
		slog.String("to", to),
		slog.String("subject", subject),
		slog.String("provider", "resend"),
	)

	req := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{to},
		Subject: subject,
		Html:    htmlBody,
		Text:    textBody,
	}

	resp, err := s.client.SendWithContext(ctx, req)
	duration := time.Since(start).Seconds()

	if err != nil {
		if s.metrics != nil {
			s.metrics.IncEmailDelivery("resend", "failure")
			s.metrics.ObserveEmailDeliveryDuration("resend", duration)
		}

		sanitizedErr, isPermanent := s.classifyAndSanitizeError(err)
		s.log.ErrorContext(ctx, "failed to send email via Resend",
			slog.String("to", to),
			slog.String("subject", subject),
			slog.Bool("permanent", isPermanent),
			slog.String("error", sanitizedErr.Error()),
		)

		return &deliveryError{
			err:       sanitizedErr,
			permanent: isPermanent,
		}
	}

	if s.metrics != nil {
		s.metrics.IncEmailDelivery("resend", "success")
		s.metrics.ObserveEmailDeliveryDuration("resend", duration)
	}

	messageID := ""
	if resp != nil {
		messageID = resp.Id
	}

	s.log.InfoContext(ctx, "email sent successfully via Resend",
		slog.String("to", to),
		slog.String("subject", subject),
		slog.String("resend_message_id", messageID),
	)

	return nil
}

// classifyAndSanitizeError maps Resend errors into sanitized, safe application errors
func (s *ResendSender) classifyAndSanitizeError(err error) (error, bool) {
	if err == nil {
		return nil, false
	}

	// Context cancellation or timeout
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("resend request timed out or was canceled: %w", err), false
	}

	// Missing required fields before HTTP request
	var missingErr *resend.MissingRequiredFieldsError
	if errors.As(err, &missingErr) {
		return fmt.Errorf("invalid resend request: %s", sanitizeErrorString(missingErr.Error())), true
	}

	// Rate limit error (transient)
	var rateLimitErr *resend.RateLimitError
	if errors.As(err, &rateLimitErr) || errors.Is(err, resend.ErrRateLimit) {
		return fmt.Errorf("resend rate limit exceeded: %s", sanitizeErrorString(rateLimitErr.Error())), false
	}

	errMsg := err.Error()
	lowerMsg := strings.ToLower(errMsg)

	// Permanent errors: authentication failure, invalid recipient, unverified domain, malformed payload
	isPermanent := strings.Contains(lowerMsg, "401") ||
		strings.Contains(lowerMsg, "403") ||
		strings.Contains(lowerMsg, "422") ||
		strings.Contains(lowerMsg, "400") ||
		strings.Contains(lowerMsg, "unauthorized") ||
		strings.Contains(lowerMsg, "forbidden") ||
		strings.Contains(lowerMsg, "not verified") ||
		strings.Contains(lowerMsg, "unprocessable") ||
		strings.Contains(lowerMsg, "invalid") ||
		strings.Contains(lowerMsg, "recipient")

	cleanMsg := sanitizeErrorString(errMsg)
	return fmt.Errorf("resend email delivery failed: %s", cleanMsg), isPermanent
}
