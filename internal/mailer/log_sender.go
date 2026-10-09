package mailer

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Varunjp/vyavsa/internal/metrics"
)

// LogSender simulates email delivery by logging metadata (useful for development and test suites)
type LogSender struct {
	metrics *metrics.Metrics
	log     *slog.Logger
}

// NewLogSender creates a local simulated email sender
func NewLogSender(log *slog.Logger) *LogSender {
	return NewLogSenderWithMetrics(nil, log)
}

// NewLogSenderWithMetrics creates a simulated email sender with Prometheus metrics recording
func NewLogSenderWithMetrics(m *metrics.Metrics, log *slog.Logger) *LogSender {
	if log == nil {
		log = slog.Default()
	}
	return &LogSender{
		metrics: m,
		log:     log,
	}
}

// SendEmail logs email dispatch metadata without printing sensitive payload contents
func (s *LogSender) SendEmail(
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
	s.log.InfoContext(ctx, "simulating email dispatch (log provider)",
		slog.String("to", to),
		slog.String("subject", subject),
		slog.String("provider", "log"),
	)

	duration := time.Since(start).Seconds()
	if s.metrics != nil {
		s.metrics.IncEmailDelivery("log", "success")
		s.metrics.ObserveEmailDeliveryDuration("log", duration)
	}

	return nil
}
