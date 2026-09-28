package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/dto"
	"github.com/Varunjp/vyavsa/internal/repository"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PlatformPlanService defines business logic for managing subscription plans
type PlatformPlanService interface {
	CreatePlan(ctx context.Context, req dto.CreatePlanRequest) (*dto.PlanResponse, error)
	GetPlanByID(ctx context.Context, id uuid.UUID) (*dto.PlanResponse, error)
	ListPlans(ctx context.Context, page, pageSize int, status string) ([]dto.PlanResponse, int64, error)
	UpdatePlan(ctx context.Context, id uuid.UUID, req dto.UpdatePlanRequest) (*dto.PlanResponse, error)
	ArchivePlan(ctx context.Context, id uuid.UUID) error
}

type platformPlanService struct {
	planRepo repository.PlatformPlanRepository
	log      *slog.Logger
}

// NewPlatformPlanService creates a new PlatformPlanService instance
func NewPlatformPlanService(planRepo repository.PlatformPlanRepository, log *slog.Logger) PlatformPlanService {
	return &platformPlanService{
		planRepo: planRepo,
		log:      log,
	}
}

func (s *platformPlanService) CreatePlan(ctx context.Context, req dto.CreatePlanRequest) (*dto.PlanResponse, error) {
	if req.Price.LessThan(decimal.Zero) {
		return nil, appErrors.NewValidation("invalid plan data", map[string]string{"price": "plan price cannot be negative"})
	}

	status := req.Status
	if status == "" {
		status = "active"
	}

	// Check for unique plan name
	existing, err := s.planRepo.GetByName(ctx, req.PlanName)
	if err == nil && existing != nil {
		return nil, appErrors.NewConflict(fmt.Sprintf("plan with name '%s' already exists", req.PlanName))
	} else if err != nil && !appErrors.IsNotFound(err) {
		return nil, err
	}

	plan := &domain.PlatformPlan{
		PlanName: req.PlanName,
		Note:     req.Note,
		Price:    req.Price,
		Status:   status,
	}

	if err := s.planRepo.Create(ctx, plan); err != nil {
		s.log.ErrorContext(ctx, "failed to create platform plan", slog.String("error", err.Error()))
		return nil, err
	}

	s.log.InfoContext(ctx, "platform plan created",
		slog.String("plan_id", plan.ID.String()),
		slog.String("plan_name", plan.PlanName),
		slog.String("price", plan.Price.String()),
	)

	resp := dto.ToPlanResponse(plan)
	return &resp, nil
}

func (s *platformPlanService) GetPlanByID(ctx context.Context, id uuid.UUID) (*dto.PlanResponse, error) {
	plan, err := s.planRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	resp := dto.ToPlanResponse(plan)
	return &resp, nil
}

func (s *platformPlanService) ListPlans(ctx context.Context, page, pageSize int, status string) ([]dto.PlanResponse, int64, error) {
	plans, total, err := s.planRepo.List(ctx, page, pageSize, status)
	if err != nil {
		return nil, 0, err
	}

	return dto.ToPlanListResponse(plans), total, nil
}

func (s *platformPlanService) UpdatePlan(ctx context.Context, id uuid.UUID, req dto.UpdatePlanRequest) (*dto.PlanResponse, error) {
	if req.Price.LessThan(decimal.Zero) {
		return nil, appErrors.NewValidation("invalid plan data", map[string]string{"price": "plan price cannot be negative"})
	}

	plan, err := s.planRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if another plan already has this name
	existing, err := s.planRepo.GetByName(ctx, req.PlanName)
	if err == nil && existing != nil && existing.ID != id {
		return nil, appErrors.NewConflict(fmt.Sprintf("another plan with name '%s' already exists", req.PlanName))
	} else if err != nil && !appErrors.IsNotFound(err) {
		return nil, err
	}

	plan.PlanName = req.PlanName
	plan.Note = req.Note
	plan.Price = req.Price
	plan.Status = req.Status

	if err := s.planRepo.Update(ctx, plan); err != nil {
		s.log.ErrorContext(ctx, "failed to update platform plan",
			slog.String("plan_id", id.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	s.log.InfoContext(ctx, "platform plan updated",
		slog.String("plan_id", id.String()),
		slog.String("plan_name", plan.PlanName),
	)

	resp := dto.ToPlanResponse(plan)
	return &resp, nil
}

func (s *platformPlanService) ArchivePlan(ctx context.Context, id uuid.UUID) error {
	if err := s.planRepo.Archive(ctx, id); err != nil {
		if appErrors.IsNotFound(err) {
			return err
		}
		s.log.ErrorContext(ctx, "failed to archive platform plan",
			slog.String("plan_id", id.String()),
			slog.String("error", err.Error()),
		)
		return err
	}

	s.log.InfoContext(ctx, "platform plan archived", slog.String("plan_id", id.String()))
	return nil
}
