package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/dto"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ==========================================
// 5. Line Sale Operations
// ==========================================

func (s *TenantOperationsService) CreateLineSale(ctx context.Context, tenantID uuid.UUID, role string, req *dto.CreateLineSaleRequest) (*domain.LineSale, error) {
	start := time.Now()
	if req.TotalAmount.IsNegative() {
		if s.metrics != nil {
			s.metrics.RecordSalesPaymentFailed("negative_total_amount")
		}
		return nil, validationErr("total_amount", "total amount cannot be negative")
	}

	// Verify customer belongs to tenant
	cust, err := s.custRepo.GetByID(ctx, tenantID, req.CustomerID)
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordSalesPaymentFailed("invalid_customer")
		}
		return nil, appErrors.NewBadRequest("invalid customer for tenant")
	}

	var cashReceived decimal.Decimal
	if req.CashAmount != nil {
		if req.CashAmount.IsNegative() {
			if s.metrics != nil {
				s.metrics.RecordSalesPaymentFailed("negative_cash_amount")
			}
			return nil, validationErr("cash_amount", "cash amount cannot be negative")
		}
		cashReceived = *req.CashAmount
	} else if req.TotalCashIn.IsNegative() {
		if s.metrics != nil {
			s.metrics.RecordSalesPaymentFailed("negative_cash_amount")
		}
		return nil, validationErr("total_cash_in", "cash amount cannot be negative")
	} else if req.TotalCashIn.GreaterThan(decimal.Zero) {
		cashReceived = req.TotalCashIn
	}

	// Extract raw bank splits
	var rawBankSplits []dto.BankPaymentSplitRequest
	if len(req.BankPayments) > 0 {
		rawBankSplits = req.BankPayments
	} else if len(req.Payments) > 0 {
		for _, p := range req.Payments {
			if p.Amount.IsNegative() || p.Amount.IsZero() {
				if s.metrics != nil {
					s.metrics.RecordSalesPaymentFailed("invalid_payment_amount")
				}
				return nil, validationErr("payment_amount", "payment amount must be greater than zero")
			}
			if p.PaymentMethod == "cash" {
				cashReceived = cashReceived.Add(p.Amount)
			} else if p.PaymentMethod == "bank" || p.PaymentMethod == "upi" || p.PaymentMethod == "online" || p.PaymentMethod == "cheque" || p.PaymentMethod == "other" {
				if p.BankID == nil || *p.BankID == uuid.Nil {
					if s.metrics != nil {
						s.metrics.RecordSalesPaymentFailed("missing_bank_id")
					}
					return nil, validationErr("bank_id", "bank account must be selected for bank payments")
				}
				rawBankSplits = append(rawBankSplits, dto.BankPaymentSplitRequest{
					BankID:   *p.BankID,
					BankName: p.BankName,
					Amount:   p.Amount,
					Note:     p.Note,
				})
			}
		}
	} else if req.BankAmount.GreaterThan(decimal.Zero) {
		if req.BankID == nil || *req.BankID == uuid.Nil {
			if s.metrics != nil {
				s.metrics.RecordSalesPaymentFailed("missing_bank_id")
			}
			return nil, validationErr("bank_id", "bank account must be selected when bank amount is specified")
		}
		rawBankSplits = append(rawBankSplits, dto.BankPaymentSplitRequest{
			BankID: *req.BankID,
			Amount: req.BankAmount,
		})
	}

	// Validate bank payments & tenant isolation
	seenBanks := make(map[uuid.UUID]bool)
	validatedBanks := make([]validatedBankPayment, 0, len(rawBankSplits))
	var bankReceived decimal.Decimal

	for _, bp := range rawBankSplits {
		if bp.BankID == uuid.Nil {
			if s.metrics != nil {
				s.metrics.RecordSalesPaymentFailed("empty_bank_id")
			}
			return nil, validationErr("bank_payments", "bank account must be selected for each bank payment")
		}
		if seenBanks[bp.BankID] {
			if s.metrics != nil {
				s.metrics.RecordSalesPaymentFailed("duplicate_bank")
			}
			return nil, validationErr("bank_payments", "duplicate bank account selected in payments")
		}
		seenBanks[bp.BankID] = true

		if bp.Amount.IsNegative() || bp.Amount.IsZero() {
			if s.metrics != nil {
				s.metrics.RecordSalesPaymentFailed("zero_or_negative_bank_amount")
			}
			return nil, validationErr("bank_payments", "bank payment amount must be greater than zero")
		}

		bank, err := s.bankRepo.GetByID(ctx, tenantID, bp.BankID)
		if err != nil {
			if s.metrics != nil {
				s.metrics.RecordSalesPaymentFailed("invalid_bank")
			}
			return nil, appErrors.NewBadRequest("invalid bank account specified for line sale payment")
		}
		if bank.Status != "active" {
			if s.metrics != nil {
				s.metrics.RecordSalesPaymentFailed("inactive_bank")
			}
			return nil, appErrors.NewBadRequest("selected bank account is inactive")
		}

		validatedBanks = append(validatedBanks, validatedBankPayment{
			BankID:   bank.ID,
			BankName: bank.BankName,
			Amount:   bp.Amount,
			Note:     bp.Note,
		})
		bankReceived = bankReceived.Add(bp.Amount)
	}

	collectedAmount := cashReceived.Add(bankReceived)
	if collectedAmount.GreaterThan(req.TotalAmount) {
		if s.metrics != nil {
			s.metrics.RecordSalesPaymentFailed("collected_exceeds_total")
		}
		return nil, validationErr("collected_amount", "collected amount cannot exceed sale total")
	}

	due := req.TotalAmount.Sub(collectedAmount)
	if due.IsNegative() {
		due = decimal.Zero
	}

	sale := &domain.LineSale{
		TenantID:        tenantID,
		CustomerID:      cust.ID,
		CustomerName:    cust.CustomerName,
		Route:           req.Route,
		Salesman:        req.Salesman,
		Note:            req.Note,
		TotalAmount:     req.TotalAmount,
		TotalCashIn:     cashReceived,
		BankAmount:      bankReceived,
		CollectedAmount: collectedAmount,
		Balance:         due,
	}

	payments := make([]domain.LineSalePayment, 0, len(validatedBanks)+1)
	if cashReceived.GreaterThan(decimal.Zero) {
		payments = append(payments, domain.LineSalePayment{
			TenantID:      tenantID,
			PaymentMethod: "cash",
			Amount:        cashReceived,
			Note:          "Cash payment",
		})
	}
	for _, vb := range validatedBanks {
		note := vb.Note
		if note == "" {
			note = "Bank payment"
		}
		payments = append(payments, domain.LineSalePayment{
			TenantID:      tenantID,
			PaymentMethod: "bank",
			BankID:        &vb.BankID,
			BankName:      vb.BankName,
			Amount:        vb.Amount,
			Note:          note,
		})
	}

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.lineSaleRepo.Create(txCtx, sale, payments); err != nil {
			return err
		}

		for _, vb := range validatedBanks {
			if err := s.bankRepo.AdjustBalance(txCtx, tenantID, vb.BankID, vb.Amount); err != nil {
				return fmt.Errorf("failed to adjust bank balance for bank %s: %w", vb.BankName, err)
			}
			tx := &domain.BankTransaction{
				TenantID:        tenantID,
				BankID:          vb.BankID,
				Amount:          vb.Amount,
				TransactionType: "credit",
				Reason:          "Line Sale: " + cust.CustomerName,
				SaleType:        "line_sale",
				SaleID:          &sale.ID,
			}
			if err := s.bankRepo.CreateTransaction(txCtx, tx); err != nil {
				return fmt.Errorf("failed to record bank transaction for bank %s: %w", vb.BankName, err)
			}
		}

		// Adjust customer current balance by remaining credit balance
		if due.GreaterThan(decimal.Zero) {
			if err := s.custRepo.AdjustBalance(txCtx, tenantID, cust.ID, due); err != nil {
				return err
			}
		}

		// Update financial summary atomically
		if s.summaryRepo != nil {
			if err := s.summaryRepo.AdjustBalances(txCtx, tenantID, cashReceived, bankReceived, due, decimal.Zero); err != nil {
				return fmt.Errorf("failed to update financial summary for line sale: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordSalesPaymentFailed("transaction_error")
			s.metrics.IncFinancialTxFailures("create_line_sale")
		}
		return nil, err
	}

	if s.metrics != nil {
		s.metrics.RecordFinancialTxDuration("create_line_sale", time.Since(start))
		if cashReceived.GreaterThan(decimal.Zero) {
			s.metrics.RecordSalesPayment("cash", cashReceived.InexactFloat64())
		}
		for _, vb := range validatedBanks {
			s.metrics.RecordSalesPayment("bank", vb.Amount.InexactFloat64())
		}
	}

	s.enqueueDailyStats(ctx, tenantID, todayString())

	return sale, nil
}

func (s *TenantOperationsService) UpdateLineSale(ctx context.Context, tenantID, id uuid.UUID, role string, req *dto.UpdateLineSaleRequest) (*domain.LineSale, error) {
	sale, err := s.lineSaleRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if err := checkCurrentDayRestriction(role, sale.CreatedAt.Format("2006-01-02")); err != nil {
		return nil, err
	}

	oldBalance := sale.Balance
	oldCash := sale.TotalCashIn

	if req.Route != "" {
		sale.Route = req.Route
	}
	if req.Salesman != "" {
		sale.Salesman = req.Salesman
	}
	if req.Note != "" {
		sale.Note = req.Note
	}
	if req.TotalAmount != nil {
		sale.TotalAmount = *req.TotalAmount
		sale.Balance = sale.TotalAmount.Sub(sale.TotalCashIn)
	}
	if req.TotalCashIn != nil {
		sale.TotalCashIn = *req.TotalCashIn
		sale.Balance = sale.TotalAmount.Sub(sale.TotalCashIn)
	}

	deltaBalance := sale.Balance.Sub(oldBalance)
	deltaCash := sale.TotalCashIn.Sub(oldCash)

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		// 1. Adjust customer balance by delta in credit
		if !deltaBalance.IsZero() && s.custRepo != nil {
			if err := s.custRepo.AdjustBalance(txCtx, tenantID, sale.CustomerID, deltaBalance); err != nil {
				return fmt.Errorf("failed to adjust customer balance on line sale update: %w", err)
			}
		}

		// 2. Update line sale record
		if err := s.lineSaleRepo.Update(txCtx, sale); err != nil {
			return err
		}

		// 3. Update financial summary cash and receivable balances atomically
		if s.summaryRepo != nil {
			if err := s.summaryRepo.AdjustBalances(txCtx, tenantID, deltaCash, decimal.Zero, deltaBalance, decimal.Zero); err != nil {
				return fmt.Errorf("failed to update financial summary on line sale update: %w", err)
			}
			_, _ = s.summaryRepo.SyncFromSourceRecords(txCtx, tenantID)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	s.enqueueDailyStats(ctx, tenantID, sale.CreatedAt.Format("2006-01-02"))
	if sale.CreatedAt.Format("2006-01-02") != todayString() {
		s.enqueueDailyStats(ctx, tenantID, todayString())
	}

	return sale, nil
}

func (s *TenantOperationsService) DeleteLineSale(ctx context.Context, tenantID, id uuid.UUID, role string) error {
	sale, err := s.lineSaleRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	if err := checkCurrentDayRestriction(role, sale.CreatedAt.Format("2006-01-02")); err != nil {
		return err
	}

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		// Reverse balance on customer if unpaid
		if sale.Balance.GreaterThan(decimal.Zero) && s.custRepo != nil {
			if err := s.custRepo.AdjustBalance(txCtx, tenantID, sale.CustomerID, sale.Balance.Neg()); err != nil {
				return fmt.Errorf("failed to reverse customer balance on line sale deletion: %w", err)
			}
		}
		if err := s.lineSaleRepo.Delete(txCtx, tenantID, id); err != nil {
			return err
		}
		if s.summaryRepo != nil {
			if err := s.summaryRepo.AdjustBalances(txCtx, tenantID, sale.TotalCashIn.Neg(), sale.BankAmount.Neg(), sale.Balance.Neg(), decimal.Zero); err != nil {
				return fmt.Errorf("failed to update financial summary on line sale deletion: %w", err)
			}
			_, _ = s.summaryRepo.SyncFromSourceRecords(txCtx, tenantID)
		}
		return nil
	})
	if err != nil {
		return err
	}

	s.enqueueDailyStats(ctx, tenantID, sale.CreatedAt.Format("2006-01-02"))
	if sale.CreatedAt.Format("2006-01-02") != todayString() {
		s.enqueueDailyStats(ctx, tenantID, todayString())
	}
	return nil
}

func (s *TenantOperationsService) GetLineSaleByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.LineSale, error) {
	return s.lineSaleRepo.GetByID(ctx, tenantID, id)
}

func (s *TenantOperationsService) ListLineSales(ctx context.Context, tenantID uuid.UUID, role string, page, pageSize int, date string, customerID *uuid.UUID, search string) ([]domain.LineSale, int64, error) {
	if role == auth.RoleTenantUser {
		date = todayString() // Force current-day for tenant users
	}
	return s.lineSaleRepo.List(ctx, tenantID, page, pageSize, date, customerID, search)
}

func (s *TenantOperationsService) RecordLineSalePayment(ctx context.Context, tenantID, lineSaleID uuid.UUID, role string, req *dto.RecordSalePaymentRequest) (*domain.LineSalePayment, error) {
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, validationErr("amount", "payment amount must be greater than zero")
	}
	if req.PaymentMethod != "cash" && req.PaymentMethod != "bank" {
		return nil, validationErr("payment_method", "payment method must be either 'cash' or 'bank'")
	}

	if req.PaymentMethod == "bank" && req.BankID != nil && *req.BankID != uuid.Nil {
		bank, err := s.bankRepo.GetByID(ctx, tenantID, *req.BankID)
		if err != nil {
			return nil, appErrors.NewBadRequest("invalid bank account specified for line sale payment")
		}
		req.BankName = bank.BankName
	}

	pDate := req.PaymentDate
	if pDate == "" {
		pDate = todayString()
	}

	var payment *domain.LineSalePayment
	err := s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		sale, err := s.lineSaleRepo.GetByIDForUpdate(txCtx, tenantID, lineSaleID)
		if err != nil {
			return err
		}
		if sale.Balance.LessThanOrEqual(decimal.Zero) {
			return appErrors.NewBadRequest("line sale has already been fully collected")
		}
		if req.Amount.GreaterThan(sale.Balance) {
			return appErrors.NewBadRequest(fmt.Sprintf("payment amount (₹%s) exceeds outstanding balance (₹%s)", req.Amount.StringFixed(2), sale.Balance.StringFixed(2)))
		}

		payment = &domain.LineSalePayment{
			TenantID:      tenantID,
			LineSaleID:    lineSaleID,
			PaymentMethod: req.PaymentMethod,
			BankID:        req.BankID,
			BankName:      req.BankName,
			Amount:        req.Amount,
			PaymentDate:   pDate,
			ReferenceID:   req.ReferenceID,
			Status:        "COMPLETED",
			Note:          req.Note,
		}
		if err := s.lineSaleRepo.CreatePayment(txCtx, payment); err != nil {
			return fmt.Errorf("failed to create line sale payment: %w", err)
		}

		sale.CollectedAmount = sale.CollectedAmount.Add(req.Amount)
		if req.PaymentMethod == "cash" {
			sale.TotalCashIn = sale.TotalCashIn.Add(req.Amount)
		} else {
			sale.BankAmount = sale.BankAmount.Add(req.Amount)
		}
		sale.Balance = sale.Balance.Sub(req.Amount)
		if err := s.lineSaleRepo.Update(txCtx, sale); err != nil {
			return fmt.Errorf("failed to update line sale balance: %w", err)
		}

		if s.summaryRepo != nil {
			var cashDelta, bankDelta decimal.Decimal
			if req.PaymentMethod == "cash" {
				cashDelta = req.Amount
			} else {
				bankDelta = req.Amount
			}
			if err := s.summaryRepo.AdjustBalances(txCtx, tenantID, cashDelta, bankDelta, decimal.Zero, decimal.Zero); err != nil {
				return fmt.Errorf("failed to adjust financial summary for line sale payment: %w", err)
			}
		}

		if s.metrics != nil {
			s.metrics.RecordSalesPayment(req.PaymentMethod, req.Amount.InexactFloat64())
		}
		return nil
	})
	if err != nil {
		if s.metrics != nil {
			s.metrics.IncFinancialTxFailures("record_line_sale_payment")
		}
		return nil, err
	}

	s.enqueueDailyStats(ctx, tenantID, todayString())
	return payment, nil
}

func (s *TenantOperationsService) ListLineSalePayments(ctx context.Context, tenantID, lineSaleID uuid.UUID) ([]domain.LineSalePayment, error) {
	return s.lineSaleRepo.ListPaymentsByLineSaleID(ctx, tenantID, lineSaleID)
}
