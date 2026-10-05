package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Varunjp/vyavsa/internal/domain"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	pQuery := `
		INSERT INTO tenant_purchase (tenant_id, customer_id, customer_name, item, quantity, total_amount, total_paid, total_pending)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
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
	).Scan(&purchase.ID, &purchase.CreatedAt, &purchase.UpdatedAt)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to create purchase: %w", err))
	}

	for i := range payments {
		payment := &payments[i]
		payment.TenantID = purchase.TenantID
		payment.PurchaseID = purchase.ID
		pmQuery := `
			INSERT INTO tenant_purchase_payment (tenant_id, purchase_id, payment_method, bank_id, bank_name, amount, note)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id, created_at, updated_at
		`
		if err := exec.QueryRow(ctx, pmQuery,
			payment.TenantID,
			payment.PurchaseID,
			payment.PaymentMethod,
			payment.BankID,
			payment.BankName,
			payment.Amount,
			payment.Note,
		).Scan(&payment.ID, &payment.CreatedAt, &payment.UpdatedAt); err != nil {
			return appErrors.NewDatabase(fmt.Errorf("failed to create purchase payment: %w", err))
		}
	}
	purchase.Payments = payments

	return nil
}

func (r *TenantPurchasePostgres) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantPurchase, error) {
	pQuery := `
		SELECT id, tenant_id, customer_id, customer_name, item, quantity, total_amount, total_paid, total_pending, created_at, updated_at
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
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("purchase not found within tenant")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get purchase: %w", err))
	}

	pmQuery := `
		SELECT id, tenant_id, purchase_id, payment_method, bank_id, bank_name, amount, note, created_at, updated_at
		FROM tenant_purchase_payment
		WHERE tenant_id = $1 AND purchase_id = $2
		ORDER BY created_at ASC
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
			&pm.PaymentMethod,
			&pm.BankID,
			&pm.BankName,
			&pm.Amount,
			&pm.Note,
			&pm.CreatedAt,
			&pm.UpdatedAt,
		); err != nil {
			return nil, appErrors.NewDatabase(fmt.Errorf("failed to scan purchase payment: %w", err))
		}
		p.Payments = append(p.Payments, pm)
	}

	return &p, nil
}

func (r *TenantPurchasePostgres) Update(ctx context.Context, purchase *domain.TenantPurchase) error {
	query := `
		UPDATE tenant_purchase
		SET customer_id = $1, customer_name = $2, item = $3, quantity = $4, total_amount = $5, total_paid = $6, total_pending = $7, updated_at = NOW()
		WHERE tenant_id = $8 AND id = $9
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

func (r *TenantPurchasePostgres) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, date string, search string) ([]domain.TenantPurchase, int64, error) {
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
		baseWhere += fmt.Sprintf(" AND created_at::date = $%d", argIdx)
		args = append(args, date)
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
		SELECT id, tenant_id, customer_id, customer_name, item, quantity, total_amount, total_paid, total_pending, created_at, updated_at
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
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to scan purchase: %w", err))
		}
		purchases = append(purchases, p)
	}

	return purchases, total, nil
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
	eQuery := `
		INSERT INTO tenant_expense (tenant_id, item, total_amount)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, eQuery,
		expense.TenantID,
		expense.Item,
		expense.TotalAmount,
	).Scan(&expense.ID, &expense.CreatedAt, &expense.UpdatedAt)
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
		SELECT id, tenant_id, item, total_amount, created_at, updated_at
		FROM tenant_expense
		WHERE tenant_id = $1 AND id = $2
	`
	exec := GetExecutor(ctx, r.pool)
	var e domain.TenantExpense
	err := exec.QueryRow(ctx, eQuery, tenantID, id).Scan(
		&e.ID,
		&e.TenantID,
		&e.Item,
		&e.TotalAmount,
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
	query := `
		UPDATE tenant_expense
		SET item = $1, total_amount = $2, updated_at = NOW()
		WHERE tenant_id = $3 AND id = $4
		RETURNING updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query,
		expense.Item,
		expense.TotalAmount,
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

	baseWhere := "WHERE tenant_id = $1"
	args := []any{tenantID}
	argIdx := 2

	if date != "" {
		baseWhere += fmt.Sprintf(" AND created_at::date = $%d", argIdx)
		args = append(args, date)
		argIdx++
	}

	if search != "" {
		baseWhere += fmt.Sprintf(" AND item ILIKE $%d", argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tenant_expense %s", baseWhere)
	exec := GetExecutor(ctx, r.pool)
	var total int64
	if err := exec.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to count expenses: %w", err))
	}

	listQuery := fmt.Sprintf(`
		SELECT id, tenant_id, item, total_amount, created_at, updated_at
		FROM tenant_expense
		%s
		ORDER BY created_at DESC
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
			&e.CreatedAt,
			&e.UpdatedAt,
		); err != nil {
			return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to scan expense: %w", err))
		}
		expenses = append(expenses, e)
	}

	return expenses, total, nil
}
