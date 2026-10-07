package pdf_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/pdf"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestGenerateRealisticSamplePDF(t *testing.T) {
	generator := pdf.NewPDFGenerator()

	tenantID := uuid.New()
	report := &domain.DailyReport{
		Tenant: domain.TenantReportInfo{
			ID:    tenantID,
			Name:  "Royal Kirana & General Stores",
			Email: "contact@royalkirana.com",
			Phone: "+91 98765 43210",
		},
		Date:          "2026-10-07",
		FormattedDate: "07 October 2026",
		GeneratedAt:   time.Date(2026, 10, 7, 18, 30, 0, 0, time.UTC),
		LineSales: []domain.LineSaleReportItem{
			{
				ID:            uuid.New(),
				InvoiceNumber: "LS-1001",
				Time:          "09:30",
				CustomerName:  "ABC Traders",
				Route:         "Market Route",
				Salesman:      "Ramesh",
				Note:          "Groceries Stock",
				TotalAmount:   decimal.NewFromInt(1250),
				CashIn:        decimal.NewFromInt(1250),
				BankAmount:    decimal.Zero,
				PaymentMethod: "Cash",
				CreatedAt:     time.Now(),
			},
			{
				ID:            uuid.New(),
				InvoiceNumber: "LS-1002",
				Time:          "10:15",
				CustomerName:  "XYZ Store",
				Route:         "Highway Route",
				Salesman:      "Suresh",
				Note:          "Bulk Spices",
				TotalAmount:   decimal.NewFromInt(2500),
				CashIn:        decimal.Zero,
				BankAmount:    decimal.NewFromInt(2500),
				PaymentMethod: "Bank / UPI",
				CreatedAt:     time.Now(),
			},
			{
				ID:            uuid.New(),
				InvoiceNumber: "LS-1003",
				Time:          "11:40",
				CustomerName:  "Lakshmi Provisions",
				Route:         "Town Square",
				Salesman:      "Ramesh",
				Note:          "Edible Oils",
				TotalAmount:   decimal.NewFromInt(850),
				CashIn:        decimal.NewFromInt(850),
				BankAmount:    decimal.Zero,
				PaymentMethod: "Cash",
				CreatedAt:     time.Now(),
			},
		},
		CounterSales: []domain.CounterSaleReportItem{
			{
				ID:            uuid.New(),
				ReceiptNumber: "CS-1001",
				Time:          "09:15",
				Item:          "Sunflower Oil 1L x 2",
				TotalAmount:   decimal.NewFromInt(500),
				Cash:          decimal.NewFromInt(500),
				PaymentMethod: "Cash",
				CreatedAt:     time.Now(),
			},
			{
				ID:            uuid.New(),
				ReceiptNumber: "CS-1002",
				Time:          "12:20",
				Item:          "Basmati Rice 5kg Bag",
				TotalAmount:   decimal.NewFromInt(750),
				Cash:          decimal.NewFromInt(750),
				PaymentMethod: "Cash",
				CreatedAt:     time.Now(),
			},
		},
		Purchases: []domain.PurchaseReportItem{
			{
				ID:            uuid.New(),
				Reference:     "PUR-1001",
				Time:          "08:30",
				SupplierName:  "Supplier A Wholesale",
				Item:          "Wheat Flour 50kg Bags",
				Quantity:      5,
				TotalAmount:   decimal.NewFromInt(5000),
				TotalPaid:     decimal.NewFromInt(5000),
				PaymentStatus: "PAID",
				CreatedAt:     time.Now(),
			},
			{
				ID:            uuid.New(),
				Reference:     "PUR-1002",
				Time:          "14:20",
				SupplierName:  "Supplier B Distributors",
				Item:          "Dairy & Butter Cartons",
				Quantity:      10,
				TotalAmount:   decimal.NewFromInt(8000),
				TotalPaid:     decimal.NewFromInt(8000),
				PaymentStatus: "PAID",
				CreatedAt:     time.Now(),
			},
		},
		Expenses: []domain.ExpenseReportItem{
			{
				ID:            uuid.New(),
				Reference:     "EXP-1001",
				Time:          "10:00",
				Category:      "Electricity",
				Description:   "Shop Electricity Bill",
				Amount:        decimal.NewFromInt(2000),
				PaymentMethod: "Bank (HDFC)",
				CreatedAt:     time.Now(),
			},
			{
				ID:            uuid.New(),
				Reference:     "EXP-1002",
				Time:          "16:30",
				Category:      "Transport",
				Description:   "Delivery Auto Fare",
				Amount:        decimal.NewFromInt(500),
				PaymentMethod: "Cash",
				CreatedAt:     time.Now(),
			},
		},
		BankBalances: []domain.BankBalanceReportItem{
			{
				BankID:         uuid.New(),
				BankName:       "State Bank of India",
				AccountNumber:  "****1234",
				CurrentBalance: decimal.NewFromInt(85000),
			},
			{
				BankID:         uuid.New(),
				BankName:       "HDFC Bank",
				AccountNumber:  "****5678",
				CurrentBalance: decimal.NewFromInt(42500),
			},
			{
				BankID:         uuid.New(),
				BankName:       "ICICI Bank",
				AccountNumber:  "****9012",
				CurrentBalance: decimal.NewFromInt(18250),
			},
		},
		CashBalance: domain.CashBalanceReport{
			LineSalesCash:    decimal.NewFromInt(2100),
			CounterSalesCash: decimal.NewFromInt(1250),
			TotalCashInflow:  decimal.NewFromInt(3350),
			PurchaseCash:     decimal.NewFromInt(5000),
			ExpenseCash:      decimal.NewFromInt(500),
			OtherCashOutflow: decimal.Zero,
			TotalCashOutflow: decimal.NewFromInt(5500),
			NetDailyCashFlow: decimal.NewFromInt(-2150),
			ClosingCash:      decimal.NewFromInt(32450),
		},
		FinancialSummary: domain.ReportFinancialSummary{
			TotalLineSales:      decimal.NewFromInt(4600),
			TotalCounterSales:   decimal.NewFromInt(1250),
			TotalPurchases:      decimal.NewFromInt(13000),
			TotalExpenses:       decimal.NewFromInt(2500),
			CashBalance:         decimal.NewFromInt(32450),
			TotalBankBalance:    decimal.NewFromInt(145750),
			TotalAvailableFunds: decimal.NewFromInt(178200),
		},
	}

	pdfBytes, err := generator.GenerateDailyReportPDF(report)
	require.NoError(t, err)
	require.NotEmpty(t, pdfBytes)

	// Save to temporary test output
	tmpDir := os.TempDir()
	outPath := filepath.Join(tmpDir, "Royal_Kirana_Daily_Report_2026-10-07.pdf")
	err = os.WriteFile(outPath, pdfBytes, 0644)
	require.NoError(t, err)

	fi, err := os.Stat(outPath)
	require.NoError(t, err)
	require.Greater(t, fi.Size(), int64(1000))
}
