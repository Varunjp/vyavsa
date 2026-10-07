package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/dto"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ==========================================
// 3. Customer Management (Admin only)
// ==========================================

func (s *TenantOperationsService) CreateCustomer(ctx context.Context, tenantID uuid.UUID, req *dto.CreateCustomerRequest) (*domain.TenantCustomer, error) {
	status := req.Status
	if status == "" {
		status = "active"
	}

	cust := &domain.TenantCustomer{
		TenantID:       tenantID,
		CustomerName:   req.CustomerName,
		Phone:          req.Phone,
		OpeningBalance: req.OpeningBalance,
		CurrentBalance: req.OpeningBalance,
		Status:         status,
	}

	err := s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.custRepo.Create(txCtx, cust); err != nil {
			return err
		}
		// If opening balance > 0, reflect in total_receivable atomically
		if req.OpeningBalance.GreaterThan(decimal.Zero) && s.summaryRepo != nil {
			if err := s.summaryRepo.AdjustBalances(txCtx, tenantID, decimal.Zero, decimal.Zero, req.OpeningBalance, decimal.Zero); err != nil {
				return fmt.Errorf("failed to update financial summary for customer opening balance: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return cust, nil
}

func (s *TenantOperationsService) UpdateCustomer(ctx context.Context, tenantID, id uuid.UUID, req *dto.UpdateCustomerRequest) (*domain.TenantCustomer, error) {
	cust, err := s.custRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if req.CustomerName != "" {
		cust.CustomerName = req.CustomerName
	}
	if req.Phone != "" {
		cust.Phone = req.Phone
	}
	if req.Status != "" {
		cust.Status = req.Status
	}

	if err := s.custRepo.Update(ctx, cust); err != nil {
		return nil, err
	}

	return cust, nil
}

func (s *TenantOperationsService) DeleteCustomer(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.custRepo.Delete(txCtx, tenantID, id); err != nil {
			return err
		}
		if s.summaryRepo != nil {
			_, _ = s.summaryRepo.SyncFromSourceRecords(txCtx, tenantID)
		}
		return nil
	})
}

func (s *TenantOperationsService) GetCustomerByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantCustomer, error) {
	return s.custRepo.GetByID(ctx, tenantID, id)
}

func (s *TenantOperationsService) ListCustomers(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search, status string) ([]dto.CustomerResponse, int64, error) {
	customers, total, err := s.custRepo.List(ctx, tenantID, page, pageSize, search, status)
	if err != nil {
		return nil, 0, err
	}
	if len(customers) == 0 {
		return []dto.CustomerResponse{}, total, nil
	}

	var payableMap map[uuid.UUID]domain.CustomerPayableSummary
	if s.purchRepo != nil {
		customerIDs := make([]uuid.UUID, len(customers))
		for i := range customers {
			customerIDs[i] = customers[i].ID
		}
		payableMap, _ = s.purchRepo.GetCustomerPayableSummariesBatch(ctx, tenantID, customerIDs)
	}

	res := make([]dto.CustomerResponse, len(customers))
	for i := range customers {
		var summary *domain.CustomerPayableSummary
		if payableMap != nil {
			if sVal, ok := payableMap[customers[i].ID]; ok {
				summary = &sVal
			}
		}
		res[i] = dto.ToCustomerDetailResponse(&customers[i], summary)
	}

	return res, total, nil
}

func (s *TenantOperationsService) AdjustCustomerBalance(ctx context.Context, tenantID, customerID uuid.UUID, req *dto.AdjustCustomerBalanceRequest) (*domain.TenantCustomer, error) {
	if req.Reason == "" {
		return nil, validationErr("reason", "reason for balance adjustment is required")
	}
	if req.NewBalance == nil && req.AdjustmentAmount == nil {
		return nil, validationErr("balance", "either new_balance or adjustment_amount must be provided")
	}

	cust, err := s.custRepo.GetByID(ctx, tenantID, customerID)
	if err != nil {
		return nil, appErrors.NewBadRequest("invalid customer for tenant")
	}

	prevBalance := cust.CurrentBalance
	var newBalance, delta decimal.Decimal

	if req.NewBalance != nil {
		newBalance = *req.NewBalance
		delta = newBalance.Sub(prevBalance)
	} else {
		delta = *req.AdjustmentAmount
		newBalance = prevBalance.Add(delta)
	}

	adjustment := &domain.CustomerBalanceAdjustment{
		TenantID:         tenantID,
		CustomerID:       customerID,
		CustomerName:     cust.CustomerName,
		PreviousBalance:  prevBalance,
		NewBalance:       newBalance,
		AdjustmentAmount: delta,
		Reason:           req.Reason,
	}

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.custRepo.SetBalance(txCtx, tenantID, customerID, newBalance); err != nil {
			return err
		}

		if err := s.custRepo.RecordAdjustment(txCtx, adjustment); err != nil {
			return err
		}

		if s.summaryRepo != nil {
			if err := s.summaryRepo.AdjustBalances(txCtx, tenantID, decimal.Zero, decimal.Zero, delta, decimal.Zero); err != nil {
				return fmt.Errorf("failed to update financial summary for customer balance adjustment: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	cust.CurrentBalance = newBalance
	return cust, nil
}

func (s *TenantOperationsService) ListCustomerAdjustments(ctx context.Context, tenantID, customerID uuid.UUID, page, pageSize int) ([]domain.CustomerBalanceAdjustment, int64, error) {
	if _, err := s.custRepo.GetByID(ctx, tenantID, customerID); err != nil {
		return nil, 0, err
	}
	return s.custRepo.ListAdjustments(ctx, tenantID, customerID, page, pageSize)
}

func (s *TenantOperationsService) GetCustomerBalance(ctx context.Context, tenantID, customerID uuid.UUID) (*dto.CustomerBalanceResponse, error) {
	cust, err := s.custRepo.GetByID(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}
	totalPurchases, totalPaid, outstandingPayable, _ := s.purchRepo.GetCustomerPayableSummary(ctx, tenantID, customerID)
	return &dto.CustomerBalanceResponse{
		CustomerID:         cust.ID,
		CustomerName:       cust.CustomerName,
		CurrentBalance:     cust.CurrentBalance,
		TotalPurchases:     totalPurchases,
		TotalPaid:          totalPaid,
		OutstandingPayable: outstandingPayable,
		Status:             cust.Status,
	}, nil
}

func (s *TenantOperationsService) GetCustomerDetails(ctx context.Context, tenantID, id uuid.UUID) (*dto.CustomerResponse, error) {
	cust, err := s.custRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	totalPurchases, totalPaid, outstandingPayable, _ := s.purchRepo.GetCustomerPayableSummary(ctx, tenantID, id)
	resp := dto.ToCustomerDetailResponse(cust, &domain.CustomerPayableSummary{
		TotalPurchases:     totalPurchases,
		TotalPaid:          totalPaid,
		OutstandingPayable: outstandingPayable,
	})
	return &resp, nil
}

func (s *TenantOperationsService) GetCustomerStatement(ctx context.Context, tenantID, customerID uuid.UUID) (*dto.CustomerStatementResponse, error) {
	cust, err := s.custRepo.GetByID(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}

	purchases, _, err := s.purchRepo.List(ctx, tenantID, 1, 1000, "", &customerID, "")
	if err != nil {
		return nil, err
	}

	payments, err := s.purchRepo.ListPaymentsByCustomerID(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}

	sort.Slice(purchases, func(i, j int) bool {
		return purchases[i].CreatedAt.Before(purchases[j].CreatedAt)
	})

	type rawEntry struct {
		Date           time.Time
		Description    string
		EntryType      string
		PurchaseAmount decimal.Decimal
		PaidAmount     decimal.Decimal
		PaymentMethod  string
		BankName       string
		ReferenceID    uuid.UUID
	}

	var rawEntries []rawEntry
	var totalPurchases, totalPaid decimal.Decimal

	initialPaidByPurchase := make(map[uuid.UUID]decimal.Decimal)
	for _, p := range payments {
		if !p.IsSettlement {
			initialPaidByPurchase[p.PurchaseID] = initialPaidByPurchase[p.PurchaseID].Add(p.Amount)
		}
	}

	for _, p := range purchases {
		initPaid := initialPaidByPurchase[p.ID]
		if initPaid.IsZero() && p.TotalPaid.GreaterThan(decimal.Zero) {
			var settPaid decimal.Decimal
			for _, pm := range payments {
				if pm.PurchaseID == p.ID && pm.IsSettlement {
					settPaid = settPaid.Add(pm.Amount)
				}
			}
			initPaid = p.TotalPaid.Sub(settPaid)
			if initPaid.IsNegative() {
				initPaid = decimal.Zero
			}
		}

		totalPurchases = totalPurchases.Add(p.TotalAmount)
		totalPaid = totalPaid.Add(initPaid)

		rawEntries = append(rawEntries, rawEntry{
			Date:           p.CreatedAt,
			Description:    p.Item,
			EntryType:      "purchase",
			PurchaseAmount: p.TotalAmount,
			PaidAmount:     initPaid,
			ReferenceID:    p.ID,
		})
	}

	for _, pm := range payments {
		if pm.IsSettlement {
			totalPaid = totalPaid.Add(pm.Amount)
			desc := "Payment to Supplier"
			if pm.Note != "" {
				desc = "Payment: " + pm.Note
			} else if pm.BankName != "" {
				desc = "Bank Settlement (" + pm.BankName + ")"
			} else if pm.PaymentMethod == "cash" {
				desc = "Cash Settlement"
			}
			rawEntries = append(rawEntries, rawEntry{
				Date:           pm.CreatedAt,
				Description:    desc,
				EntryType:      "settlement_payment",
				PurchaseAmount: decimal.Zero,
				PaidAmount:     pm.Amount,
				PaymentMethod:  pm.PaymentMethod,
				BankName:       pm.BankName,
				ReferenceID:    pm.ID,
			})
		}
	}

	sort.Slice(rawEntries, func(i, j int) bool {
		return rawEntries[i].Date.Before(rawEntries[j].Date)
	})

	var runningBalance decimal.Decimal
	entries := make([]dto.CustomerStatementEntry, len(rawEntries))
	for i, re := range rawEntries {
		if re.EntryType == "purchase" {
			runningBalance = runningBalance.Add(re.PurchaseAmount.Sub(re.PaidAmount))
		} else {
			runningBalance = runningBalance.Sub(re.PaidAmount)
		}

		entries[i] = dto.CustomerStatementEntry{
			ID:             re.ReferenceID,
			Date:           re.Date,
			FormattedDate:  re.Date.Format("02 Jan 2006"),
			Description:    re.Description,
			EntryType:      re.EntryType,
			PurchaseAmount: re.PurchaseAmount,
			PaidAmount:     re.PaidAmount,
			Balance:        runningBalance,
			PaymentMethod:  re.PaymentMethod,
			BankName:       re.BankName,
			ReferenceID:    re.ReferenceID,
		}
	}

	outstandingPayable := totalPurchases.Sub(totalPaid)
	if outstandingPayable.IsNegative() {
		outstandingPayable = decimal.Zero
	}

	return &dto.CustomerStatementResponse{
		CustomerID:         cust.ID,
		CustomerName:       cust.CustomerName,
		Phone:              cust.Phone,
		OpeningBalance:     cust.OpeningBalance,
		CurrentBalance:     cust.CurrentBalance,
		TotalPurchases:     totalPurchases,
		TotalPaid:          totalPaid,
		OutstandingPayable: outstandingPayable,
		Entries:            entries,
	}, nil
}
