package worker_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/worker"
	"github.com/google/uuid"
)

type mockDailyStatsRepo struct {
	mu           sync.Mutex
	calls        int32
	failUntil    int32
	syncedDates  []string
	syncedTenant []uuid.UUID
}

func (m *mockDailyStatsRepo) ComputeAndSyncDailyStats(ctx context.Context, tenantID uuid.UUID, date string) (*domain.TenantDailyStats, error) {
	call := atomic.AddInt32(&m.calls, 1)
	if call <= m.failUntil {
		return nil, errors.New("simulated database transient error")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.syncedDates = append(m.syncedDates, date)
	m.syncedTenant = append(m.syncedTenant, tenantID)
	return &domain.TenantDailyStats{TenantID: tenantID, Date: date}, nil
}

func (m *mockDailyStatsRepo) GetByDate(ctx context.Context, tenantID uuid.UUID, date string) (*domain.TenantDailyStats, error) {
	return nil, nil
}

func (m *mockDailyStatsRepo) GetLatestAvailable(ctx context.Context, tenantID uuid.UUID, beforeDate string) (*domain.TenantDailyStats, error) {
	return nil, nil
}

func (m *mockDailyStatsRepo) Upsert(ctx context.Context, stats *domain.TenantDailyStats) error {
	return nil
}

func TestDailyStatsWorker_SuccessAndIdempotency(t *testing.T) {
	repoMock := &mockDailyStatsRepo{}
	w := worker.NewDailyStatsWorker(repoMock, nil, nil, nil, 3)

	ctx := context.Background()
	w.Start(ctx)
	defer w.Stop()

	tenantID := uuid.New()
	date := "2026-10-06"

	// Enqueue twice for same date (simulating multiple rapid financial txs on the same day)
	if err := w.EnqueueDailyStats(ctx, tenantID, date); err != nil {
		t.Fatalf("unexpected enqueue error: %v", err)
	}
	if err := w.EnqueueDailyStats(ctx, tenantID, date); err != nil {
		t.Fatalf("unexpected enqueue error: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		repoMock.mu.Lock()
		count := len(repoMock.syncedDates)
		repoMock.mu.Unlock()
		if count >= 2 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	repoMock.mu.Lock()
	defer repoMock.mu.Unlock()
	if len(repoMock.syncedDates) != 2 {
		t.Fatalf("expected 2 executions, got %d", len(repoMock.syncedDates))
	}
	for _, d := range repoMock.syncedDates {
		if d != date {
			t.Errorf("expected date %s, got %s", date, d)
		}
	}
}

func TestDailyStatsWorker_Retry(t *testing.T) {
	repoMock := &mockDailyStatsRepo{failUntil: 1}
	w := worker.NewDailyStatsWorker(repoMock, nil, nil, nil, 3)

	ctx := context.Background()
	w.Start(ctx)
	defer w.Stop()

	tenantID := uuid.New()
	date := "2026-10-06"

	if err := w.EnqueueDailyStats(ctx, tenantID, date); err != nil {
		t.Fatalf("unexpected enqueue error: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&repoMock.calls) >= 2 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if atomic.LoadInt32(&repoMock.calls) < 2 {
		t.Fatalf("expected at least 2 attempts for retry, got %d", atomic.LoadInt32(&repoMock.calls))
	}

	repoMock.mu.Lock()
	defer repoMock.mu.Unlock()
	if len(repoMock.syncedDates) != 1 {
		t.Fatalf("expected eventual sync, synced count: %d", len(repoMock.syncedDates))
	}
}

func TestDailyStatsWorker_GracefulShutdown(t *testing.T) {
	repoMock := &mockDailyStatsRepo{}
	w := worker.NewDailyStatsWorker(repoMock, nil, nil, nil, 3)

	ctx := context.Background()
	w.Start(ctx)

	tenantID := uuid.New()
	for i := 0; i < 4; i++ {
		_ = w.EnqueueDailyStats(ctx, tenantID, "2026-10-06")
	}

	done := make(chan struct{})
	go func() {
		w.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("worker shutdown timed out")
	}

	repoMock.mu.Lock()
	defer repoMock.mu.Unlock()
	if len(repoMock.syncedDates) != 4 {
		t.Fatalf("expected all 4 jobs drained on shutdown, got %d", len(repoMock.syncedDates))
	}
}
