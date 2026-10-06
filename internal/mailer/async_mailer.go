package mailer

import (
	"context"
	"time"
)

// EmailTaskQueue abstracts the asynchronous email dispatch queue
type EmailTaskQueue interface {
	EnqueueEmail(ctx context.Context, toEmail, otp string, expiry time.Duration) error
}

// AsyncMailer implements Mailer by enqueueing email tasks to a background worker
type AsyncMailer struct {
	queue EmailTaskQueue
}

// NewAsyncMailer creates a non-blocking Mailer implementation
func NewAsyncMailer(queue EmailTaskQueue) *AsyncMailer {
	return &AsyncMailer{queue: queue}
}

// SendPasswordResetOTP queues the password recovery email for asynchronous background delivery
func (m *AsyncMailer) SendPasswordResetOTP(ctx context.Context, toEmail, otp string, expiry time.Duration) error {
	return m.queue.EnqueueEmail(ctx, toEmail, otp, expiry)
}
