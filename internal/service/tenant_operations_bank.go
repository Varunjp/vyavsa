package service

import (
	"context"
	"fmt"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/dto"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ==========================================
// 4. Bank Management (Admin only)
// ==========================================

func (s *TenantOperationsService) CreateBank(ctx context.Context, tenantID uuid.UUID, req *dto.CreateBankRequest) (*domain.TenantBank, error) {
	status := req.Status
	if status == "" {
		status = "active"
	}

	currentBal := req.OpeningBalance
	if currentBal.IsNegative() {
		currentBal = decimal.Zero
	}

	bank := &domain.TenantBank{
		TenantID:       tenantID,
		BankName:       req.BankName,
		AccountNumber:  req.AccountNumber,
		IFSC:           req.IFSC,
		CurrentBalance: currentBal,
		Status:         status,
	}

	err := s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.bankRepo.Create(txCtx, bank); err != nil {
			return err
		}
		if currentBal.GreaterThan(decimal.Zero) {
			tx := &domain.BankTransaction{
				TenantID:        tenantID,
				BankID:          bank.ID,
				Amount:          currentBal,
				TransactionType: "credit",
				Reason:          "Opening Balance",
			}
			if err := s.bankRepo.CreateTransaction(txCtx, tx); err != nil {
				return err
			}
			if s.summaryRepo != nil {
				if err := s.summaryRepo.AdjustBalances(txCtx, tenantID, decimal.Zero, currentBal, decimal.Zero, decimal.Zero); err != nil {
					return fmt.Errorf("failed to update financial summary for bank opening balance: %w", err)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return bank, nil
}

func (s *TenantOperationsService) UpdateBank(ctx context.Context, tenantID, id uuid.UUID, req *dto.UpdateBankRequest) (*domain.TenantBank, error) {
	bank, err := s.bankRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if req.BankName != "" {
		bank.BankName = req.BankName
	}
	if req.AccountNumber != "" {
		bank.AccountNumber = req.AccountNumber
	}
	if req.IFSC != "" {
		bank.IFSC = req.IFSC
	}
	if req.Status != "" {
		bank.Status = req.Status
	}

	if err := s.bankRepo.Update(ctx, bank); err != nil {
		return nil, err
	}

	return bank, nil
}

func (s *TenantOperationsService) DeleteBank(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.bankRepo.Delete(txCtx, tenantID, id); err != nil {
			return err
		}
		if s.summaryRepo != nil {
			_, _ = s.summaryRepo.SyncFromSourceRecords(txCtx, tenantID)
		}
		return nil
	})
}

func (s *TenantOperationsService) GetBankByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantBank, error) {
	return s.bankRepo.GetByID(ctx, tenantID, id)
}

func (s *TenantOperationsService) ListBanks(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search, status string) ([]domain.TenantBank, int64, error) {
	return s.bankRepo.List(ctx, tenantID, page, pageSize, search, status)
}

func (s *TenantOperationsService) ListBankTransactions(ctx context.Context, tenantID, bankID uuid.UUID, page, pageSize int) ([]domain.BankTransaction, int64, error) {
	if _, err := s.bankRepo.GetByID(ctx, tenantID, bankID); err != nil {
		return nil, 0, err
	}
	return s.bankRepo.ListTransactions(ctx, tenantID, bankID, page, pageSize)
}
