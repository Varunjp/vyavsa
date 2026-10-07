package service

import (
	"context"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/dto"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ==========================================
// 2. Employee Management (Admin only)
// ==========================================

func (s *TenantOperationsService) CreateEmployee(ctx context.Context, tenantID uuid.UUID, req *dto.CreateEmployeeRequest) (*domain.TenantEmployee, error) {
	if req.Salary.IsNegative() {
		return nil, validationErr("salary", "salary cannot be negative")
	}
	if req.OTRate.IsNegative() {
		return nil, validationErr("ot_rate", "ot rate cannot be negative")
	}

	status := req.Status
	if status == "" {
		status = "active"
	}

	emp := &domain.TenantEmployee{
		TenantID: tenantID,
		Name:     req.Name,
		Phone:    req.Phone,
		Salary:   req.Salary,
		OTRate:   req.OTRate,
		Status:   status,
	}

	err := s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.empRepo.Create(txCtx, emp); err != nil {
			return err
		}
		// Initialize salary balance record to zero
		return s.salaryRepo.UpsertBalance(txCtx, &domain.EmployeeSalary{
			TenantID:   tenantID,
			EmployeeID: emp.ID,
			Balance:    decimal.Zero,
		})
	})
	if err != nil {
		return nil, err
	}

	return emp, nil
}

func (s *TenantOperationsService) UpdateEmployee(ctx context.Context, tenantID, id uuid.UUID, req *dto.UpdateEmployeeRequest) (*domain.TenantEmployee, error) {
	emp, err := s.empRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		emp.Name = req.Name
	}
	if req.Phone != "" {
		emp.Phone = req.Phone
	}
	if req.Salary != nil {
		if req.Salary.IsNegative() {
			return nil, validationErr("salary", "salary cannot be negative")
		}
		emp.Salary = *req.Salary
	}
	if req.OTRate != nil {
		if req.OTRate.IsNegative() {
			return nil, validationErr("ot_rate", "ot rate cannot be negative")
		}
		emp.OTRate = *req.OTRate
	}
	if req.Status != "" {
		emp.Status = req.Status
	}

	if err := s.empRepo.Update(ctx, emp); err != nil {
		return nil, err
	}

	if req.Salary != nil {
		_, _ = s.salaryRepo.RecalculateBalance(ctx, tenantID, id)
	}

	return emp, nil
}

func (s *TenantOperationsService) DeleteEmployee(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.empRepo.Delete(ctx, tenantID, id)
}

func (s *TenantOperationsService) GetEmployeeByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantEmployee, error) {
	return s.empRepo.GetByID(ctx, tenantID, id)
}

func (s *TenantOperationsService) ListEmployees(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search, status string) ([]domain.TenantEmployee, int64, error) {
	return s.empRepo.List(ctx, tenantID, page, pageSize, search, status)
}
