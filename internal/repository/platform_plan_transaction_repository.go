package repository

import (
	"context"
	"time"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// MonthlyRevenueAggregate holds monthly total revenue and count of transactions
type MonthlyRevenueAggregate struct {
	MonthKey   string // "2026-04"
	MonthLabel string // "Apr 2026"
	Revenue    decimal.Decimal
	Count      int64
}

// PlatformPlanTransactionRepository defines persistence contracts for SaaS plan payments and transactions
type PlatformPlanTransactionRepository interface {
	Create(ctx context.Context, tx *domain.PlatformPlanTransaction) error
	ListByTenantID(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]domain.PlatformPlanTransaction, int64, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.PlatformPlanTransaction, error)
	GetByTransactionID(ctx context.Context, tenantID uuid.UUID, transactionID string) (*domain.PlatformPlanTransaction, error)
	ListAll(ctx context.Context, page, pageSize int, status string) ([]domain.PlatformPlanTransactionWithTenant, int64, error)
	GetMonthlyReceivedIncome(ctx context.Context, since time.Time) (decimal.Decimal, error)
	GetMonthlyRevenueTrend(ctx context.Context, since time.Time) ([]MonthlyRevenueAggregate, error)
}
