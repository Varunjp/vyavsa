package dto

import (
	"time"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CreatePlanRequest represents payload to create a new subscription plan
type CreatePlanRequest struct {
	PlanName string          `json:"plan_name" binding:"required,min=2,max=100"`
	Note     string          `json:"note"`
	Price    decimal.Decimal `json:"price"`
	Status   string          `json:"status" binding:"omitempty,oneof=active inactive archived"`
}

// UpdatePlanRequest represents payload to update an existing plan
type UpdatePlanRequest struct {
	PlanName string          `json:"plan_name" binding:"required,min=2,max=100"`
	Note     string          `json:"note"`
	Price    decimal.Decimal `json:"price"`
	Status   string          `json:"status" binding:"required,oneof=active inactive archived"`
}

// PlanResponse represents public serialization of a plan
type PlanResponse struct {
	ID        uuid.UUID       `json:"id"`
	PlanName  string          `json:"plan_name"`
	Note      string          `json:"note,omitempty"`
	Price     decimal.Decimal `json:"price"`
	Status    string          `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// ToPlanResponse converts domain.PlatformPlan to PlanResponse
func ToPlanResponse(p *domain.PlatformPlan) PlanResponse {
	return PlanResponse{
		ID:        p.ID,
		PlanName:  p.PlanName,
		Note:      p.Note,
		Price:     p.Price,
		Status:    p.Status,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

// ToPlanListResponse converts slice of domain.PlatformPlan to slice of PlanResponse
func ToPlanListResponse(plans []domain.PlatformPlan) []PlanResponse {
	resp := make([]PlanResponse, len(plans))
	for i := range plans {
		resp[i] = ToPlanResponse(&plans[i])
	}
	return resp
}
