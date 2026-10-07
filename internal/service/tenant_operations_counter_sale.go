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
// 6. Counter Sale Operations
// ==========================================

func (s *TenantOperationsService) CreateCounterSale(ctx context.Context, tenantID uuid.UUID, role string, req *dto.CreateCounterSaleRequest) (*domain.CounterSale, error) {
	start := time.Now()
	if req.TotalAmount.IsNegative() {
		if s.metrics != nil {
			s.metrics.RecordSalesPaymentFailed("negative_total_amount")
		}
		return nil, validationErr("total_amount", "total amount cannot be negative")
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
	} else if req.Cash.IsNegative() {
		if s.metrics != nil {
			s.metrics.RecordSalesPaymentFailed("negative_cash_amount")
		}
		return nil, validationErr("cash", "cash amount cannot be negative")
	} else if req.Cash.GreaterThan(decimal.Zero) {
		cashReceived = req.Cash
	}

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
			} else if p.PaymentMethod == "bank" || p.PaymentMethod == "online" || p.PaymentMethod == "upi" || p.PaymentMethod == "cheque" || p.PaymentMethod == "other" {
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
	} else if (req.PaymentMethod == "bank" || req.PaymentMethod == "online" || req.PaymentMethod == "upi") && req.Account.GreaterThan(decimal.Zero) && req.BankID != nil {
		rawBankSplits = append(rawBankSplits, dto.BankPaymentSplitRequest{
			BankID: *req.BankID,
			Amount: req.Account,
		})
	}

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
			return nil, appErrors.NewBadRequest("invalid bank account specified for counter sale")
		}
		if bank.Status != "active" {
			if s.metrics != nil {
				s.metrics.RecordSalesPaymentFailed("inactive_bank")
			}
			return nil, appErrors.NewBadRequest("selected bank account is not active")
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

	var primaryBankID *uuid.UUID
	if len(validatedBanks) > 0 {
		primaryBankID = &validatedBanks[0].BankID
	}

	method := req.PaymentMethod
	if method == "" {
		if cashReceived.GreaterThan(decimal.Zero) && bankReceived.GreaterThan(decimal.Zero) {
			method = "split"
		} else if bankReceived.GreaterThan(decimal.Zero) {
			method = "bank"
		} else {
			method = "cash"
		}
	}

	sale := &domain.CounterSale{
		TenantID:        tenantID,
		Item:            req.Item,
		Price:           req.Price,
		TotalAmount:     req.TotalAmount,
		PaymentMethod:   method,
		Cash:            cashReceived,
		BankAmount:      bankReceived,
		BankID:          primaryBankID,
		CollectedAmount: collectedAmount,
		Account:         due,
	}

	payments := make([]domain.CounterSalePayment, 0, len(validatedBanks)+1)
	if cashReceived.GreaterThan(decimal.Zero) {
		payments = append(payments, domain.CounterSalePayment{
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
		payments = append(payments, domain.CounterSalePayment{
			TenantID:      tenantID,
			PaymentMethod: "bank",
			BankID:        &vb.BankID,
			BankName:      vb.BankName,
			Amount:        vb.Amount,
			Note:          note,
		})
	}

	err := s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.countSaleRepo.Create(txCtx, sale, payments); err != nil {
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
				Reason:          "Counter Sale: " + sale.Item,
				SaleType:        "counter_sale",
				SaleID:          &sale.ID,
			}
			if err := s.bankRepo.CreateTransaction(txCtx, tx); err != nil {
				return fmt.Errorf("failed to record bank transaction for bank %s: %w", vb.BankName, err)
			}
		}

		// Update financial summary atomically
		if s.summaryRepo != nil {
			if err := s.summaryRepo.AdjustBalances(txCtx, tenantID, cashReceived, bankReceived, due, decimal.Zero); err != nil {
				return fmt.Errorf("failed to update financial summary for counter sale: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordSalesPaymentFailed("transaction_error")
			s.metrics.IncFinancialTxFailures("create_counter_sale")
		}
		return nil, err
	}

	if s.metrics != nil {
		s.metrics.RecordFinancialTxDuration("create_counter_sale", time.Since(start))
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

func (s *TenantOperationsService) UpdateCounterSale(ctx context.Context, tenantID, id uuid.UUID, role string, req *dto.UpdateCounterSaleRequest) (*domain.CounterSale, error) {
	sale, err := s.countSaleRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if err := checkCurrentDayRestriction(role, sale.CreatedAt.Format("2006-01-02")); err != nil {
		return nil, err
	}

	oldCash := sale.Cash
	oldBank := sale.BankAmount

	if req.Item != "" {
		sale.Item = req.Item
	}
	if req.Price != nil {
		sale.Price = *req.Price
	}
	if req.TotalAmount != nil {
		sale.TotalAmount = *req.TotalAmount
	}
	if req.PaymentMethod != "" {
		sale.PaymentMethod = req.PaymentMethod
	}
	if req.Cash != nil {
		sale.Cash = *req.Cash
	}
	if req.BankAmount != nil {
		sale.BankAmount = *req.BankAmount
	}
	if req.Account != nil {
		sale.Account = *req.Account
	}

	deltaCash := sale.Cash.Sub(oldCash)
	deltaBank := sale.BankAmount.Sub(oldBank)

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.countSaleRepo.Update(txCtx, sale); err != nil {
			return err
		}

		if s.summaryRepo != nil {
			if err := s.summaryRepo.AdjustBalances(txCtx, tenantID, deltaCash, deltaBank, decimal.Zero, decimal.Zero); err != nil {
				return fmt.Errorf("failed to update financial summary on counter sale update: %w", err)
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

func (s *TenantOperationsService) DeleteCounterSale(ctx context.Context, tenantID, id uuid.UUID, role string) error {
	sale, err := s.countSaleRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	if err := checkCurrentDayRestriction(role, sale.CreatedAt.Format("2006-01-02")); err != nil {
		return err
	}

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.countSaleRepo.Delete(txCtx, tenantID, id); err != nil {
			return err
		}
		if s.summaryRepo != nil {
			if err := s.summaryRepo.AdjustBalances(txCtx, tenantID, sale.Cash.Neg(), sale.BankAmount.Neg(), decimal.Zero, decimal.Zero); err != nil {
				return fmt.Errorf("failed to update financial summary on counter sale deletion: %w", err)
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

func (s *TenantOperationsService) GetCounterSaleByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.CounterSale, error) {
	return s.countSaleRepo.GetByID(ctx, tenantID, id)
}

func (s *TenantOperationsService) ListCounterSales(ctx context.Context, tenantID uuid.UUID, role string, page, pageSize int, date string, search string) ([]domain.CounterSale, int64, error) {
	if role == auth.RoleTenantUser {
		date = todayString()
	}
	return s.countSaleRepo.List(ctx, tenantID, page, pageSize, date, search)
}

func (s *TenantOperationsService) RecordCounterSalePayment(ctx context.Context, tenantID, counterSaleID uuid.UUID, role string, req *dto.RecordSalePaymentRequest) (*domain.CounterSalePayment, error) {
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, validationErr("amount", "payment amount must be greater than zero")
	}
	if req.PaymentMethod != "cash" && req.PaymentMethod != "bank" {
		return nil, validationErr("payment_method", "payment method must be either 'cash' or 'bank'")
	}

	if req.PaymentMethod == "bank" && req.BankID != nil && *req.BankID != uuid.Nil {
		bank, err := s.bankRepo.GetByID(ctx, tenantID, *req.BankID)
		if err != nil {
			return nil, appErrors.NewBadRequest("invalid bank account specified for counter sale payment")
		}
		req.BankName = bank.BankName
	}

	pDate := req.PaymentDate
	if pDate == "" {
		pDate = todayString()
	}

	var payment *domain.CounterSalePayment
	err := s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		sale, err := s.countSaleRepo.GetByIDForUpdate(txCtx, tenantID, counterSaleID)
		if err != nil {
			return err
		}
		pending := sale.TotalAmount.Sub(sale.CollectedAmount)
		if pending.LessThanOrEqual(decimal.Zero) {
			return appErrors.NewBadRequest("counter sale has already been fully collected")
		}
		if req.Amount.GreaterThan(pending) {
			return appErrors.NewBadRequest(fmt.Sprintf("payment amount (₹%s) exceeds pending amount (₹%s)", req.Amount.StringFixed(2), pending.StringFixed(2)))
		}

		payment = &domain.CounterSalePayment{
			TenantID:      tenantID,
			CounterSaleID: counterSaleID,
			PaymentMethod: req.PaymentMethod,
			BankID:        req.BankID,
			BankName:      req.BankName,
			Amount:        req.Amount,
			PaymentDate:   pDate,
			ReferenceID:   req.ReferenceID,
			Status:        "COMPLETED",
			Note:          req.Note,
		}
		if err := s.countSaleRepo.CreatePayment(txCtx, payment); err != nil {
			return fmt.Errorf("failed to create counter sale payment: %w", err)
		}

		sale.CollectedAmount = sale.CollectedAmount.Add(req.Amount)
		if req.PaymentMethod == "cash" {
			sale.Cash = sale.Cash.Add(req.Amount)
		} else {
			sale.BankAmount = sale.BankAmount.Add(req.Amount)
		}
		if err := s.countSaleRepo.Update(txCtx, sale); err != nil {
			return fmt.Errorf("failed to update counter sale: %w", err)
		}

		if s.summaryRepo != nil {
			var cashDelta, bankDelta decimal.Decimal
			if req.PaymentMethod == "cash" {
				cashDelta = req.Amount
			} else {
				bankDelta = req.Amount
			}
			if err := s.summaryRepo.AdjustBalances(txCtx, tenantID, cashDelta, bankDelta, decimal.Zero, decimal.Zero); err != nil {
				return fmt.Errorf("failed to adjust financial summary for counter sale payment: %w", err)
			}
		}

		if s.metrics != nil {
			s.metrics.RecordSalesPayment(req.PaymentMethod, req.Amount.InexactFloat64())
		}
		return nil
	})
	if err != nil {
		if s.metrics != nil {
			s.metrics.IncFinancialTxFailures("record_counter_sale_payment")
		}
		return nil, err
	}

	s.enqueueDailyStats(ctx, tenantID, todayString())
	return payment, nil
}

func (s *TenantOperationsService) ListCounterSalePayments(ctx context.Context, tenantID, counterSaleID uuid.UUID) ([]domain.CounterSalePayment, error) {
	return s.countSaleRepo.ListPaymentsByCounterSaleID(ctx, tenantID, counterSaleID)
}
