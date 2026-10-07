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
// 9. Attendance & Overtime & Advances
// ==========================================

func (s *TenantOperationsService) RecordAttendance(ctx context.Context, tenantID uuid.UUID, role string, req *dto.RecordAttendanceRequest) (*domain.Attendance, error) {
	emp, err := s.empRepo.GetByID(ctx, tenantID, req.EmployeeID)
	if err != nil {
		return nil, appErrors.NewBadRequest("invalid employee for tenant")
	}

	attDate := req.Date
	if attDate == "" {
		attDate = todayString()
	}

	if err := checkCurrentDayRestriction(role, attDate); err != nil {
		return nil, err
	}

	// Fetch existing record to calculate salary and advance deltas, ensuring idempotent updates
	existing, err := s.attRepo.GetByEmployeeAndDate(ctx, tenantID, req.EmployeeID, attDate)
	if err != nil {
		return nil, err
	}

	var oldDailySalary decimal.Decimal
	var oldAdvance decimal.Decimal
	if existing != nil {
		oldDailySalary = existing.DailySalary
		oldAdvance = existing.Advance
	}

	var newDailySalary decimal.Decimal
	if req.Status == "present" {
		newDailySalary = emp.Salary
	} else if req.Status == "half_day" {
		newDailySalary = emp.Salary.Div(decimal.NewFromInt(2))
	} else {
		newDailySalary = decimal.Zero // absent, leave
	}

	var oldOTAmount decimal.Decimal
	if existing != nil {
		oldOTAmount = existing.OTAmount
	}

	newOTAmount := req.OTAmount
	if newOTAmount.IsZero() && req.OT.GreaterThan(decimal.Zero) {
		newOTAmount = req.OT.Mul(emp.OTRate)
	}
	otDelta := newOTAmount.Sub(oldOTAmount)

	salaryDelta := newDailySalary.Sub(oldDailySalary)
	advanceDelta := req.Advance.Sub(oldAdvance)

	att := &domain.Attendance{
		TenantID:    tenantID,
		EmployeeID:  req.EmployeeID,
		Date:        attDate,
		Status:      req.Status,
		DailySalary: newDailySalary,
		OT:          req.OT,
		OTAmount:    newOTAmount,
		Advance:     req.Advance,
	}

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.attRepo.Upsert(txCtx, att); err != nil {
			return err
		}

		// Handle overtime delta
		if !otDelta.IsZero() {
			if otDelta.GreaterThan(decimal.Zero) {
				otRecord := &domain.EmployeeOvertime{
					TenantID:     tenantID,
					EmployeeID:   req.EmployeeID,
					EmployeeName: emp.Name,
					Amount:       otDelta,
					OvertimeDate: attDate,
					Notes:        "Attendance overtime",
				}
				if err := s.otRepo.Create(txCtx, otRecord); err != nil {
					return fmt.Errorf("failed to create overtime transaction: %w", err)
				}
			}
			if err := s.salaryRepo.AdjustBalance(txCtx, tenantID, req.EmployeeID, otDelta); err != nil {
				return fmt.Errorf("failed to adjust employee salary balance for overtime: %w", err)
			}
		}

		// Credit/debit employee salary balance by salary delta (avoids crediting absent, handles status change/retry idempotently)
		if !salaryDelta.IsZero() {
			if err := s.salaryRepo.AdjustBalance(txCtx, tenantID, req.EmployeeID, salaryDelta); err != nil {
				return fmt.Errorf("failed to adjust employee salary balance for attendance: %w", err)
			}
		}

		// If advance delta changed, adjust cash balance and employee salary balance
		if !advanceDelta.IsZero() {
			if s.summaryRepo != nil {
				if err := s.summaryRepo.AdjustBalances(txCtx, tenantID, advanceDelta.Neg(), decimal.Zero, decimal.Zero, decimal.Zero); err != nil {
					return fmt.Errorf("failed to adjust financial summary for attendance advance: %w", err)
				}
			}
			if advanceDelta.GreaterThan(decimal.Zero) {
				advRecord := &domain.EmployeeAdvance{
					TenantID:      tenantID,
					EmployeeID:    req.EmployeeID,
					EmployeeName:  emp.Name,
					Amount:        advanceDelta,
					PaymentMethod: "cash",
					AdvanceDate:   attDate,
					Notes:         "Attendance cash advance",
				}
				if err := s.advRepo.Create(txCtx, advRecord); err != nil {
					return fmt.Errorf("failed to create advance transaction: %w", err)
				}
			}
			// Deduct advance from employee salary balance
			if err := s.salaryRepo.AdjustBalance(txCtx, tenantID, req.EmployeeID, advanceDelta.Neg()); err != nil {
				return fmt.Errorf("failed to adjust employee salary balance for advance: %w", err)
			}
		}

		_, _ = s.salaryRepo.RecalculateBalance(txCtx, tenantID, req.EmployeeID)
		return nil
	})
	if err != nil {
		if s.metrics != nil {
			s.metrics.IncFinancialTxFailures("record_attendance")
		}
		return nil, err
	}

	s.enqueueDailyStats(ctx, tenantID, attDate)
	return att, nil
}

func (s *TenantOperationsService) RecordOvertime(ctx context.Context, tenantID uuid.UUID, role string, req *dto.RecordOvertimeRequest) (*domain.Attendance, error) {
	if req.Amount.IsNegative() || req.Amount.IsZero() {
		return nil, validationErr("amount", "overtime amount must be strictly greater than zero")
	}

	emp, err := s.empRepo.GetByID(ctx, tenantID, req.EmployeeID)
	if err != nil {
		return nil, appErrors.NewBadRequest("invalid employee for tenant")
	}

	date := req.Date
	if date == "" {
		date = todayString()
	}

	if err := checkCurrentDayRestriction(role, date); err != nil {
		return nil, err
	}

	var updatedAtt *domain.Attendance
	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		// 1. Record permanent overtime transaction in employee_overtime table
		otRecord := &domain.EmployeeOvertime{
			TenantID:     tenantID,
			EmployeeID:   req.EmployeeID,
			EmployeeName: emp.Name,
			Amount:       req.Amount,
			OvertimeDate: date,
			ReferenceID:  req.ReferenceID,
			Notes:        req.Note,
		}
		if err := s.otRepo.Create(txCtx, otRecord); err != nil {
			return fmt.Errorf("failed to create overtime transaction: %w", err)
		}

		// 2. Update daily attendance record (storing direct OT amount)
		existing, err := s.attRepo.GetByEmployeeAndDate(txCtx, tenantID, req.EmployeeID, date)
		if err != nil {
			return err
		}

		if existing == nil {
			existing = &domain.Attendance{
				TenantID:   tenantID,
				EmployeeID: req.EmployeeID,
				Date:       date,
				Status:     "present",
				OT:         req.OT,
				OTAmount:   req.Amount,
				Advance:    decimal.Zero,
			}
		} else {
			if req.OT.GreaterThan(decimal.Zero) {
				existing.OT = existing.OT.Add(req.OT)
			}
			existing.OTAmount = existing.OTAmount.Add(req.Amount)
		}

		if err := s.attRepo.Upsert(txCtx, existing); err != nil {
			return err
		}
		updatedAtt = existing

		// 3. Add OT amount to employee salary balance
		if err := s.salaryRepo.AdjustBalance(txCtx, tenantID, req.EmployeeID, req.Amount); err != nil {
			return fmt.Errorf("failed to adjust salary balance for overtime: %w", err)
		}
		_, _ = s.salaryRepo.RecalculateBalance(txCtx, tenantID, req.EmployeeID)

		if s.metrics != nil {
			s.metrics.IncOvertimeCreated()
		}
		return nil
	})
	if err != nil {
		if s.metrics != nil {
			s.metrics.IncFinancialTxFailures("record_overtime")
		}
		return nil, err
	}

	s.enqueueDailyStats(ctx, tenantID, date)
	return updatedAtt, nil
}

func (s *TenantOperationsService) RecordAdvance(ctx context.Context, tenantID uuid.UUID, role string, req *dto.RecordAdvanceRequest) (*domain.Attendance, error) {
	if req.Amount.IsNegative() || req.Amount.IsZero() {
		return nil, validationErr("amount", "advance amount must be greater than zero")
	}

	emp, err := s.empRepo.GetByID(ctx, tenantID, req.EmployeeID)
	if err != nil {
		return nil, appErrors.NewBadRequest("invalid employee for tenant")
	}

	date := req.Date
	if date == "" {
		date = todayString()
	}

	if err := checkCurrentDayRestriction(role, date); err != nil {
		return nil, err
	}

	pm := req.PaymentMethod
	if pm == "" {
		pm = "cash"
	}

	var updatedAtt *domain.Attendance
	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		// 1. Record permanent transaction in employee_advances table
		adv := &domain.EmployeeAdvance{
			TenantID:      tenantID,
			EmployeeID:    req.EmployeeID,
			EmployeeName:  emp.Name,
			Amount:        req.Amount,
			PaymentMethod: pm,
			ReferenceID:   req.ReferenceID,
			AdvanceDate:   date,
			Notes:         req.Note,
		}
		if err := s.advRepo.Create(txCtx, adv); err != nil {
			return fmt.Errorf("failed to create advance transaction: %w", err)
		}

		// 2. Update daily attendance record
		existing, err := s.attRepo.GetByEmployeeAndDate(txCtx, tenantID, req.EmployeeID, date)
		if err != nil {
			return err
		}

		if existing == nil {
			existing = &domain.Attendance{
				TenantID:   tenantID,
				EmployeeID: req.EmployeeID,
				Date:       date,
				Status:     "present",
				OT:         decimal.Zero,
				OTAmount:   decimal.Zero,
				Advance:    req.Amount,
			}
		} else {
			existing.Advance = existing.Advance.Add(req.Amount)
		}

		if err := s.attRepo.Upsert(txCtx, existing); err != nil {
			return err
		}
		updatedAtt = existing

		// 3. Update financial summary: deduct cash (or bank)
		if s.summaryRepo != nil {
			var cashDelta, bankDelta decimal.Decimal
			if req.PaymentMethod == "bank" {
				bankDelta = req.Amount.Neg()
			} else {
				cashDelta = req.Amount.Neg()
			}
			if err := s.summaryRepo.AdjustBalances(txCtx, tenantID, cashDelta, bankDelta, decimal.Zero, decimal.Zero); err != nil {
				return fmt.Errorf("failed to adjust financial summary for advance: %w", err)
			}
		}

		// 4. Deduct advance from employee salary balance
		if err := s.salaryRepo.AdjustBalance(txCtx, tenantID, req.EmployeeID, req.Amount.Neg()); err != nil {
			return fmt.Errorf("failed to adjust salary balance for advance: %w", err)
		}
		_, _ = s.salaryRepo.RecalculateBalance(txCtx, tenantID, req.EmployeeID)

		if s.metrics != nil {
			s.metrics.IncEmployeeAdvancesCreated()
		}
		return nil
	})
	if err != nil {
		if s.metrics != nil {
			s.metrics.IncFinancialTxFailures("record_advance")
		}
		return nil, err
	}

	s.enqueueDailyStats(ctx, tenantID, date)
	return updatedAtt, nil
}

func (s *TenantOperationsService) UpdateAttendance(ctx context.Context, tenantID, id uuid.UUID, role string, req *dto.UpdateAttendanceRequest) (*domain.Attendance, error) {
	att, err := s.attRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if err := checkCurrentDayRestriction(role, att.Date); err != nil {
		return nil, err
	}

	if req.Status != "" {
		att.Status = req.Status
	}
	if req.OT != nil {
		att.OT = *req.OT
	}
	if req.OTAmount != nil {
		att.OTAmount = *req.OTAmount
	} else if req.OT != nil {
		emp, err := s.empRepo.GetByID(ctx, tenantID, att.EmployeeID)
		if err == nil {
			att.OTAmount = req.OT.Mul(emp.OTRate)
		}
	}
	if req.Advance != nil {
		att.Advance = *req.Advance
	}

	if err := s.attRepo.Update(ctx, att); err != nil {
		return nil, err
	}

	_, _ = s.salaryRepo.RecalculateBalance(ctx, tenantID, att.EmployeeID)
	s.enqueueDailyStats(ctx, tenantID, att.Date)
	return att, nil
}

func (s *TenantOperationsService) GetAttendanceByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Attendance, error) {
	return s.attRepo.GetByID(ctx, tenantID, id)
}

func (s *TenantOperationsService) ListAttendance(ctx context.Context, tenantID uuid.UUID, role string, page, pageSize int, date string, employeeID *uuid.UUID) ([]domain.Attendance, int64, error) {
	if role == auth.RoleTenantUser && date == "" {
		date = todayString()
	}
	return s.attRepo.List(ctx, tenantID, page, pageSize, date, employeeID)
}

func (s *TenantOperationsService) ListEmployeeAdvances(ctx context.Context, tenantID uuid.UUID, page, pageSize int, employeeID *uuid.UUID, date string) ([]domain.EmployeeAdvance, int64, error) {
	return s.advRepo.List(ctx, tenantID, page, pageSize, employeeID, date)
}

func (s *TenantOperationsService) ListEmployeeOvertime(ctx context.Context, tenantID uuid.UUID, page, pageSize int, employeeID *uuid.UUID, date string) ([]domain.EmployeeOvertime, int64, error) {
	return s.otRepo.List(ctx, tenantID, page, pageSize, employeeID, date)
}
