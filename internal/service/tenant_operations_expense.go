package service

import (
	"context"
	"fmt"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/dto"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ==========================================
// 8. Expense Operations
// ==========================================

func (s *TenantOperationsService) CreateExpense(ctx context.Context, tenantID uuid.UUID, role string, req *dto.CreateExpenseRequest) (*domain.TenantExpense, error) {
	if req.TotalAmount.IsNegative() {
		return nil, validationErr("total_amount", "total amount cannot be negative")
	}

	isAdvance := req.Category == "employee_advance" || (req.EmployeeID != nil && *req.EmployeeID != uuid.Nil)
	var employee *domain.TenantEmployee
	if isAdvance {
		if req.EmployeeID == nil || *req.EmployeeID == uuid.Nil {
			return nil, validationErr("employee_id", "an employee must be selected when expense type is employee advance")
		}
		emp, err := s.empRepo.GetByID(ctx, tenantID, *req.EmployeeID)
		if err != nil {
			return nil, appErrors.NewBadRequest("invalid employee for tenant")
		}
		employee = emp
		req.Category = "employee_advance"
	}

	expense := &domain.TenantExpense{
		TenantID:    tenantID,
		Item:        req.Item,
		TotalAmount: req.TotalAmount,
		Category:    req.Category,
		EmployeeID:  req.EmployeeID,
	}
	if employee != nil {
		expense.EmployeeName = employee.Name
	}

	payments := make([]domain.TenantExpensePayment, len(req.Payments))
	var cashPaid, bankPaid decimal.Decimal
	for i, p := range req.Payments {
		if p.PaymentMethod == "cash" {
			cashPaid = cashPaid.Add(p.Amount)
		} else {
			bankPaid = bankPaid.Add(p.Amount)
		}
		if p.BankID != nil && *p.BankID != uuid.Nil {
			bank, err := s.bankRepo.GetByID(ctx, tenantID, *p.BankID)
			if err != nil {
				return nil, appErrors.NewBadRequest("invalid bank account specified for expense")
			}
			p.BankName = bank.BankName
		}
		payments[i] = domain.TenantExpensePayment{
			TenantID:      tenantID,
			PaymentMethod: p.PaymentMethod,
			BankID:        p.BankID,
			BankName:      p.BankName,
			Amount:        p.Amount,
			Note:          p.Note,
		}
	}

	if len(payments) == 0 && req.TotalAmount.GreaterThan(decimal.Zero) {
		pm := req.PaymentMethod
		if pm == "" {
			pm = "cash"
		}
		if pm == "cash" {
			cashPaid = req.TotalAmount
		} else {
			bankPaid = req.TotalAmount
		}
		payments = append(payments, domain.TenantExpensePayment{
			TenantID:      tenantID,
			PaymentMethod: pm,
			BankID:        req.BankID,
			BankName:      req.BankName,
			Amount:        req.TotalAmount,
		})
	}

	err := s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.expRepo.Create(txCtx, expense, payments); err != nil {
			return err
		}

		// If this is an employee advance, record in employee_advances and update attendance & salary balance
		if isAdvance && employee != nil {
			pm := req.PaymentMethod
			if pm == "" {
				pm = "cash"
			}
			advRecord := &domain.EmployeeAdvance{
				TenantID:      tenantID,
				EmployeeID:    employee.ID,
				EmployeeName:  employee.Name,
				Amount:        req.TotalAmount,
				PaymentMethod: pm,
				ReferenceID:   req.ReferenceID,
				AdvanceDate:   todayString(),
				Notes:         req.Item,
			}
			if err := s.advRepo.Create(txCtx, advRecord); err != nil {
				return fmt.Errorf("failed to record advance ledger entry: %w", err)
			}

			// Update attendance advance for today
			att, err := s.attRepo.GetByEmployeeAndDate(txCtx, tenantID, employee.ID, todayString())
			if err == nil {
				if att == nil {
					att = &domain.Attendance{
						TenantID:   tenantID,
						EmployeeID: employee.ID,
						Date:       todayString(),
						Status:     "present",
						Advance:    req.TotalAmount,
					}
				} else {
					att.Advance = att.Advance.Add(req.TotalAmount)
				}
				_ = s.attRepo.Upsert(txCtx, att)
			}

			// Deduct from employee net salary balance
			if err := s.salaryRepo.AdjustBalance(txCtx, tenantID, employee.ID, req.TotalAmount.Neg()); err != nil {
				return fmt.Errorf("failed to adjust salary balance for advance: %w", err)
			}
			_, _ = s.salaryRepo.RecalculateBalance(txCtx, tenantID, employee.ID)

			if s.metrics != nil {
				s.metrics.IncEmployeeAdvancesCreated()
			}
		}

		if s.summaryRepo != nil {
			cashDelta := cashPaid.Neg()
			bankDelta := bankPaid.Neg()
			if err := s.summaryRepo.AdjustBalances(txCtx, tenantID, cashDelta, bankDelta, decimal.Zero, decimal.Zero); err != nil {
				return fmt.Errorf("failed to adjust financial summary for expense: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		if s.metrics != nil {
			s.metrics.IncFinancialTxFailures("create_expense")
		}
		return nil, err
	}

	s.enqueueDailyStats(ctx, tenantID, todayString())
	return expense, nil
}

func (s *TenantOperationsService) UpdateExpense(ctx context.Context, tenantID, id uuid.UUID, role string, req *dto.UpdateExpenseRequest) (*domain.TenantExpense, error) {
	exp, err := s.expRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if err := checkCurrentDayRestriction(role, exp.CreatedAt.Format("2006-01-02")); err != nil {
		return nil, err
	}

	oldAmount := exp.TotalAmount

	if req.Item != "" {
		exp.Item = req.Item
	}
	if req.TotalAmount != nil {
		exp.TotalAmount = *req.TotalAmount
	}

	deltaAmount := exp.TotalAmount.Sub(oldAmount)

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.expRepo.Update(txCtx, exp); err != nil {
			return err
		}

		if s.summaryRepo != nil && !deltaAmount.IsZero() {
			if err := s.summaryRepo.AdjustBalances(txCtx, tenantID, deltaAmount.Neg(), decimal.Zero, decimal.Zero, decimal.Zero); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if s.metrics != nil {
			s.metrics.IncFinancialTxFailures("update_expense")
		}
		return nil, err
	}

	s.enqueueDailyStats(ctx, tenantID, exp.CreatedAt.Format("2006-01-02"))
	if exp.CreatedAt.Format("2006-01-02") != todayString() {
		s.enqueueDailyStats(ctx, tenantID, todayString())
	}

	return exp, nil
}

func (s *TenantOperationsService) DeleteExpense(ctx context.Context, tenantID, id uuid.UUID, role string) error {
	exp, err := s.expRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	if err := checkCurrentDayRestriction(role, exp.CreatedAt.Format("2006-01-02")); err != nil {
		return err
	}

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.expRepo.Delete(txCtx, tenantID, id); err != nil {
			return err
		}
		if s.summaryRepo != nil {
			if err := s.summaryRepo.AdjustBalances(txCtx, tenantID, exp.TotalAmount, decimal.Zero, decimal.Zero, decimal.Zero); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if s.metrics != nil {
			s.metrics.IncFinancialTxFailures("delete_expense")
		}
		return err
	}

	s.enqueueDailyStats(ctx, tenantID, exp.CreatedAt.Format("2006-01-02"))
	if exp.CreatedAt.Format("2006-01-02") != todayString() {
		s.enqueueDailyStats(ctx, tenantID, todayString())
	}
	return nil
}

func (s *TenantOperationsService) GetExpenseByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantExpense, error) {
	return s.expRepo.GetByID(ctx, tenantID, id)
}

func (s *TenantOperationsService) ListExpenses(ctx context.Context, tenantID uuid.UUID, role string, page, pageSize int, date string, search string) ([]domain.TenantExpense, int64, error) {
	if role == auth.RoleTenantUser {
		date = todayString()
	}
	return s.expRepo.List(ctx, tenantID, page, pageSize, date, search)
}
