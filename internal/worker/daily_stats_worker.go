package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Varunjp/vyavsa/internal/cache"
	"github.com/Varunjp/vyavsa/internal/metrics"
	"github.com/Varunjp/vyavsa/internal/repository"
	"github.com/google/uuid"
)

const (
	// DailyStatsQueueKey is the Redis list key for asynchronous daily statistics jobs
	DailyStatsQueueKey = "vyavsa:queue:daily_stats"
)

// DailyStatsJob represents a payload to compute and sync daily stats for a tenant
type DailyStatsJob struct {
	TenantID uuid.UUID `json:"tenant_id"`
	Date     string    `json:"date"`
}

// DailyStatsEnqueuer defines contract for enqueueing daily stats calculation
type DailyStatsEnqueuer interface {
	Enqueue(ctx context.Context, tenantID uuid.UUID, date string) error
}

// DailyStatsWorker processes daily stats asynchronously with idempotency and retries
type DailyStatsWorker struct {
	statsRepo    repository.TenantDailyStatsRepository
	redis        *cache.Redis
	metrics      *metrics.Metrics
	log          *slog.Logger
	inMemoryChan chan DailyStatsJob
	stopCh       chan struct{}
	wg           sync.WaitGroup
	maxRetries   int
}

// NewDailyStatsWorker initializes a background daily stats worker
func NewDailyStatsWorker(
	statsRepo repository.TenantDailyStatsRepository,
	redis *cache.Redis,
	m *metrics.Metrics,
	log *slog.Logger,
	maxRetries int,
) *DailyStatsWorker {
	if maxRetries <= 0 {
		maxRetries = 3
	}
	if log == nil {
		log = slog.Default()
	}
	return &DailyStatsWorker{
		statsRepo:    statsRepo,
		redis:        redis,
		metrics:      m,
		log:          log,
		inMemoryChan: make(chan DailyStatsJob, 2048),
		stopCh:       make(chan struct{}),
		maxRetries:   maxRetries,
	}
}

// Enqueue submits a daily stats job to Redis queue or in-memory fallback
func (w *DailyStatsWorker) Enqueue(ctx context.Context, tenantID uuid.UUID, date string) error {
	job := DailyStatsJob{TenantID: tenantID, Date: date}

	if w.redis != nil && w.redis.Client != nil {
		data, err := json.Marshal(job)
		if err == nil {
			err = w.redis.Client.LPush(ctx, DailyStatsQueueKey, data).Err()
			if err == nil {
				return nil
			}
			w.log.WarnContext(ctx, "failed to enqueue daily stats to redis, falling back to memory queue",
				slog.String("error", err.Error()),
				slog.String("tenant_id", tenantID.String()),
			)
		}
	}

	select {
	case w.inMemoryChan <- job:
		return nil
	default:
		w.log.ErrorContext(ctx, "in-memory daily stats queue is full, dropping job",
			slog.String("tenant_id", tenantID.String()),
			slog.String("date", date),
		)
		if w.metrics != nil {
			w.metrics.IncDailyStatsWorkerFailure()
		}
		return fmt.Errorf("daily stats worker queue is full")
	}
}

// EnqueueDailyStats is an alias for Enqueue
func (w *DailyStatsWorker) EnqueueDailyStats(ctx context.Context, tenantID uuid.UUID, date string) error {
	return w.Enqueue(ctx, tenantID, date)
}

// Start begins processing background jobs
func (w *DailyStatsWorker) Start(ctx context.Context) {
	w.wg.Add(1)
	go w.run(ctx)
}

// Stop signals graceful shutdown and waits for workers to finish
func (w *DailyStatsWorker) Stop() {
	close(w.stopCh)
	w.wg.Wait()
}

func (w *DailyStatsWorker) run(ctx context.Context) {
	defer w.wg.Done()
	w.log.Info("starting daily stats background worker")

	for {
		select {
		case <-w.stopCh:
			w.drainRemaining(context.Background())
			w.log.Info("daily stats background worker stopped gracefully")
			return
		default:
		}

		job, ok := w.fetchNextJob(ctx)
		if !ok {
			select {
			case <-w.stopCh:
				w.drainRemaining(context.Background())
				w.log.Info("daily stats background worker stopped gracefully")
				return
			case <-time.After(50 * time.Millisecond):
			}
			continue
		}

		w.processWithRetry(ctx, job)
	}
}

func (w *DailyStatsWorker) fetchNextJob(ctx context.Context) (DailyStatsJob, bool) {
	if w.redis != nil && w.redis.Client != nil {
		res, err := w.redis.Client.RPop(ctx, DailyStatsQueueKey).Result()
		if err == nil && res != "" {
			var job DailyStatsJob
			if err := json.Unmarshal([]byte(res), &job); err == nil {
				return job, true
			}
		}
	}

	select {
	case job := <-w.inMemoryChan:
		return job, true
	default:
		return DailyStatsJob{}, false
	}
}

func (w *DailyStatsWorker) drainRemaining(ctx context.Context) {
	for {
		select {
		case job := <-w.inMemoryChan:
			w.processWithRetry(ctx, job)
		default:
			return
		}
	}
}

func (w *DailyStatsWorker) processWithRetry(ctx context.Context, job DailyStatsJob) {
	if w.statsRepo == nil {
		w.log.WarnContext(ctx, "daily stats repo is nil, skipping job", slog.String("date", job.Date))
		return
	}
	var err error
	for attempt := 1; attempt <= w.maxRetries; attempt++ {
		procCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		_, err = w.statsRepo.ComputeAndSyncDailyStats(procCtx, job.TenantID, job.Date)
		cancel()

		if err == nil {
			if w.metrics != nil {
				w.metrics.IncDailyStatsWorkerJob()
			}
			return
		}

		w.log.WarnContext(ctx, "daily stats calculation attempt failed",
			slog.Int("attempt", attempt),
			slog.Int("max_retries", w.maxRetries),
			slog.String("tenant_id", job.TenantID.String()),
			slog.String("date", job.Date),
			slog.String("error", err.Error()),
		)

		if attempt < w.maxRetries {
			time.Sleep(time.Duration(attempt*50) * time.Millisecond)
		}
	}

	w.log.ErrorContext(ctx, "daily stats calculation permanently failed after retries",
		slog.String("tenant_id", job.TenantID.String()),
		slog.String("date", job.Date),
		slog.String("error", err.Error()),
	)
	if w.metrics != nil {
		w.metrics.IncDailyStatsWorkerFailure()
	}
}
