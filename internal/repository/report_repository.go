package repository

import (
	"context"
	"time"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/google/uuid"
)

// DailyReportRepository defines the contract for fetching daily business financial report records
type DailyReportRepository interface {
	GetDailyReportData(ctx context.Context, tenantID uuid.UUID, start, end time.Time, loc *time.Location) (*domain.DailyReport, error)
}
