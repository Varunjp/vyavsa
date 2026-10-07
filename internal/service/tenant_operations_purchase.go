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
// 7. Purchase Operations
// ==========================================

func (s *TenantOperationsService) CreatePurchase(ctx context.Context, tenantID uuid.UUID, role string, req *dto.CreatePurchaseRequest) (*domain.TenantPurchase, error) {
	start := time.Now()
	if req.TotalAmount.IsNegative() {
		return nil, validationErr("total_amount", "total amount cannot be negative")
	}

	purchase := &domain.TenantPurchase{
		TenantID:    tenantID,
		Item:        req.Item,
		Quantity:    req.Quantity,
		TotalAmount: req.TotalAmount,
	}

	if req.CustomerID != nil && *req.CustomerID != uuid.Nil {
		cust, err := s.custRepo.GetByID(ctx, tenantID, *req.CustomerID)
		if err != nil {
			return nil, appErrors.NewBadRequest("invalid customer for tenant")
		}
		if cust.Status != "active" {
			return nil, appErrors.NewBadRequest("customer is not active")
		}
		purchase.CustomerID = &cust.ID
		purchase.CustomerName = cust.CustomerName
	}

	// Process payments: multi-bank split / cash / single payment
	var cashPaid, bankPaid decimal.Decimal
	payments := make([]domain.TenantPurchasePayment, 0)
	var validatedBanks []validatedBankPayment

	if len(req.BankPayments) > 0 || req.CashAmount != nil {
		if req.CashAmount != nil && req.CashAmount.GreaterThan(decimal.Zero) {
			cashPaid = *req.CashAmount
			payments = append(payments, domain.TenantPurchasePayment{
				TenantID:      tenantID,
				CustomerID:    purchase.CustomerID,
				PaymentMethod: "cash",
				Amount:        cashPaid,
				Note:          "Cash payment",
			})
		}

		if len(req.BankPayments) > 0 {
			seenBanks := make(map[uuid.UUID]bool)
			for _, bp := range req.BankPayments {
				if bp.Amount.LessThanOrEqual(decimal.Zero) {
					return nil, validationErr("amount", "each bank payment amount must be greater than zero")
				}
				if seenBanks[bp.BankID] {
					return nil, appErrors.NewBadRequest(fmt.Sprintf("duplicate bank account detected: bank %s can only be selected once", bp.BankID))
				}
				seenBanks[bp.BankID] = true

				bank, err := s.bankRepo.GetByID(ctx, tenantID, bp.BankID)
				if err != nil {
					return nil, appErrors.NewBadRequest("invalid bank account specified for purchase")
				}
				if bank.Status != "active" {
					return nil, appErrors.NewBadRequest(fmt.Sprintf("bank account %s is not active", bank.BankName))
				}
				bankPaid = bankPaid.Add(bp.Amount)
				validatedBanks = append(validatedBanks, validatedBankPayment{
					BankID:   bank.ID,
					BankName: bank.BankName,
					Amount:   bp.Amount,
					Note:     bp.Note,
				})
				payments = append(payments, domain.TenantPurchasePayment{
					TenantID:      tenantID,
					CustomerID:    purchase.CustomerID,
					PaymentMethod: "bank",
					BankID:        &bp.BankID,
					BankName:      bank.BankName,
					Amount:        bp.Amount,
					Note:          bp.Note,
				})
			}
		}
	} else if len(req.Payments) > 0 {
		for _, p := range req.Payments {
			if p.PaymentMethod == "cash" {
				cashPaid = cashPaid.Add(p.Amount)
			} else {
				bankPaid = bankPaid.Add(p.Amount)
			}
			if p.BankID != nil && *p.BankID != uuid.Nil {
				bank, err := s.bankRepo.GetByID(ctx, tenantID, *p.BankID)
				if err != nil {
					return nil, appErrors.NewBadRequest("invalid bank account specified for purchase")
				}
				p.BankName = bank.BankName
				validatedBanks = append(validatedBanks, validatedBankPayment{
					BankID:   bank.ID,
					BankName: bank.BankName,
					Amount:   p.Amount,
					Note:     p.Note,
				})
			}
			payments = append(payments, domain.TenantPurchasePayment{
				TenantID:      tenantID,
				CustomerID:    purchase.CustomerID,
				PaymentMethod: p.PaymentMethod,
				BankID:        p.BankID,
				BankName:      p.BankName,
				Amount:        p.Amount,
				Note:          p.Note,
			})
		}
	} else if req.TotalPaid.GreaterThan(decimal.Zero) {
		cashPaid = req.TotalPaid
		payments = append(payments, domain.TenantPurchasePayment{
			TenantID:      tenantID,
			CustomerID:    purchase.CustomerID,
			PaymentMethod: "cash",
			Amount:        req.TotalPaid,
		})
	}

	actualPaid := cashPaid.Add(bankPaid)
	purchase.TotalPaid = actualPaid
	pending := purchase.TotalAmount.Sub(purchase.TotalPaid)
	if pending.IsNegative() {
		pending = decimal.Zero
	}
	purchase.TotalPending = pending
	purchase.OutstandingAmount = purchase.TotalPending
	purchase.PaymentStatus = domain.ComputePurchasePaymentStatus(purchase.TotalAmount, purchase.TotalPaid)

	err := s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.purchRepo.Create(txCtx, purchase, payments); err != nil {
			return err
		}

		for _, vb := range validatedBanks {
			if err := s.bankRepo.AdjustBalance(txCtx, tenantID, vb.BankID, vb.Amount.Neg()); err != nil {
				return fmt.Errorf("failed to deduct bank balance for %s: %w", vb.BankName, err)
			}
			tx := &domain.BankTransaction{
				TenantID:        tenantID,
				BankID:          vb.BankID,
				Amount:          vb.Amount,
				TransactionType: "debit",
				Reason:          "Purchase: " + purchase.Item,
				SaleType:        "purchase",
				SaleID:          &purchase.ID,
			}
			if err := s.bankRepo.CreateTransaction(txCtx, tx); err != nil {
				return fmt.Errorf("failed to record bank transaction for %s: %w", vb.BankName, err)
			}
		}

		// Financial summary: paid reduces cash/bank, pending adds to payable atomically
		if s.summaryRepo != nil {
			if err := s.summaryRepo.AdjustBalances(txCtx, tenantID, cashPaid.Neg(), bankPaid.Neg(), decimal.Zero, pending); err != nil {
				return fmt.Errorf("failed to update financial summary for purchase: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		if s.metrics != nil {
			s.metrics.IncFinancialTxFailures("create_purchase")
		}
		return nil, err
	}

	if s.metrics != nil {
		s.metrics.RecordFinancialTxDuration("create_purchase", time.Since(start))
		if cashPaid.GreaterThan(decimal.Zero) {
			s.metrics.RecordPurchasePayment("cash", cashPaid.InexactFloat64())
		}
		if bankPaid.GreaterThan(decimal.Zero) {
			s.metrics.RecordPurchasePayment("bank", bankPaid.InexactFloat64())
		}
	}

	s.enqueueDailyStats(ctx, tenantID, todayString())

	return purchase, nil
}

func (s *TenantOperationsService) UpdatePurchase(ctx context.Context, tenantID, id uuid.UUID, role string, req *dto.UpdatePurchaseRequest) (*domain.TenantPurchase, error) {
	purchase, err := s.purchRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if err := checkCurrentDayRestriction(role, purchase.CreatedAt.Format("2006-01-02")); err != nil {
		return nil, err
	}

	oldPaid := purchase.TotalPaid
	oldPending := purchase.TotalPending

	if req.Item != "" {
		purchase.Item = req.Item
	}
	if req.Quantity != nil {
		purchase.Quantity = *req.Quantity
	}
	if req.TotalAmount != nil {
		purchase.TotalAmount = *req.TotalAmount
		purchase.TotalPending = purchase.TotalAmount.Sub(purchase.TotalPaid)
	}
	if req.TotalPaid != nil {
		purchase.TotalPaid = *req.TotalPaid
		purchase.TotalPending = purchase.TotalAmount.Sub(purchase.TotalPaid)
	}
	if purchase.TotalPending.IsNegative() {
		purchase.TotalPending = decimal.Zero
	}
	purchase.OutstandingAmount = purchase.TotalPending
	purchase.PaymentStatus = domain.ComputePurchasePaymentStatus(purchase.TotalAmount, purchase.TotalPaid)

	if req.CustomerID != nil {
		if *req.CustomerID == uuid.Nil {
			purchase.CustomerID = nil
			purchase.CustomerName = ""
		} else {
			cust, err := s.custRepo.GetByID(ctx, tenantID, *req.CustomerID)
			if err != nil {
				return nil, appErrors.NewBadRequest("invalid customer for tenant")
			}
			if cust.Status != "active" {
				return nil, appErrors.NewBadRequest("customer is not active")
			}
			purchase.CustomerID = &cust.ID
			purchase.CustomerName = cust.CustomerName
		}
	}

	deltaPaid := purchase.TotalPaid.Sub(oldPaid)
	deltaPending := purchase.TotalPending.Sub(oldPending)

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.purchRepo.Update(txCtx, purchase); err != nil {
			return err
		}

		if s.summaryRepo != nil {
			if err := s.summaryRepo.AdjustBalances(txCtx, tenantID, deltaPaid.Neg(), decimal.Zero, decimal.Zero, deltaPending); err != nil {
				return fmt.Errorf("failed to update financial summary on purchase update: %w", err)
			}
			_, _ = s.summaryRepo.SyncFromSourceRecords(txCtx, tenantID)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	s.enqueueDailyStats(ctx, tenantID, purchase.CreatedAt.Format("2006-01-02"))
	if purchase.CreatedAt.Format("2006-01-02") != todayString() {
		s.enqueueDailyStats(ctx, tenantID, todayString())
	}

	return purchase, nil
}

func (s *TenantOperationsService) DeletePurchase(ctx context.Context, tenantID, id uuid.UUID, role string) error {
	purchase, err := s.purchRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	if err := checkCurrentDayRestriction(role, purchase.CreatedAt.Format("2006-01-02")); err != nil {
		return err
	}

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.purchRepo.Delete(txCtx, tenantID, id); err != nil {
			return err
		}
		if s.summaryRepo != nil {
			if err := s.summaryRepo.AdjustBalances(txCtx, tenantID, purchase.TotalPaid, decimal.Zero, decimal.Zero, purchase.TotalPending.Neg()); err != nil {
				return fmt.Errorf("failed to update financial summary on purchase deletion: %w", err)
			}
			_, _ = s.summaryRepo.SyncFromSourceRecords(txCtx, tenantID)
		}
		return nil
	})
	if err != nil {
		return err
	}

	s.enqueueDailyStats(ctx, tenantID, purchase.CreatedAt.Format("2006-01-02"))
	if purchase.CreatedAt.Format("2006-01-02") != todayString() {
		s.enqueueDailyStats(ctx, tenantID, todayString())
	}
	return nil
}

func (s *TenantOperationsService) GetPurchaseByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantPurchase, error) {
	p, err := s.purchRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	p.OutstandingAmount = p.TotalPending
	if p.PaymentStatus == "" {
		p.PaymentStatus = domain.ComputePurchasePaymentStatus(p.TotalAmount, p.TotalPaid)
	}
	return p, nil
}

func (s *TenantOperationsService) ListPurchases(ctx context.Context, tenantID uuid.UUID, role string, page, pageSize int, date string, customerID *uuid.UUID, search string) ([]domain.TenantPurchase, int64, error) {
	if role == auth.RoleTenantUser {
		date = todayString()
	}
	purchases, total, err := s.purchRepo.List(ctx, tenantID, page, pageSize, date, customerID, search)
	if err != nil {
		return nil, 0, err
	}
	for i := range purchases {
		purchases[i].OutstandingAmount = purchases[i].TotalPending
		if purchases[i].PaymentStatus == "" {
			purchases[i].PaymentStatus = domain.ComputePurchasePaymentStatus(purchases[i].TotalAmount, purchases[i].TotalPaid)
		}
	}
	return purchases, total, nil
}

func (s *TenantOperationsService) RecordSupplierPayment(ctx context.Context, tenantID uuid.UUID, role string, req *dto.RecordSupplierPaymentRequest) (*dto.SupplierPaymentResponse, error) {
	start := time.Now()
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		s.metrics.RecordPurchasePaymentFailed("invalid_amount")
		return nil, validationErr("amount", "payment amount must be strictly greater than zero")
	}

	if (req.CustomerID == nil || *req.CustomerID == uuid.Nil) && (req.PurchaseID == nil || *req.PurchaseID == uuid.Nil) {
		s.metrics.RecordPurchasePaymentFailed("missing_target")
		return nil, appErrors.NewBadRequest("either customer_id or purchase_id must be provided")
	}

	if req.PaymentMethod != "cash" && req.PaymentMethod != "bank" {
		s.metrics.RecordPurchasePaymentFailed("invalid_method")
		return nil, validationErr("payment_method", "payment method must be either 'cash' or 'bank'")
	}

	var cashPaid, bankPaid decimal.Decimal
	var validatedBanks []validatedBankPayment

	if req.PaymentMethod == "cash" {
		cashPaid = req.Amount
	} else {
		// Payment via bank
		if len(req.BankPayments) > 0 {
			var splitSum decimal.Decimal
			seenBanks := make(map[uuid.UUID]bool)
			for _, bp := range req.BankPayments {
				if bp.Amount.LessThanOrEqual(decimal.Zero) {
					s.metrics.RecordPurchasePaymentFailed("invalid_split_amount")
					return nil, validationErr("amount", "each bank payment amount must be greater than zero")
				}
				if seenBanks[bp.BankID] {
					s.metrics.RecordPurchasePaymentFailed("duplicate_bank")
					return nil, appErrors.NewBadRequest(fmt.Sprintf("duplicate bank account detected: bank %s can only be selected once", bp.BankID))
				}
				seenBanks[bp.BankID] = true

				bank, err := s.bankRepo.GetByID(ctx, tenantID, bp.BankID)
				if err != nil {
					s.metrics.RecordPurchasePaymentFailed("invalid_bank")
					return nil, appErrors.NewBadRequest("one or more bank accounts do not exist or belong to another tenant")
				}
				if bank.Status != "active" {
					s.metrics.RecordPurchasePaymentFailed("inactive_bank")
					return nil, appErrors.NewBadRequest(fmt.Sprintf("bank account %s is not active", bank.BankName))
				}
				splitSum = splitSum.Add(bp.Amount)
				validatedBanks = append(validatedBanks, validatedBankPayment{
					BankID:   bank.ID,
					BankName: bank.BankName,
					Amount:   bp.Amount,
					Note:     bp.Note,
				})
			}
			if !splitSum.Equal(req.Amount) {
				s.metrics.RecordPurchasePaymentFailed("split_mismatch")
				return nil, validationErr("amount", fmt.Sprintf("sum of bank payments (%s) does not match total amount (%s)", splitSum.StringFixed(2), req.Amount.StringFixed(2)))
			}
			bankPaid = splitSum
		} else {
			if req.BankID == nil || *req.BankID == uuid.Nil {
				s.metrics.RecordPurchasePaymentFailed("missing_bank")
				return nil, validationErr("bank_id", "bank account is required when payment method is 'bank'")
			}
			bank, err := s.bankRepo.GetByID(ctx, tenantID, *req.BankID)
			if err != nil {
				s.metrics.RecordPurchasePaymentFailed("invalid_bank")
				return nil, appErrors.NewBadRequest("bank account does not exist or belongs to another tenant")
			}
			if bank.Status != "active" {
				s.metrics.RecordPurchasePaymentFailed("inactive_bank")
				return nil, appErrors.NewBadRequest(fmt.Sprintf("bank account %s is not active", bank.BankName))
			}
			bankPaid = req.Amount
			validatedBanks = append(validatedBanks, validatedBankPayment{
				BankID:   bank.ID,
				BankName: bank.BankName,
				Amount:   req.Amount,
				Note:     req.Note,
			})
		}
	}

	var response *dto.SupplierPaymentResponse

	err := s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		var customer *domain.TenantCustomer
		var customerID uuid.UUID
		var targetPurchases []domain.TenantPurchase

		if req.PurchaseID != nil && *req.PurchaseID != uuid.Nil {
			p, err := s.purchRepo.GetByIDForUpdate(txCtx, tenantID, *req.PurchaseID)
			if err != nil {
				return err
			}
			if p.TotalPending.LessThanOrEqual(decimal.Zero) {
				return appErrors.NewBadRequest(fmt.Sprintf("purchase '%s' has already been fully paid (pending: ₹0.00)", p.Item))
			}
			if req.Amount.GreaterThan(p.TotalPending) {
				return appErrors.NewBadRequest(fmt.Sprintf("payment amount (%s) exceeds pending balance of purchase (%s)", req.Amount.StringFixed(2), p.TotalPending.StringFixed(2)))
			}
			if p.CustomerID != nil && *p.CustomerID != uuid.Nil {
				customerID = *p.CustomerID
				cust, err := s.custRepo.GetByIDForUpdate(txCtx, tenantID, customerID)
				if err == nil {
					customer = cust
				}
			}
			targetPurchases = []domain.TenantPurchase{*p}
		} else {
			customerID = *req.CustomerID
			cust, err := s.custRepo.GetByIDForUpdate(txCtx, tenantID, customerID)
			if err != nil {
				return appErrors.NewBadRequest("invalid customer for tenant")
			}
			customer = cust

			purchases, err := s.purchRepo.GetCustomerPurchasesForUpdate(txCtx, tenantID, customerID)
			if err != nil {
				return err
			}
			targetPurchases = purchases
		}

		// Calculate total outstanding pending
		var totalOutstanding decimal.Decimal
		for _, p := range targetPurchases {
			totalOutstanding = totalOutstanding.Add(p.TotalPending)
		}

		if totalOutstanding.LessThanOrEqual(decimal.Zero) {
			return appErrors.NewBadRequest("customer has no outstanding payable balance")
		}

		if req.Amount.GreaterThan(totalOutstanding) {
			return appErrors.NewBadRequest(fmt.Sprintf("payment amount (%s) exceeds outstanding payable balance (%s)", req.Amount.StringFixed(2), totalOutstanding.StringFixed(2)))
		}

		remaining := req.Amount
		allocations := make([]dto.SupplierPaymentAllocation, 0)
		paymentID := uuid.New()

		// Allocate FIFO to oldest outstanding purchases first
		for i := range targetPurchases {
			if remaining.IsZero() {
				break
			}
			p := &targetPurchases[i]
			alloc := decimal.Min(remaining, p.TotalPending)
			prevPending := p.TotalPending

			p.TotalPaid = p.TotalPaid.Add(alloc)
			p.TotalPending = p.TotalPending.Sub(alloc)
			p.PaymentStatus = domain.ComputePurchasePaymentStatus(p.TotalAmount, p.TotalPaid)
			p.OutstandingAmount = p.TotalPending

			if err := s.purchRepo.Update(txCtx, p); err != nil {
				return fmt.Errorf("failed to update purchase %s: %w", p.ID, err)
			}

			// Record payment split(s) on this purchase
			custID := p.CustomerID
			if custID == nil && customer != nil {
				custID = &customer.ID
			}

			pDate := req.PaymentDate
			if pDate == "" {
				pDate = todayString()
			}
			refID := req.ReferenceID

			if req.PaymentMethod == "cash" {
				paymentRecord := &domain.TenantPurchasePayment{
					TenantID:      tenantID,
					PurchaseID:    p.ID,
					CustomerID:    custID,
					PaymentMethod: "cash",
					Amount:        alloc,
					PaymentDate:   pDate,
					ReferenceID:   refID,
					Status:        "COMPLETED",
					Note:          req.Note,
					IsSettlement:  true,
				}
				if err := s.purchRepo.CreatePayment(txCtx, paymentRecord); err != nil {
					return fmt.Errorf("failed to record cash purchase payment: %w", err)
				}
			} else {
				if len(validatedBanks) == 1 {
					vb := validatedBanks[0]
					paymentRecord := &domain.TenantPurchasePayment{
						TenantID:      tenantID,
						PurchaseID:    p.ID,
						CustomerID:    custID,
						PaymentMethod: "bank",
						BankID:        &vb.BankID,
						BankName:      vb.BankName,
						Amount:        alloc,
						PaymentDate:   pDate,
						ReferenceID:   refID,
						Status:        "COMPLETED",
						Note:          req.Note,
						IsSettlement:  true,
					}
					if err := s.purchRepo.CreatePayment(txCtx, paymentRecord); err != nil {
						return fmt.Errorf("failed to record bank purchase payment: %w", err)
					}
				} else {
					for _, vb := range validatedBanks {
						share := alloc.Mul(vb.Amount).Div(req.Amount)
						if share.GreaterThan(decimal.Zero) {
							paymentRecord := &domain.TenantPurchasePayment{
								TenantID:      tenantID,
								PurchaseID:    p.ID,
								CustomerID:    custID,
								PaymentMethod: "bank",
								BankID:        &vb.BankID,
								BankName:      vb.BankName,
								Amount:        share,
								PaymentDate:   pDate,
								ReferenceID:   refID,
								Status:        "COMPLETED",
								Note:          req.Note,
								IsSettlement:  true,
							}
							if err := s.purchRepo.CreatePayment(txCtx, paymentRecord); err != nil {
								return fmt.Errorf("failed to record split bank purchase payment: %w", err)
							}
						}
					}
				}
			}

			allocations = append(allocations, dto.SupplierPaymentAllocation{
				PurchaseID:      p.ID,
				Item:            p.Item,
				PreviousPending: prevPending,
				AmountSettled:   alloc,
				NewPending:      p.TotalPending,
				PaymentStatus:   p.PaymentStatus,
			})

			remaining = remaining.Sub(alloc)
		}

		// Adjust bank accounts and record bank transactions
		custName := "Walk-in Supplier"
		if customer != nil {
			custName = customer.CustomerName
		}
		for _, vb := range validatedBanks {
			if err := s.bankRepo.AdjustBalance(txCtx, tenantID, vb.BankID, vb.Amount.Neg()); err != nil {
				return fmt.Errorf("failed to deduct bank balance for %s: %w", vb.BankName, err)
			}
			tx := &domain.BankTransaction{
				TenantID:        tenantID,
				BankID:          vb.BankID,
				Amount:          vb.Amount,
				TransactionType: "debit",
				Reason:          "Supplier Settlement: " + custName,
				SaleType:        "purchase_settlement",
			}
			if err := s.bankRepo.CreateTransaction(txCtx, tx); err != nil {
				return fmt.Errorf("failed to record bank transaction for %s: %w", vb.BankName, err)
			}
		}

		// Financial summary: Cash/Bank balance reduces, TotalPayable reduces atomically
		if s.summaryRepo != nil {
			if err := s.summaryRepo.AdjustBalances(txCtx, tenantID, cashPaid.Neg(), bankPaid.Neg(), decimal.Zero, req.Amount.Neg()); err != nil {
				return fmt.Errorf("failed to update financial summary for supplier payment: %w", err)
			}
			_, _ = s.summaryRepo.SyncFromSourceRecords(txCtx, tenantID)
		}

		newOutstanding := totalOutstanding.Sub(req.Amount)
		response = &dto.SupplierPaymentResponse{
			PaymentID:           paymentID,
			CustomerID:          customerID,
			CustomerName:        custName,
			Amount:              req.Amount,
			PaymentMethod:       req.PaymentMethod,
			PreviousOutstanding: totalOutstanding,
			NewOutstanding:      newOutstanding,
			Allocations:         allocations,
			CreatedAt:           time.Now().UTC(),
		}

		return nil
	})

	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordPurchasePaymentFailed("transaction_error")
			s.metrics.IncFinancialTxFailures("record_purchase_payment")
		}
		return nil, err
	}

	if s.metrics != nil {
		s.metrics.RecordFinancialTxDuration("record_purchase_payment", time.Since(start))
	}

	s.enqueueDailyStats(ctx, tenantID, todayString())

	if req.PaymentMethod == "cash" {
		s.metrics.RecordPurchasePayment("cash", cashPaid.InexactFloat64())
	} else {
		s.metrics.RecordPurchasePayment("bank", bankPaid.InexactFloat64())
	}

	return response, nil
}

func (s *TenantOperationsService) ListPurchasePaymentsByPurchaseID(ctx context.Context, tenantID, purchaseID uuid.UUID) ([]domain.TenantPurchasePayment, error) {
	return s.purchRepo.ListPaymentsByPurchaseID(ctx, tenantID, purchaseID)
}
