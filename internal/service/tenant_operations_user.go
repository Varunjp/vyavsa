package service

import (
	"context"
	"fmt"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/dto"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
)

// ==========================================
// 1. Tenant User Management (Admin only)
// ==========================================

func (s *TenantOperationsService) CreateTenantUser(ctx context.Context, tenantID uuid.UUID, req *dto.CreateTenantUserRequest) (*domain.TenantUser, error) {
	existing, _ := s.userRepo.GetByTenantAndEmail(ctx, tenantID, req.Email)
	if existing != nil {
		return nil, appErrors.NewConflict("user with this email already exists within organization")
	}

	hash, err := s.hasher.Hash(req.Password)
	if err != nil {
		return nil, appErrors.NewInternal(fmt.Errorf("failed to hash password during user creation: %w", err))
	}

	status := req.Status
	if status == "" {
		status = "active"
	}

	user := &domain.TenantUser{
		TenantID:     tenantID,
		Name:         req.Name,
		Role:         req.Role,
		Email:        req.Email,
		PasswordHash: hash,
		Status:       status,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *TenantOperationsService) UpdateTenantUser(ctx context.Context, tenantID, id uuid.UUID, req *dto.UpdateTenantUserRequest) (*domain.TenantUser, error) {
	user, err := s.userRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Role != "" {
		user.Role = req.Role
	}
	if req.Status != "" {
		user.Status = req.Status
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	if req.Password != "" {
		hash, err := s.hasher.Hash(req.Password)
		if err != nil {
			return nil, appErrors.NewInternal(fmt.Errorf("failed to hash new password during user update: %w", err))
		}
		if err := s.userRepo.UpdatePassword(ctx, tenantID, id, hash); err != nil {
			return nil, err
		}
	}

	return user, nil
}

func (s *TenantOperationsService) DeleteTenantUser(ctx context.Context, tenantID, currentUserID, id uuid.UUID) error {
	if currentUserID == id {
		return appErrors.NewBadRequest("cannot delete your own active administrator account")
	}
	return s.userRepo.Delete(ctx, tenantID, id)
}

func (s *TenantOperationsService) GetTenantUserByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantUser, error) {
	return s.userRepo.GetByID(ctx, tenantID, id)
}

func (s *TenantOperationsService) ListTenantUsers(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search, role, status string) ([]domain.TenantUser, int64, error) {
	return s.userRepo.List(ctx, tenantID, page, pageSize, search, role, status)
}
