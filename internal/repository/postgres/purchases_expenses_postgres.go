package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Varunjp/vyavsa/internal/domain"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// ==========================================
// 1. TenantPurchasePostgres
// ==========================================

type TenantPurchasePostgres struct {
	pool *pgxpool.Pool
}

func NewTenantPurchasePostgres(pool *pgxpool.Pool) *TenantPurchasePostgres {
	return &TenantPurchasePostgres{pool: pool}
}

func (r *TenantPurchasePostgres) Create(ctx context.Context, purchase *domain.TenantPurchase, payments []domain.TenantPurchasePayment) error {
	if purchase.PaymentStatus == "" {
		purchase.PaymentStatus = domain.ComputePurchasePaymentStatus(purchase.TotalAmount, purchase.TotalPaid)
	}
	purchase.OutstandingAmount = purchase.TotalPending

	pQuery := `
		INSERT INTO tenant_purchase (tenant_id, customer_id, customer_name, item, quantity, total_amount, total_paid, total_pending, payment_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, pQuery,
		purchase.TenantID,
		purchase.CustomerID,
		purchase.CustomerName,
		purchase.Item,
		purchase.Quantity,
		purchase.TotalAmount,
		purchase.TotalPaid,
		purchase.TotalPending,
		purchase.PaymentStatus,
	).Scan(&purchase.ID, &purchase.CreatedAt, &purchase.UpdatedAt)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to create purchase: %w", err))
	}

	for i := range payments {
		payment := &payments[i]
		payment.TenantID = purchase.TenantID
		payment.PurchaseID = purchase.ID
		if payment.CustomerID == nil {
			payment.CustomerID = purchase.CustomerID
		}
		pDate := payment.PaymentDate
		if pDate == "" {
			pDate = time.Now().UTC().Format("2006-01-02")
		}
		pStatus := payment.Status
		if pStatus == "" {
			pStatus = "COMPLETED"
		}
		pmQuery := `
			INSERT INTO tenant_purchase_payment (tenant_id, purchase_id, customer_id, payment_method, bank_id, bank_name, amount, payment_date, reference_id, status, note, is_settlement)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8::date, $9, $10, $11, $12)
			RETURNING id, payment_date::text, reference_id, status, created_at, updated_at
		`
		if err := exec.QueryRow(ctx, pmQuery,
			payment.TenantID,
			payment.PurchaseID,
			payment.CustomerID,
			payment.PaymentMethod,
			payment.BankID,
			payment.BankName,
			payment.Amount,
			pDate,
			payment.ReferenceID,
			pStatus,
			payment.Note,
			payment.IsSettlement,
		).Scan(&payment.ID, &payment.PaymentDate, &payment.ReferenceID, &payment.Status, &payment.CreatedAt, &payment.UpdatedAt); err != nil {
			return appErrors.NewDatabase(fmt.Errorf("failed to create purchase payment: %w", err))
		}
	}
	purchase.Payments = payments

	return nil
}

func (r *TenantPurchasePostgres) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantPurchase, error) {
	pQuery := `
		SELECT id, tenant_id, customer_id, customer_name, item, quantity, total_amount, total_paid, total_pending, payment_status, created_at, updated_at
		FROM tenant_purchase
		WHERE tenant_id = $1 AND id = $2
	`
	exec := GetExecutor(ctx, r.pool)
	var p domain.TenantPurchase
	err := exec.QueryRow(ctx, pQuery, tenantID, id).Scan(
		&p.ID,
		&p.TenantID,
		&p.CustomerID,
		&p.CustomerName,
		&p.Item,
		&p.Quantity,
		&p.TotalAmount,
		&p.TotalPaid,
		&p.TotalPending,
		&p.PaymentStatus,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("purchase not found within tenant")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get purchase: %w", err))
	}
	p.OutstandingAmount = p.TotalPending

	pmQuery := `
		SELECT id, tenant_id, purchase_id, customer_id, payment_method, bank_id, bank_name, amount,
		       COALESCE(payment_date::text, created_at::date::text) as payment_date,
		       COALESCE(reference_id, '') as reference_id,
		       COALESCE(status, 'COMPLETED') as status,
		       note, is_settlement, created_at, updated_at
		FROM tenant_purchase_payment
		WHERE tenant_id = $1 AND purchase_id = $2
		ORDER BY COALESCE(payment_date, created_at::date) ASC, created_at ASC
	`
	rows, err := exec.Query(ctx, pmQuery, tenantID, id)
	if err != nil {
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get purchase payments: %w", err))
	}
	defer rows.Close()

	p.Payments = make([]domain.TenantPurchasePayment, 0)
	for rows.Next() {
		var pm domain.TenantPurchasePayment
		if err := rows.Scan(
			&pm.ID,
			&pm.TenantID,
			&pm.PurchaseID,
			&pm.CustomerID,
			&pm.PaymentMethod,
			&pm.BankID,
			&pm.BankName,
			&pm.Amount,
			&pm.PaymentDate,
			&pm.ReferenceID,
			&pm.Status,
			&pm.Note,
			&pm.IsSettlement,
			&pm.CreatedAt,
			&pm.UpdatedAt,
		); err != nil {
			return nil, appErrors.NewDatabase(fmt.Errorf("failed to scan purchase payment: %w", err))
		}
		p.Payments = append(p.Payments, pm)
	}

	return &p, nil
}

func (r *TenantPurchasePostgres) GetByIDForUpdate(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantPurchase, error) {
	pQuery := `
		SELECT id, tenant_id, customer_id, customer_name, item, quantity, total_amount, total_paid, total_pending, payment_status, created_at, updated_at
		FROM tenant_purchase
		WHERE tenant_id = $1 AND id = $2
		FOR UPDATE
	`
	exec := GetExecutor(ctx, r.pool)
	var p domain.TenantPurchase
	err := exec.QueryRow(ctx, pQuery, tenantID, id).Scan(
		&p.ID,
		&p.TenantID,
		&p.CustomerID,
		&p.CustomerName,
		&p.Item,
		&p.Quantity,
		&p.TotalAmount,
		&p.TotalPaid,
		&p.TotalPending,
		&p.PaymentStatus,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("purchase not found within tenant")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get purchase for update: %w", err))
	}
	p.OutstandingAmount = p.TotalPending
	return &p, nil
}

func (r *TenantPurchasePostgres) GetCustomerPurchasesForUpdate(ctx context.Context, tenantID, customerID uuid.UUID) ([]domain.TenantPurchase, error) {
	pQuery := `
		SELECT id, tenant_id, customer_id, customer_name, item, quantity, total_amount, total_paid, total_pending, payment_status, created_at, updated_at
		FROM tenant_purchase
		WHERE tenant_id = $1 AND customer_id = $2 AND total_pending > 0
		ORDER BY created_at ASC
		FOR UPDATE
	`
	exec := GetExecutor(ctx, r.pool)
	rows, err := exec.Query(ctx, pQuery, tenantID, customerID)
	if err != nil {
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get customer purchases for update: %w", err))
	}
	defer rows.Close()

	purchases := make([]domain.TenantPurchase, 0)
	for rows.Next() {
		var p domain.TenantPurchase
		if err := rows.Scan(
			&p.ID,
			&p.TenantID,
			&p.CustomerID,
			&p.CustomerName,
			&p.Item,
			&p.Quantity,
			&p.TotalAmount,
			&p.TotalPaid,
			&p.TotalPending,
			&p.PaymentStatus,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, appErrors.NewDatabase(fmt.Errorf("failed to scan customer purchase for update: %w", err))
		}
		p.OutstandingAmount = p.TotalPending
		purchases = append(purchases, p)
	}

	return purchases, nil
}

func (r *TenantPurchasePostgres) Update(ctx context.Context, purchase *domain.TenantPurchase) error {
	if purchase.PaymentStatus == "" {
		purchase.PaymentStatus = domain.ComputePurchasePaymentStatus(purchase.TotalAmount, purchase.TotalPaid)
	}
	purchase.OutstandingAmount = purchase.TotalPending

	query := `
		UPDATE tenant_purchase
		SET customer_id = $1, customer_name = $2, item = $3, quantity = $4, total_amount = $5, total_paid = $6, total_pending = $7, payment_status = $8, updated_at = NOW()
		WHERE tenant_id = $9 AND id = $10
		RETURNING updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query,
		purchase.CustomerID,
		purchase.CustomerName,
		purchase.Item,
		purchase.Quantity,
		purchase.TotalAmount,
		purchase.TotalPaid,
		purchase.TotalPending,
		purchase.PaymentStatus,
		purchase.TenantID,
		purchase.ID,
	).Scan(&purchase.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return appErrors.NewNotFound("purchase not found within tenant")
		}
		return appErrors.NewDatabase(fmt.Errorf("failed to update purchase: %w", err))
	}
	return nil
}

func (r *TenantPurchasePostgres) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	query := `DELETE FROM tenant_purchase WHERE tenant_id = $1 AND id = $2`
	exec := GetExecutor(ctx, r.pool)
	tag, err := exec.Exec(ctx, query, tenantID, id)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to delete purchase: %w", err))
	}
	if tag.RowsAffected() == 0 {
		return appErrors.NewNotFound("purchase not found within tenant")
	}
	return nil
}

func (r *TenantPurchasePostgres) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, date string, customerID *uuid.UUID, search string) ([]domain.TenantPurchase, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	baseWhere := "WHERE tenant_id = $1"
	args := []any{tenantID}
	argIdx := 2

	if date != "" {
		if start, end, err := ParseDateRangeUTC(date); err == nil {
			baseWhere += fmt.Sprintf(" AND created_at >= $%d AND created_at < $%d", argIdx, argIdx+1)
			args = append(args, start, end)
			argIdx += 2
		} else {
			baseWhere += fmt.Sprintf(" AND created_at::date = $%d", argIdx)
			args = append(args, date)
			argIdx++
		}
	}

	if customerID != nil && *customerID != uuid.Nil {
		baseWhere += fmt.Sprintf(" AND customer_id = $%d", argIdx)
		args = append(args, *customerID)
		argIdx++
	}

	if search != "" {
		baseWhere += fmt.Sprintf(" AND (item ILIKE $%d OR customer_name ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tenant_purchase %s", baseWhere)
	exec := GetExecutor(ctx, r.pool)
	var total int64
	if err := exec.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to count purchases: %w", err))
	}

	listQuery := fmt.Sprintf(`
		SELECT id, tenant_id, customer_id, customer_name, item, quantity, total_amount, total_paid, total_pending, payment_status, created_at, updated_at
		FROM tenant_purchase
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, baseWhere, argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := exec.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to list purchases: %w", err))
	}
	defer rows.Close()

	purchases := make([]domain.TenantPurchase, 0)
	for rows.Next() {
		var p domain.TenantPurchase
		if err := rows.Scan(
			&p.ID,
			&p.TenantID,
			&p.CustomerID,
			&p.CustomerName,
			&p.Item,
			&p.Quantity,
			&p.TotalAmount,
			&p.TotalPaid,
			&p.TotalPending,
			&p.PaymentStatus,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to scan purchase: %w", err))
		}
		p.OutstandingAmount = p.TotalPending
		purchases = append(purchases, p)
	}

	return purchases, total, nil
}

func (r *TenantPurchasePostgres) CreatePayment(ctx context.Context, payment *domain.TenantPurchasePayment) error {
	pDate := payment.PaymentDate
	if pDate == "" {
		pDate = time.Now().UTC().Format("2006-01-02")
	}
	pStatus := payment.Status
	if pStatus == "" {
		pStatus = "COMPLETED"
	}
	pmQuery := `
		INSERT INTO tenant_purchase_payment (tenant_id, purchase_id, customer_id, payment_method, bank_id, bank_name, amount, payment_date, reference_id, status, note, is_settlement)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::date, $9, $10, $11, $12)
		RETURNING id, payment_date::text, reference_id, status, created_at, updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, pmQuery,
		payment.TenantID,
		payment.PurchaseID,
		payment.CustomerID,
		payment.PaymentMethod,
		payment.BankID,
		payment.BankName,
		payment.Amount,
		pDate,
		payment.ReferenceID,
		pStatus,
		payment.Note,
		payment.IsSettlement,
	).Scan(&payment.ID, &payment.PaymentDate, &payment.ReferenceID, &payment.Status, &payment.CreatedAt, &payment.UpdatedAt)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to create purchase payment: %w", err))
	}
	return nil
}

func (r *TenantPurchasePostgres) ListPaymentsByPurchaseID(ctx context.Context, tenantID, purchaseID uuid.UUID) ([]domain.TenantPurchasePayment, error) {
	pmQuery := `
		SELECT id, tenant_id, purchase_id, customer_id, payment_method, bank_id, bank_name, amount,
		       COALESCE(payment_date::text, created_at::date::text) as payment_date,
		       COALESCE(reference_id, '') as reference_id,
		       COALESCE(status, 'COMPLETED') as status,
		       note, is_settlement, created_at, updated_at
		FROM tenant_purchase_payment
		WHERE tenant_id = $1 AND purchase_id = $2
		ORDER BY COALESCE(payment_date, created_at::date) ASC, created_at ASC
	`
	exec := GetExecutor(ctx, r.pool)
	rows, err := exec.Query(ctx, pmQuery, tenantID, purchaseID)
	if err != nil {
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get purchase payments: %w", err))
	}
	defer rows.Close()

	payments := make([]domain.TenantPurchasePayment, 0)
	for rows.Next() {
		var pm domain.TenantPurchasePayment
		if err := rows.Scan(
			&pm.ID,
			&pm.TenantID,
			&pm.PurchaseID,
			&pm.CustomerID,
			&pm.PaymentMethod,
			&pm.BankID,
			&pm.BankName,
			&pm.Amount,
			&pm.PaymentDate,
			&pm.ReferenceID,
			&pm.Status,
			&pm.Note,
			&pm.IsSettlement,
			&pm.CreatedAt,
			&pm.UpdatedAt,
		); err != nil {
			return nil, appErrors.NewDatabase(fmt.Errorf("failed to scan purchase payment: %w", err))
		}
		payments = append(payments, pm)
	}
	return payments, nil
}

func (r *TenantPurchasePostgres) ListPaymentsByCustomerID(ctx context.Context, tenantID, customerID uuid.UUID) ([]domain.TenantPurchasePayment, error) {
	pmQuery := `
		SELECT p.id, p.tenant_id, p.purchase_id, p.customer_id, p.payment_method, p.bank_id, p.bank_name, p.amount,
		       COALESCE(p.payment_date::text, p.created_at::date::text) as payment_date,
		       COALESCE(p.reference_id, '') as reference_id,
		       COALESCE(p.status, 'COMPLETED') as status,
		       p.note, p.is_settlement, p.created_at, p.updated_at
		FROM tenant_purchase_payment p
		LEFT JOIN tenant_purchase tp ON tp.id = p.purchase_id
		WHERE p.tenant_id = $1 AND (p.customer_id = $2 OR tp.customer_id = $2)
		ORDER BY COALESCE(p.payment_date, p.created_at::date) ASC, p.created_at ASC
	`
	exec := GetExecutor(ctx, r.pool)
	rows, err := exec.Query(ctx, pmQuery, tenantID, customerID)
	if err != nil {
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to list customer purchase payments: %w", err))
	}
	defer rows.Close()

	payments := make([]domain.TenantPurchasePayment, 0)
	for rows.Next() {
		var pm domain.TenantPurchasePayment
		if err := rows.Scan(
			&pm.ID,
			&pm.TenantID,
			&pm.PurchaseID,
			&pm.CustomerID,
			&pm.PaymentMethod,
			&pm.BankID,
			&pm.BankName,
			&pm.Amount,
			&pm.PaymentDate,
			&pm.ReferenceID,
			&pm.Status,
			&pm.Note,
			&pm.IsSettlement,
			&pm.CreatedAt,
			&pm.UpdatedAt,
		); err != nil {
			return nil, appErrors.NewDatabase(fmt.Errorf("failed to scan customer purchase payment: %w", err))
		}
		payments = append(payments, pm)
	}
	return payments, nil
}

func (r *TenantPurchasePostgres) GetCustomerPayableSummary(ctx context.Context, tenantID, customerID uuid.UUID) (totalPurchases, totalPaid, outstandingPayable decimal.Decimal, err error) {
	query := `
		SELECT
			COALESCE(SUM(total_amount), 0.00),
			COALESCE(SUM(total_paid), 0.00),
			COALESCE(SUM(total_pending), 0.00)
		FROM tenant_purchase
		WHERE tenant_id = $1 AND customer_id = $2
	`
	exec := GetExecutor(ctx, r.pool)
	err = exec.QueryRow(ctx, query, tenantID, customerID).Scan(&totalPurchases, &totalPaid, &outstandingPayable)
	if err != nil {
		return decimal.Zero, decimal.Zero, decimal.Zero, appErrors.NewDatabase(fmt.Errorf("failed to get customer payable summary: %w", err))
	}
	return totalPurchases, totalPaid, outstandingPayable, nil
}

func (r *TenantPurchasePostgres) GetCustomerPayableSummariesBatch(ctx context.Context, tenantID uuid.UUID, customerIDs []uuid.UUID) (map[uuid.UUID]domain.CustomerPayableSummary, error) {
	result := make(map[uuid.UUID]domain.CustomerPayableSummary)
	if len(customerIDs) == 0 {
		return result, nil
	}

	query := `
		SELECT
			customer_id,
			COALESCE(SUM(total_amount), 0.00),
			COALESCE(SUM(total_paid), 0.00),
			COALESCE(SUM(total_pending), 0.00)
		FROM tenant_purchase
		WHERE tenant_id = $1 AND customer_id = ANY($2)
		GROUP BY customer_id
	`
	exec := GetExecutor(ctx, r.pool)
	rows, err := exec.Query(ctx, query, tenantID, customerIDs)
	if err != nil {
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get customer payable summaries batch: %w", err))
	}
	defer rows.Close()

	for rows.Next() {
		var custID *uuid.UUID
		var summary domain.CustomerPayableSummary
		if err := rows.Scan(&custID, &summary.TotalPurchases, &summary.TotalPaid, &summary.OutstandingPayable); err != nil {
			return nil, appErrors.NewDatabase(fmt.Errorf("failed to scan customer payable summary: %w", err))
		}
		if custID != nil {
			result[*custID] = summary
		}
	}
	if err := rows.Err(); err != nil {
		return nil, appErrors.NewDatabase(fmt.Errorf("rows error in customer payable summaries batch: %w", err))
	}

	return result, nil
}

// ==========================================
// 2. TenantExpensePostgres
// ==========================================

type TenantExpensePostgres struct {
	pool *pgxpool.Pool
}

func NewTenantExpensePostgres(pool *pgxpool.Pool) *TenantExpensePostgres {
	return &TenantExpensePostgres{pool: pool}
}

func (r *TenantExpensePostgres) Create(ctx context.Context, expense *domain.TenantExpense, payments []domain.TenantExpensePayment) error {
	cat := expense.Category
	if cat == "" {
		cat = "general"
	}
	eQuery := `
		INSERT INTO tenant_expense (tenant_id, item, total_amount, category, employee_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, category, created_at, updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, eQuery,
		expense.TenantID,
		expense.Item,
		expense.TotalAmount,
		cat,
		expense.EmployeeID,
	).Scan(&expense.ID, &expense.Category, &expense.CreatedAt, &expense.UpdatedAt)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to create expense: %w", err))
	}

	for i := range payments {
		payment := &payments[i]
		payment.TenantID = expense.TenantID
		payment.ExpenseID = expense.ID
		pmQuery := `
			INSERT INTO tenant_expense_payment (tenant_id, expense_id, payment_method, bank_id, bank_name, amount, note)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id, created_at, updated_at
		`
		if err := exec.QueryRow(ctx, pmQuery,
			payment.TenantID,
			payment.ExpenseID,
			payment.PaymentMethod,
			payment.BankID,
			payment.BankName,
			payment.Amount,
			payment.Note,
		).Scan(&payment.ID, &payment.CreatedAt, &payment.UpdatedAt); err != nil {
			return appErrors.NewDatabase(fmt.Errorf("failed to create expense payment: %w", err))
		}
	}
	expense.Payments = payments

	return nil
}

func (r *TenantExpensePostgres) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantExpense, error) {
	eQuery := `
		SELECT e.id, e.tenant_id, e.item, e.total_amount, COALESCE(e.category, 'general'), e.employee_id, COALESCE(emp.name, ''), e.created_at, e.updated_at
		FROM tenant_expense e
		LEFT JOIN tenant_employees emp ON emp.id = e.employee_id
		WHERE e.tenant_id = $1 AND e.id = $2
	`
	exec := GetExecutor(ctx, r.pool)
	var e domain.TenantExpense
	err := exec.QueryRow(ctx, eQuery, tenantID, id).Scan(
		&e.ID,
		&e.TenantID,
		&e.Item,
		&e.TotalAmount,
		&e.Category,
		&e.EmployeeID,
		&e.EmployeeName,
		&e.CreatedAt,
		&e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("expense not found within tenant")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get expense: %w", err))
	}

	pmQuery := `
		SELECT id, tenant_id, expense_id, payment_method, bank_id, bank_name, amount, note, created_at, updated_at
		FROM tenant_expense_payment
		WHERE tenant_id = $1 AND expense_id = $2
		ORDER BY created_at ASC
	`
	rows, err := exec.Query(ctx, pmQuery, tenantID, id)
	if err != nil {
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get expense payments: %w", err))
	}
	defer rows.Close()

	e.Payments = make([]domain.TenantExpensePayment, 0)
	for rows.Next() {
		var pm domain.TenantExpensePayment
		if err := rows.Scan(
			&pm.ID,
			&pm.TenantID,
			&pm.ExpenseID,
			&pm.PaymentMethod,
			&pm.BankID,
			&pm.BankName,
			&pm.Amount,
			&pm.Note,
			&pm.CreatedAt,
			&pm.UpdatedAt,
		); err != nil {
			return nil, appErrors.NewDatabase(fmt.Errorf("failed to scan expense payment: %w", err))
		}
		e.Payments = append(e.Payments, pm)
	}

	return &e, nil
}

func (r *TenantExpensePostgres) Update(ctx context.Context, expense *domain.TenantExpense) error {
	cat := expense.Category
	if cat == "" {
		cat = "general"
	}
	query := `
		UPDATE tenant_expense
		SET item = $1, total_amount = $2, category = $3, employee_id = $4, updated_at = NOW()
		WHERE tenant_id = $5 AND id = $6
		RETURNING updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query,
		expense.Item,
		expense.TotalAmount,
		cat,
		expense.EmployeeID,
		expense.TenantID,
		expense.ID,
	).Scan(&expense.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return appErrors.NewNotFound("expense not found within tenant")
		}
		return appErrors.NewDatabase(fmt.Errorf("failed to update expense: %w", err))
	}
	return nil
}

func (r *TenantExpensePostgres) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	query := `DELETE FROM tenant_expense WHERE tenant_id = $1 AND id = $2`
	exec := GetExecutor(ctx, r.pool)
	tag, err := exec.Exec(ctx, query, tenantID, id)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to delete expense: %w", err))
	}
	if tag.RowsAffected() == 0 {
		return appErrors.NewNotFound("expense not found within tenant")
	}
	return nil
}

func (r *TenantExpensePostgres) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, date string, search string) ([]domain.TenantExpense, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	baseWhere := "WHERE e.tenant_id = $1"
	args := []any{tenantID}
	argIdx := 2

	if date != "" {
		if start, end, err := ParseDateRangeUTC(date); err == nil {
			baseWhere += fmt.Sprintf(" AND e.created_at >= $%d AND e.created_at < $%d", argIdx, argIdx+1)
			args = append(args, start, end)
			argIdx += 2
		} else {
			baseWhere += fmt.Sprintf(" AND e.created_at::date = $%d", argIdx)
			args = append(args, date)
			argIdx++
		}
	}

	if search != "" {
		baseWhere += fmt.Sprintf(" AND (e.item ILIKE $%d OR COALESCE(e.category, '') ILIKE $%d OR COALESCE(emp.name, '') ILIKE $%d)", argIdx, argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM tenant_expense e
		LEFT JOIN tenant_employees emp ON emp.id = e.employee_id
		%s
	`, baseWhere)
	exec := GetExecutor(ctx, r.pool)
	var total int64
	if err := exec.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to count expenses: %w", err))
	}

	listQuery := fmt.Sprintf(`
		SELECT e.id, e.tenant_id, e.item, e.total_amount, COALESCE(e.category, 'general'), e.employee_id, COALESCE(emp.name, ''), e.created_at, e.updated_at
		FROM tenant_expense e
		LEFT JOIN tenant_employees emp ON emp.id = e.employee_id
		%s
		ORDER BY e.created_at DESC
		LIMIT $%d OFFSET $%d
	`, baseWhere, argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := exec.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to list expenses: %w", err))
	}
	defer rows.Close()

	expenses := make([]domain.TenantExpense, 0)
	for rows.Next() {
		var e domain.TenantExpense
		if err := rows.Scan(
			&e.ID,
			&e.TenantID,
			&e.Item,
			&e.TotalAmount,
			&e.Category,
			&e.EmployeeID,
			&e.EmployeeName,
			&e.CreatedAt,
			&e.UpdatedAt,
		); err != nil {
			return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to scan expense: %w", err))
		}
		expenses = append(expenses, e)
	}

	return expenses, total, nil
}
