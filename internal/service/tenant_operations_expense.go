package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/dto"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Standard Expense Payment Error Codes
const (
	ErrCodeInvalidPaymentMethod    = "INVALID_PAYMENT_METHOD"
	ErrCodeInvalidPaymentBreakdown = "INVALID_PAYMENT_BREAKDOWN"
	ErrCodeBankAccountNotFound     = "BANK_ACCOUNT_NOT_FOUND"
	ErrCodeBankAccountNotActive    = "BANK_ACCOUNT_NOT_ACTIVE"
	ErrCodePaymentAmountMismatch   = "PAYMENT_AMOUNT_MISMATCH"
	ErrCodeDuplicateBankAccount    = "DUPLICATE_BANK_ACCOUNT"
)

// ==========================================
// 8. Expense Operations
// ==========================================

// validateAndBuildExpensePayments validates cash and multi-bank payment breakdowns
func (s *TenantOperationsService) validateAndBuildExpensePayments(
	ctx context.Context,
	tenantID uuid.UUID,
	totalAmount decimal.Decimal,
	paymentMethod string,
	cashAmount *decimal.Decimal,
	bankPayments []dto.BankPaymentSplitDTO,
	legacyPayments []dto.ExpensePaymentRequest,
	legacyBankID *uuid.UUID,
	legacyBankName string,
) (string, decimal.Decimal, []domain.TenantExpensePayment, error) {
	if totalAmount.LessThanOrEqual(decimal.Zero) {
		return "", decimal.Zero, nil, appErrors.New(ErrCodePaymentAmountMismatch, "expense total amount must be greater than zero", http.StatusBadRequest, nil)
	}

	// 1. Infer payment method if not explicitly provided
	if paymentMethod == "" {
		if len(bankPayments) > 0 && cashAmount != nil && cashAmount.GreaterThan(decimal.Zero) {
			paymentMethod = "cash_bank"
		} else if len(bankPayments) > 0 {
			paymentMethod = "bank"
		} else if legacyBankID != nil && *legacyBankID != uuid.Nil {
			paymentMethod = "bank"
		} else {
			paymentMethod = "cash"
		}
	}

	// 2. Validate payment method is supported
	if paymentMethod != "cash" && paymentMethod != "bank" && paymentMethod != "cash_bank" {
		return "", decimal.Zero, nil, appErrors.New(
			ErrCodeInvalidPaymentMethod,
			fmt.Sprintf("invalid payment method '%s', must be 'cash', 'bank', or 'cash_bank'", paymentMethod),
			http.StatusBadRequest,
			nil,
		)
	}

	// 3. Fallback: normalize legacy single-bank or legacy payments into bankPayments if empty
	if len(bankPayments) == 0 {
		if len(legacyPayments) > 0 {
			for _, lp := range legacyPayments {
				if lp.PaymentMethod == "bank" {
					var bID uuid.UUID
					if lp.BankID != nil {
						bID = *lp.BankID
					}
					bankPayments = append(bankPayments, dto.BankPaymentSplitDTO{
						BankAccountID: bID,
						BankName:      lp.BankName,
						Amount:        lp.Amount,
						Note:          lp.Note,
					})
				}
			}
		} else if paymentMethod == "bank" && legacyBankID != nil && *legacyBankID != uuid.Nil {
			bankPayments = append(bankPayments, dto.BankPaymentSplitDTO{
				BankAccountID: *legacyBankID,
				BankName:      legacyBankName,
				Amount:        totalAmount,
			})
		}
	}

	// 4. Validate breakdown according to payment method
	switch paymentMethod {
	case "cash":
		if len(bankPayments) > 0 {
			return "", decimal.Zero, nil, appErrors.New(
				ErrCodeInvalidPaymentBreakdown,
				"bank payments must be empty for cash only payment method",
				http.StatusBadRequest,
				nil,
			)
		}
		if cashAmount != nil && !cashAmount.Equal(totalAmount) {
			return "", decimal.Zero, nil, appErrors.New(
				ErrCodePaymentAmountMismatch,
				fmt.Sprintf("cash amount (%s) must equal total expense amount (%s) for cash payment", cashAmount.String(), totalAmount.String()),
				http.StatusBadRequest,
				nil,
			)
		}
		payments := []domain.TenantExpensePayment{
			{
				TenantID:      tenantID,
				PaymentMethod: "cash",
				Amount:        totalAmount,
			},
		}
		return "cash", totalAmount, payments, nil

	case "bank":
		if cashAmount != nil && cashAmount.GreaterThan(decimal.Zero) {
			return "", decimal.Zero, nil, appErrors.New(
				ErrCodeInvalidPaymentBreakdown,
				"cash amount must be zero for bank only payment method",
				http.StatusBadRequest,
				nil,
			)
		}
		if len(bankPayments) == 0 {
			return "", decimal.Zero, nil, appErrors.New(
				ErrCodeInvalidPaymentBreakdown,
				"at least one bank payment is required for bank payment method",
				http.StatusBadRequest,
				nil,
			)
		}

		payments, bankTotal, err := s.validateBankPayments(ctx, tenantID, bankPayments)
		if err != nil {
			return "", decimal.Zero, nil, err
		}

		if !bankTotal.Equal(totalAmount) {
			return "", decimal.Zero, nil, appErrors.New(
				ErrCodePaymentAmountMismatch,
				fmt.Sprintf("sum of bank payments (%s) must equal total expense amount (%s)", bankTotal.String(), totalAmount.String()),
				http.StatusBadRequest,
				nil,
			)
		}
		return "bank", decimal.Zero, payments, nil

	case "cash_bank":
		if cashAmount == nil || cashAmount.LessThanOrEqual(decimal.Zero) {
			return "", decimal.Zero, nil, appErrors.New(
				ErrCodeInvalidPaymentBreakdown,
				"cash amount must be greater than zero for cash + bank payment method",
				http.StatusBadRequest,
				nil,
			)
		}
		if len(bankPayments) == 0 {
			return "", decimal.Zero, nil, appErrors.New(
				ErrCodeInvalidPaymentBreakdown,
				"at least one bank payment is required for cash + bank payment method",
				http.StatusBadRequest,
				nil,
			)
		}

		bankRows, bankTotal, err := s.validateBankPayments(ctx, tenantID, bankPayments)
		if err != nil {
			return "", decimal.Zero, nil, err
		}

		totalPaid := cashAmount.Add(bankTotal)
		if !totalPaid.Equal(totalAmount) {
			return "", decimal.Zero, nil, appErrors.New(
				ErrCodePaymentAmountMismatch,
				fmt.Sprintf("payment breakdown total (%s) must equal total expense amount (%s)", totalPaid.String(), totalAmount.String()),
				http.StatusBadRequest,
				nil,
			)
		}

		payments := append([]domain.TenantExpensePayment{
			{
				TenantID:      tenantID,
				PaymentMethod: "cash",
				Amount:        *cashAmount,
			},
		}, bankRows...)

		return "cash_bank", *cashAmount, payments, nil

	default:
		return "", decimal.Zero, nil, appErrors.New(ErrCodeInvalidPaymentMethod, "unsupported payment method", http.StatusBadRequest, nil)
	}
}

// validateBankPayments validates each individual bank split row
func (s *TenantOperationsService) validateBankPayments(
	ctx context.Context,
	tenantID uuid.UUID,
	bankPayments []dto.BankPaymentSplitDTO,
) ([]domain.TenantExpensePayment, decimal.Decimal, error) {
	seenBanks := make(map[uuid.UUID]bool)
	var bankTotal decimal.Decimal
	payments := make([]domain.TenantExpensePayment, 0, len(bankPayments))

	for _, bp := range bankPayments {
		bID := bp.GetBankID()
		if bID == uuid.Nil {
			return nil, decimal.Zero, appErrors.New(
				ErrCodeInvalidPaymentBreakdown,
				"bank account id is required and must be a valid UUID",
				http.StatusBadRequest,
				nil,
			)
		}
		if bp.Amount.LessThanOrEqual(decimal.Zero) {
			return nil, decimal.Zero, appErrors.New(
				ErrCodeInvalidPaymentBreakdown,
				"bank payment amount must be greater than zero",
				http.StatusBadRequest,
				nil,
			)
		}
		if seenBanks[bID] {
			return nil, decimal.Zero, appErrors.New(
				ErrCodeDuplicateBankAccount,
				fmt.Sprintf("duplicate bank account selected: %s cannot appear multiple times in the same expense", bID),
				http.StatusBadRequest,
				nil,
			)
		}
		seenBanks[bID] = true

		bank, err := s.bankRepo.GetByID(ctx, tenantID, bID)
		if err != nil {
			return nil, decimal.Zero, appErrors.New(
				ErrCodeBankAccountNotFound,
				fmt.Sprintf("bank account %s not found or does not belong to tenant", bID),
				http.StatusBadRequest,
				nil,
			)
		}
		if bank.Status != "" && bank.Status != "active" {
			return nil, decimal.Zero, appErrors.New(
				ErrCodeBankAccountNotActive,
				fmt.Sprintf("bank account %s is not active", bank.BankName),
				http.StatusBadRequest,
				nil,
			)
		}

		bankIDCopy := bID
		bankName := bp.BankName
		if bankName == "" {
			bankName = bank.BankName
		}

		payments = append(payments, domain.TenantExpensePayment{
			TenantID:      tenantID,
			PaymentMethod: "bank",
			BankID:        &bankIDCopy,
			BankName:      bankName,
			Amount:        bp.Amount,
			Note:          bp.Note,
		})
		bankTotal = bankTotal.Add(bp.Amount)
	}

	return payments, bankTotal, nil
}

func (s *TenantOperationsService) CreateExpense(ctx context.Context, tenantID uuid.UUID, role string, req *dto.CreateExpenseRequest) (*domain.TenantExpense, error) {
	start := time.Now()
	totalAmount := req.GetTotalAmount()
	if totalAmount.LessThanOrEqual(decimal.Zero) {
		return nil, appErrors.New(ErrCodePaymentAmountMismatch, "total amount must be greater than zero", http.StatusBadRequest, nil)
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

	paymentMethod, cashPaid, payments, err := s.validateAndBuildExpensePayments(
		ctx,
		tenantID,
		totalAmount,
		req.PaymentMethod,
		req.CashAmount,
		req.BankPayments,
		req.Payments,
		req.BankID,
		req.BankName,
	)
	if err != nil {
		return nil, err
	}

	expense := &domain.TenantExpense{
		TenantID:      tenantID,
		Item:          req.Item,
		TotalAmount:   totalAmount,
		Amount:        totalAmount,
		Category:      req.Category,
		EmployeeID:    req.EmployeeID,
		PaymentMethod: paymentMethod,
	}
	if employee != nil {
		expense.EmployeeName = employee.Name
	}

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.expRepo.Create(txCtx, expense, payments); err != nil {
			return err
		}

		// If this is an employee advance, record in employee_advances and update attendance & salary balance
		if isAdvance && employee != nil {
			advMethod := paymentMethod
			if advMethod == "cash_bank" {
				advMethod = "mixed"
			}
			advRecord := &domain.EmployeeAdvance{
				TenantID:      tenantID,
				EmployeeID:    employee.ID,
				EmployeeName:  employee.Name,
				ExpenseID:     &expense.ID,
				Amount:        totalAmount,
				PaymentMethod: advMethod,
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
						Advance:    totalAmount,
					}
				} else {
					att.Advance = att.Advance.Add(totalAmount)
				}
				_ = s.attRepo.Upsert(txCtx, att)
			}

			// Deduct from employee net salary balance
			if err := s.salaryRepo.AdjustBalance(txCtx, tenantID, employee.ID, totalAmount.Neg()); err != nil {
				return fmt.Errorf("failed to adjust salary balance for advance: %w", err)
			}
			_, _ = s.salaryRepo.RecalculateBalance(txCtx, tenantID, employee.ID)

			if s.metrics != nil {
				s.metrics.IncEmployeeAdvancesCreated()
			}
		}

		// Adjust bank accounts and record individual bank ledger transactions
		var bankPaid decimal.Decimal
		for _, p := range payments {
			if p.PaymentMethod == "bank" && p.BankID != nil {
				bankPaid = bankPaid.Add(p.Amount)
				if err := s.bankRepo.AdjustBalance(txCtx, tenantID, *p.BankID, p.Amount.Neg()); err != nil {
					return fmt.Errorf("failed to deduct bank balance for %s: %w", p.BankName, err)
				}
				tx := &domain.BankTransaction{
					TenantID:        tenantID,
					BankID:          *p.BankID,
					Amount:          p.Amount,
					TransactionType: "debit",
					Reason:          "Expense: " + expense.Item,
					SaleType:        "expense",
					SaleID:          &expense.ID,
				}
				if err := s.bankRepo.CreateTransaction(txCtx, tx); err != nil {
					return fmt.Errorf("failed to record bank transaction for %s: %w", p.BankName, err)
				}
			}
		}

		// Update financial summary balances atomically
		if s.summaryRepo != nil {
			if err := s.summaryRepo.AdjustBalances(txCtx, tenantID, cashPaid.Neg(), bankPaid.Neg(), decimal.Zero, decimal.Zero); err != nil {
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

	if s.metrics != nil {
		s.metrics.RecordFinancialTxDuration("create_expense", time.Since(start))
		s.metrics.IncExpenseCreated(expense.Category)
		s.metrics.RecordExpensePayment(expense.PaymentMethod, expense.TotalAmount.InexactFloat64())
	}

	s.enqueueDailyStats(ctx, tenantID, todayString())
	expense.BuildPaymentBreakdown()
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
	if req.Category != "" {
		exp.Category = req.Category
	}
	if req.EmployeeID != nil {
		exp.EmployeeID = req.EmployeeID
	}
	reqAmount := req.GetTotalAmount()
	if reqAmount != nil && !reqAmount.IsZero() {
		exp.TotalAmount = *reqAmount
		exp.Amount = *reqAmount
	}

	paymentMethod := req.PaymentMethod
	if paymentMethod == "" {
		paymentMethod = exp.PaymentMethod
	}
	if paymentMethod == "" {
		paymentMethod = "cash"
	}

	validatedMethod, newCashPaid, newPayments, err := s.validateAndBuildExpensePayments(
		ctx,
		tenantID,
		exp.TotalAmount,
		paymentMethod,
		req.CashAmount,
		req.BankPayments,
		req.Payments,
		nil,
		"",
	)
	if err != nil {
		return nil, err
	}
	exp.PaymentMethod = validatedMethod

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		// 1. Revert previous bank balances and delete previous bank transactions
		var oldCashPaid, oldBankPaid decimal.Decimal
		for _, p := range exp.Payments {
			if p.PaymentMethod == "bank" && p.BankID != nil {
				oldBankPaid = oldBankPaid.Add(p.Amount)
				if err := s.bankRepo.AdjustBalance(txCtx, tenantID, *p.BankID, p.Amount); err != nil {
					return fmt.Errorf("failed to revert bank balance for %s: %w", p.BankName, err)
				}
			} else if p.PaymentMethod == "cash" {
				oldCashPaid = oldCashPaid.Add(p.Amount)
			}
		}
		if len(exp.Payments) == 0 {
			oldCashPaid = oldAmount
		}

		if err := s.bankRepo.DeleteTransactionsBySaleID(txCtx, tenantID, id); err != nil {
			return fmt.Errorf("failed to remove old bank transactions for expense: %w", err)
		}

		// 2. Revert previous summary balances
		if s.summaryRepo != nil {
			if err := s.summaryRepo.AdjustBalances(txCtx, tenantID, oldCashPaid, oldBankPaid, decimal.Zero, decimal.Zero); err != nil {
				return fmt.Errorf("failed to revert financial summary: %w", err)
			}
		}

		// 3. Update expense and persist updated payments
		if err := s.expRepo.UpdateWithPayments(txCtx, exp, newPayments); err != nil {
			return err
		}

		// 4. Apply new bank deductions and create new bank transactions
		var newBankPaid decimal.Decimal
		for _, p := range newPayments {
			if p.PaymentMethod == "bank" && p.BankID != nil {
				newBankPaid = newBankPaid.Add(p.Amount)
				if err := s.bankRepo.AdjustBalance(txCtx, tenantID, *p.BankID, p.Amount.Neg()); err != nil {
					return fmt.Errorf("failed to deduct bank balance for %s: %w", p.BankName, err)
				}
				tx := &domain.BankTransaction{
					TenantID:        tenantID,
					BankID:          *p.BankID,
					Amount:          p.Amount,
					TransactionType: "debit",
					Reason:          "Expense: " + exp.Item,
					SaleType:        "expense",
					SaleID:          &exp.ID,
				}
				if err := s.bankRepo.CreateTransaction(txCtx, tx); err != nil {
					return fmt.Errorf("failed to record bank transaction for %s: %w", p.BankName, err)
				}
			}
		}

		// 5. Apply new financial summary balances
		if s.summaryRepo != nil {
			if err := s.summaryRepo.AdjustBalances(txCtx, tenantID, newCashPaid.Neg(), newBankPaid.Neg(), decimal.Zero, decimal.Zero); err != nil {
				return fmt.Errorf("failed to apply updated financial summary: %w", err)
			}
		}

		// 6. If employee advance, adjust salary balance for delta
		deltaAmount := exp.TotalAmount.Sub(oldAmount)
		if exp.Category == "employee_advance" && exp.EmployeeID != nil && !deltaAmount.IsZero() {
			if err := s.salaryRepo.AdjustBalance(txCtx, tenantID, *exp.EmployeeID, deltaAmount.Neg()); err != nil {
				return fmt.Errorf("failed to adjust salary balance for updated advance: %w", err)
			}
			_, _ = s.salaryRepo.RecalculateBalance(txCtx, tenantID, *exp.EmployeeID)
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

	exp.BuildPaymentBreakdown()
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
		var cashPaid, bankPaid decimal.Decimal
		for _, p := range exp.Payments {
			if p.PaymentMethod == "bank" && p.BankID != nil {
				bankPaid = bankPaid.Add(p.Amount)
				if err := s.bankRepo.AdjustBalance(txCtx, tenantID, *p.BankID, p.Amount); err != nil {
					return fmt.Errorf("failed to revert bank balance for %s: %w", p.BankName, err)
				}
			} else if p.PaymentMethod == "cash" {
				cashPaid = cashPaid.Add(p.Amount)
			}
		}
		if len(exp.Payments) == 0 {
			cashPaid = exp.TotalAmount
		}

		if err := s.bankRepo.DeleteTransactionsBySaleID(txCtx, tenantID, id); err != nil {
			return fmt.Errorf("failed to delete bank transactions for expense: %w", err)
		}

		if err := s.expRepo.Delete(txCtx, tenantID, id); err != nil {
			return err
		}

		if s.summaryRepo != nil {
			if err := s.summaryRepo.AdjustBalances(txCtx, tenantID, cashPaid, bankPaid, decimal.Zero, decimal.Zero); err != nil {
				return err
			}
		}

		if exp.Category == "employee_advance" && exp.EmployeeID != nil {
			if err := s.salaryRepo.AdjustBalance(txCtx, tenantID, *exp.EmployeeID, exp.TotalAmount); err != nil {
				return fmt.Errorf("failed to restore salary balance for deleted advance: %w", err)
			}
			_, _ = s.salaryRepo.RecalculateBalance(txCtx, tenantID, *exp.EmployeeID)
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
	if s.expRepo == nil {
		return nil, appErrors.NewNotFound("expense repository not configured")
	}
	exp, err := s.expRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	exp.BuildPaymentBreakdown()
	return exp, nil
}

func (s *TenantOperationsService) ListExpenses(ctx context.Context, tenantID uuid.UUID, role string, page, pageSize int, date string, search string) ([]domain.TenantExpense, int64, error) {
	if s.expRepo == nil {
		return []domain.TenantExpense{}, 0, nil
	}
	if role == auth.RoleTenantUser {
		date = todayString()
	}
	expenses, total, err := s.expRepo.List(ctx, tenantID, page, pageSize, date, search)
	if err != nil {
		return nil, 0, err
	}
	for i := range expenses {
		expenses[i].BuildPaymentBreakdown()
	}
	return expenses, total, nil
}
