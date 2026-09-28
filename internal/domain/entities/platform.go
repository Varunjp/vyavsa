package entities

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PlatformAdmin struct {
	ID        uuid.UUID
	Username  string
	Email     string
	Phone     string
	Status    string
	Password  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type PlatformPlan struct {
	ID        uuid.UUID
	PlanName  string
	Note      string
	Price     decimal.Decimal
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type PlatformSubscriptions struct {
	ID       uuid.UUID
	TenantID uuid.UUID

	CurrentPlanID   uuid.UUID
	CurrentPlanName string

	EndDate time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

type PlatformPlanTransactions struct {
	ID       uuid.UUID
	TenantID uuid.UUID

	TransactionID string
	PaymentMethod string
	PlanID        uuid.UUID
	Amount        decimal.Decimal

	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}
