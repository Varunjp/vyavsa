package worker_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/worker"
)

type mockMailer struct {
	mu           sync.Mutex
	calls        int32
	failUntil    int32
	deliveredOTP []string
}

func (m *mockMailer) SendPasswordResetOTP(ctx context.Context, toEmail, otp string, expiry time.Duration) error {
	call := atomic.AddInt32(&m.calls, 1)
	if call <= m.failUntil {
		return errors.New("simulated SMTP error")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.deliveredOTP = append(m.deliveredOTP, otp)
	return nil
}

func TestEmailWorker_MalformedTask(t *testing.T) {
	mailerMock := &mockMailer{}
	w := worker.NewEmailWorker(mailerMock, nil, nil, nil, 3)

	err := w.EnqueueEmail(context.Background(), "", "123456", 5*time.Minute)
	if err == nil {
		t.Fatal("expected error for empty recipient email, got nil")
	}
}

func TestEmailWorker_SuccessfulDelivery(t *testing.T) {
	mailerMock := &mockMailer{}
	w := worker.NewEmailWorker(mailerMock, nil, nil, nil, 3)

	ctx := context.Background()
	w.Start(ctx)
	defer w.Stop()

	err := w.EnqueueEmail(ctx, "user@example.com", "654321", 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected enqueue error: %v", err)
	}

	// Wait briefly for worker to process
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mailerMock.mu.Lock()
		delivered := len(mailerMock.deliveredOTP)
		mailerMock.mu.Unlock()
		if delivered > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	mailerMock.mu.Lock()
	defer mailerMock.mu.Unlock()
	if len(mailerMock.deliveredOTP) != 1 || mailerMock.deliveredOTP[0] != "654321" {
		t.Fatalf("expected 1 delivered OTP with 654321, got: %v", mailerMock.deliveredOTP)
	}
}

func TestEmailWorker_RetryAfterFailure(t *testing.T) {
	// Fails on attempt 1, succeeds on attempt 2
	mailerMock := &mockMailer{failUntil: 1}
	w := worker.NewEmailWorker(mailerMock, nil, nil, nil, 3)

	ctx := context.Background()
	w.Start(ctx)
	defer w.Stop()

	err := w.EnqueueEmail(ctx, "retry@example.com", "777888", 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected enqueue error: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&mailerMock.calls) >= 2 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if atomic.LoadInt32(&mailerMock.calls) < 2 {
		t.Fatalf("expected at least 2 attempts for retry, got %d", atomic.LoadInt32(&mailerMock.calls))
	}

	mailerMock.mu.Lock()
	defer mailerMock.mu.Unlock()
	if len(mailerMock.deliveredOTP) != 1 {
		t.Fatalf("expected eventual delivery, delivered count: %d", len(mailerMock.deliveredOTP))
	}
}

func TestEmailWorker_PermanentFailureHandling(t *testing.T) {
	// Fails on all attempts (up to maxRetries = 2)
	mailerMock := &mockMailer{failUntil: 10}
	w := worker.NewEmailWorker(mailerMock, nil, nil, nil, 2)

	ctx := context.Background()
	w.Start(ctx)
	defer w.Stop()

	err := w.EnqueueEmail(ctx, "fail@example.com", "999999", 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected enqueue error: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&mailerMock.calls) >= 2 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if atomic.LoadInt32(&mailerMock.calls) != 2 {
		t.Fatalf("expected exactly 2 retry attempts, got %d", atomic.LoadInt32(&mailerMock.calls))
	}

	mailerMock.mu.Lock()
	defer mailerMock.mu.Unlock()
	if len(mailerMock.deliveredOTP) != 0 {
		t.Fatalf("expected 0 delivered OTPs on permanent failure, got %d", len(mailerMock.deliveredOTP))
	}
}

func TestEmailWorker_GracefulShutdown(t *testing.T) {
	mailerMock := &mockMailer{}
	w := worker.NewEmailWorker(mailerMock, nil, nil, nil, 3)

	ctx := context.Background()
	w.Start(ctx)

	// Enqueue tasks before stopping
	for i := 0; i < 5; i++ {
		_ = w.EnqueueEmail(ctx, "drain@example.com", "111111", 5*time.Minute)
	}

	// Stop worker and ensure all queued jobs drain cleanly without hangs
	done := make(chan struct{})
	go func() {
		w.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("worker shutdown timed out, failed to drain gracefully")
	}

	mailerMock.mu.Lock()
	defer mailerMock.mu.Unlock()
	if len(mailerMock.deliveredOTP) != 5 {
		t.Fatalf("expected all 5 queued emails to be drained on shutdown, got %d", len(mailerMock.deliveredOTP))
	}
}
