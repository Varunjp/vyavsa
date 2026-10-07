package service

import (
	"context"
	"fmt"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/dto"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ==========================================
// 10. Salary Operations (Admin only)
// ==========================================

func (s *TenantOperationsService) ListSalaries(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search string) ([]domain.EmployeeSalary, int64, error) {
	return s.salaryRepo.List(ctx, tenantID, page, pageSize, search)
}

func (s *TenantOperationsService) ListPendingSalaries(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]domain.EmployeeSalary, int64, error) {
	return s.salaryRepo.ListPending(ctx, tenantID, page, pageSize)
}

func (s *TenantOperationsService) GetSalaryByEmployeeID(ctx context.Context, tenantID, employeeID uuid.UUID) (*domain.EmployeeSalary, error) {
	return s.salaryRepo.GetByEmployeeID(ctx, tenantID, employeeID)
}

func (s *TenantOperationsService) UpdateSalaryBalance(ctx context.Context, tenantID, employeeID uuid.UUID, req *dto.UpdateSalaryBalanceRequest) (*domain.EmployeeSalary, error) {
	_, err := s.empRepo.GetByID(ctx, tenantID, employeeID)
	if err != nil {
		return nil, appErrors.NewBadRequest("invalid employee for tenant")
	}

	salary := &domain.EmployeeSalary{
		TenantID:   tenantID,
		EmployeeID: employeeID,
		Balance:    req.Balance,
	}

	if err := s.salaryRepo.UpsertBalance(ctx, salary); err != nil {
		return nil, err
	}

	return s.salaryRepo.GetByEmployeeID(ctx, tenantID, employeeID)
}

func (s *TenantOperationsService) PaySalary(ctx context.Context, tenantID, employeeID uuid.UUID, req *dto.PaySalaryRequest) (*domain.EmployeeSalaryPayment, error) {
	if req.Amount.IsNegative() || req.Amount.IsZero() {
		return nil, validationErr("amount", "salary payment amount must be greater than zero")
	}

	emp, err := s.empRepo.GetByID(ctx, tenantID, employeeID)
	if err != nil {
		return nil, appErrors.NewBadRequest("invalid employee for tenant")
	}

	if req.BankID != nil && *req.BankID != uuid.Nil {
		bank, err := s.bankRepo.GetByID(ctx, tenantID, *req.BankID)
		if err != nil {
			return nil, appErrors.NewBadRequest("invalid bank account specified for salary disbursement")
		}
		req.BankName = bank.BankName
	}

	pDate := req.PaymentDate
	if pDate == "" {
		pDate = todayString()
	}

	payment := &domain.EmployeeSalaryPayment{
		TenantID:      tenantID,
		EmployeeID:    employeeID,
		EmployeeName:  emp.Name,
		PaymentMethod: req.PaymentMethod,
		BankID:        req.BankID,
		BankName:      req.BankName,
		Amount:        req.Amount,
		PaymentDate:   pDate,
		ReferenceID:   req.ReferenceID,
		Status:        "COMPLETED",
		Note:          req.Note,
	}

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		// Acquire row-level lock on employee salary to prevent double payments
		currentSalary, err := s.salaryRepo.GetByEmployeeIDForUpdate(txCtx, tenantID, employeeID)
		if err != nil {
			return err
		}
		if currentSalary.Balance.LessThanOrEqual(decimal.Zero) {
			return appErrors.NewBadRequest(fmt.Sprintf("employee '%s' has no outstanding salary balance to pay", emp.Name))
		}
		if req.Amount.GreaterThan(currentSalary.Balance) {
			return appErrors.NewBadRequest(fmt.Sprintf("payment amount (₹%s) exceeds outstanding salary balance (₹%s)", req.Amount.StringFixed(2), currentSalary.Balance.StringFixed(2)))
		}

		if err := s.salaryRepo.CreatePayment(txCtx, payment); err != nil {
			return err
		}

		// Deduct payment amount from employee salary balance
		if err := s.salaryRepo.AdjustBalance(txCtx, tenantID, employeeID, req.Amount.Neg()); err != nil {
			return err
		}
		_, _ = s.salaryRepo.RecalculateBalance(txCtx, tenantID, employeeID)

		// Update financial summary
		if s.summaryRepo != nil {
			var cashDelta, bankDelta decimal.Decimal
			if req.PaymentMethod == "cash" {
				cashDelta = req.Amount.Neg()
			} else {
				bankDelta = req.Amount.Neg()
			}
			if err := s.summaryRepo.AdjustBalances(txCtx, tenantID, cashDelta, bankDelta, decimal.Zero, decimal.Zero); err != nil {
				return fmt.Errorf("failed to adjust financial summary for salary payment: %w", err)
			}
		}

		if s.metrics != nil {
			s.metrics.IncSalaryPayments()
		}
		return nil
	})
	if err != nil {
		if s.metrics != nil {
			s.metrics.IncFinancialTxFailures("pay_salary")
		}
		return nil, err
	}

	s.enqueueDailyStats(ctx, tenantID, todayString())
	return payment, nil
}

func (s *TenantOperationsService) ListSalaryPayments(ctx context.Context, tenantID uuid.UUID, employeeID *uuid.UUID, page, pageSize int) ([]domain.EmployeeSalaryPayment, int64, error) {
	return s.salaryRepo.ListPayments(ctx, tenantID, employeeID, page, pageSize)
}

func (s *TenantOperationsService) GetSalaryStatement(ctx context.Context, tenantID, employeeID uuid.UUID) (*domain.EmployeeSalaryStatement, error) {
	return s.salaryRepo.GetStatement(ctx, tenantID, employeeID)
}
