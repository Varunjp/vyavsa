package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Varunjp/vyavsa/internal/cache"
	"github.com/Varunjp/vyavsa/internal/mailer"
	"github.com/Varunjp/vyavsa/internal/metrics"
)

const (
	// EmailQueueKey is the Redis list key for asynchronous email jobs
	EmailQueueKey = "vyavsa:queue:email"
)

// EmailJob represents an asynchronous email dispatch task
type EmailJob struct {
	ToEmail string        `json:"to_email"`
	OTP     string        `json:"otp"`
	Expiry  time.Duration `json:"expiry"`
}

// EmailWorker processes email jobs asynchronously in the background
type EmailWorker struct {
	mailer       mailer.Mailer
	redis        *cache.Redis
	metrics      *metrics.Metrics
	log          *slog.Logger
	inMemoryChan chan EmailJob
	stopCh       chan struct{}
	wg           sync.WaitGroup
	maxRetries   int
}

// NewEmailWorker creates a background worker for email sending
func NewEmailWorker(
	m mailer.Mailer,
	redis *cache.Redis,
	metrics *metrics.Metrics,
	log *slog.Logger,
	maxRetries int,
) *EmailWorker {
	if maxRetries <= 0 {
		maxRetries = 3
	}
	if log == nil {
		log = slog.Default()
	}
	return &EmailWorker{
		mailer:       m,
		redis:        redis,
		metrics:      metrics,
		log:          log,
		inMemoryChan: make(chan EmailJob, 1024),
		stopCh:       make(chan struct{}),
		maxRetries:   maxRetries,
	}
}

// Enqueue queues an email dispatch task
func (w *EmailWorker) Enqueue(ctx context.Context, job EmailJob) error {
	if job.ToEmail == "" {
		if w.metrics != nil {
			w.metrics.IncEmailWorkerFailure()
		}
		return fmt.Errorf("recipient email cannot be empty")
	}

	if w.redis != nil && w.redis.Client != nil {
		data, err := json.Marshal(job)
		if err == nil {
			err = w.redis.Client.LPush(ctx, EmailQueueKey, data).Err()
			if err == nil {
				return nil
			}
			w.log.WarnContext(ctx, "failed to enqueue email to redis, falling back to in-memory queue",
				slog.String("error", err.Error()),
				slog.String("to", job.ToEmail),
			)
		}
	}

	select {
	case w.inMemoryChan <- job:
		return nil
	default:
		w.log.ErrorContext(ctx, "in-memory email queue is full, dropping email task",
			slog.String("to", job.ToEmail),
		)
		if w.metrics != nil {
			w.metrics.IncEmailWorkerFailure()
		}
		return fmt.Errorf("email worker queue is full")
	}
}

// EnqueueEmail implements mailer.EmailTaskQueue
func (w *EmailWorker) EnqueueEmail(ctx context.Context, toEmail, otp string, expiry time.Duration) error {
	return w.Enqueue(ctx, EmailJob{
		ToEmail: toEmail,
		OTP:     otp,
		Expiry:  expiry,
	})
}

// Start begins worker execution
func (w *EmailWorker) Start(ctx context.Context) {
	w.wg.Add(1)
	go w.run(ctx)
}

// Stop signals graceful shutdown and waits for workers to drain
func (w *EmailWorker) Stop() {
	close(w.stopCh)
	w.wg.Wait()
}

func (w *EmailWorker) run(ctx context.Context) {
	defer w.wg.Done()
	w.log.Info("starting email background worker")

	for {
		select {
		case <-w.stopCh:
			w.drainRemaining(context.Background())
			w.log.Info("email background worker stopped gracefully")
			return
		default:
		}

		job, ok := w.fetchNextJob(ctx)
		if !ok {
			select {
			case <-w.stopCh:
				w.drainRemaining(context.Background())
				w.log.Info("email background worker stopped gracefully")
				return
			case <-time.After(50 * time.Millisecond):
			}
			continue
		}

		w.processWithRetry(ctx, job)
	}
}

func (w *EmailWorker) fetchNextJob(ctx context.Context) (EmailJob, bool) {
	if w.redis != nil && w.redis.Client != nil {
		res, err := w.redis.Client.RPop(ctx, EmailQueueKey).Result()
		if err == nil && res != "" {
			var job EmailJob
			if err := json.Unmarshal([]byte(res), &job); err == nil {
				return job, true
			}
			// Malformed task handling
			w.log.ErrorContext(ctx, "malformed email task payload in redis queue", slog.String("raw", res))
			if w.metrics != nil {
				w.metrics.IncEmailWorkerFailure()
			}
		}
	}

	select {
	case job := <-w.inMemoryChan:
		return job, true
	default:
		return EmailJob{}, false
	}
}

func (w *EmailWorker) drainRemaining(ctx context.Context) {
	for {
		select {
		case job := <-w.inMemoryChan:
			w.processWithRetry(ctx, job)
		default:
			return
		}
	}
}

func (w *EmailWorker) processWithRetry(ctx context.Context, job EmailJob) {
	if w.mailer == nil {
		w.log.WarnContext(ctx, "mailer is nil, skipping email job", slog.String("to", job.ToEmail))
		return
	}
	var err error
	for attempt := 1; attempt <= w.maxRetries; attempt++ {
		procCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		err = w.mailer.SendPasswordResetOTP(procCtx, job.ToEmail, job.OTP, job.Expiry)
		cancel()

		if err == nil {
			if w.metrics != nil {
				w.metrics.IncEmailWorkerJob()
			}
			w.log.InfoContext(ctx, "email delivered successfully by background worker",
				slog.String("to", job.ToEmail),
			)
			return
		}

		if w.metrics != nil {
			w.metrics.IncEmailWorkerRetry()
		}

		w.log.WarnContext(ctx, "failed to dispatch email, retrying",
			slog.Int("attempt", attempt),
			slog.Int("max_retries", w.maxRetries),
			slog.String("to", job.ToEmail),
			slog.String("error", err.Error()),
		)

		if attempt < w.maxRetries {
			time.Sleep(time.Duration(attempt*50) * time.Millisecond)
		}
	}

	w.log.ErrorContext(ctx, "email delivery permanently failed after retries",
		slog.String("to", job.ToEmail),
		slog.String("error", err.Error()),
	)
	if w.metrics != nil {
		w.metrics.IncEmailWorkerFailure()
	}
}
