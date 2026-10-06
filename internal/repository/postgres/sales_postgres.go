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
)

// ==========================================
// 1. LineSalePostgres
// ==========================================

type LineSalePostgres struct {
	pool *pgxpool.Pool
}

func NewLineSalePostgres(pool *pgxpool.Pool) *LineSalePostgres {
	return &LineSalePostgres{pool: pool}
}

func (r *LineSalePostgres) Create(ctx context.Context, sale *domain.LineSale, payments []domain.LineSalePayment) error {
	saleQuery := `
		INSERT INTO line_sale (tenant_id, customer_id, customer_name, route, salesman, note, total_amount, total_cash_in, bank_amount, collected_amount, balance)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, saleQuery,
		sale.TenantID,
		sale.CustomerID,
		sale.CustomerName,
		sale.Route,
		sale.Salesman,
		sale.Note,
		sale.TotalAmount,
		sale.TotalCashIn,
		sale.BankAmount,
		sale.CollectedAmount,
		sale.Balance,
	).Scan(&sale.ID, &sale.CreatedAt, &sale.UpdatedAt)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to create line sale: %w", err))
	}

	for i := range payments {
		payment := &payments[i]
		payment.TenantID = sale.TenantID
		payment.LineSaleID = sale.ID
		pDate := payment.PaymentDate
		if pDate == "" {
			pDate = sale.CreatedAt.Format("2006-01-02")
		}
		pStatus := payment.Status
		if pStatus == "" {
			pStatus = "COMPLETED"
		}
		pmQuery := `
			INSERT INTO line_sale_payments (tenant_id, line_sale_id, payment_method, bank_id, bank_name, amount, payment_date, reference_id, status, note)
			VALUES ($1, $2, $3, $4, $5, $6, $7::date, $8, $9, $10)
			RETURNING id, payment_date::text, reference_id, status, created_at, updated_at
		`
		if err := exec.QueryRow(ctx, pmQuery,
			payment.TenantID,
			payment.LineSaleID,
			payment.PaymentMethod,
			payment.BankID,
			payment.BankName,
			payment.Amount,
			pDate,
			payment.ReferenceID,
			pStatus,
			payment.Note,
		).Scan(&payment.ID, &payment.PaymentDate, &payment.ReferenceID, &payment.Status, &payment.CreatedAt, &payment.UpdatedAt); err != nil {
			return appErrors.NewDatabase(fmt.Errorf("failed to create line sale payment: %w", err))
		}
	}
	sale.Payments = payments

	return nil
}

func (r *LineSalePostgres) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.LineSale, error) {
	saleQuery := `
		SELECT id, tenant_id, customer_id, customer_name, route, salesman, note, total_amount, total_cash_in, bank_amount, collected_amount, balance, created_at, updated_at
		FROM line_sale
		WHERE tenant_id = $1 AND id = $2
	`
	exec := GetExecutor(ctx, r.pool)
	var sale domain.LineSale
	err := exec.QueryRow(ctx, saleQuery, tenantID, id).Scan(
		&sale.ID,
		&sale.TenantID,
		&sale.CustomerID,
		&sale.CustomerName,
		&sale.Route,
		&sale.Salesman,
		&sale.Note,
		&sale.TotalAmount,
		&sale.TotalCashIn,
		&sale.BankAmount,
		&sale.CollectedAmount,
		&sale.Balance,
		&sale.CreatedAt,
		&sale.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("line sale not found within tenant")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get line sale: %w", err))
	}

	pmQuery := `
		SELECT id, tenant_id, line_sale_id, payment_method, bank_id, bank_name, amount,
		       COALESCE(payment_date::text, created_at::date::text) as payment_date,
		       COALESCE(reference_id, '') as reference_id,
		       COALESCE(status, 'COMPLETED') as status,
		       note, created_at, updated_at
		FROM line_sale_payments
		WHERE tenant_id = $1 AND line_sale_id = $2
		ORDER BY COALESCE(payment_date, created_at::date) ASC, created_at ASC
	`
	rows, err := exec.Query(ctx, pmQuery, tenantID, id)
	if err != nil {
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get line sale payments: %w", err))
	}
	defer rows.Close()

	sale.Payments = make([]domain.LineSalePayment, 0)
	for rows.Next() {
		var p domain.LineSalePayment
		if err := rows.Scan(
			&p.ID,
			&p.TenantID,
			&p.LineSaleID,
			&p.PaymentMethod,
			&p.BankID,
			&p.BankName,
			&p.Amount,
			&p.PaymentDate,
			&p.ReferenceID,
			&p.Status,
			&p.Note,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, appErrors.NewDatabase(fmt.Errorf("failed to scan line sale payment: %w", err))
		}
		sale.Payments = append(sale.Payments, p)
	}

	return &sale, nil
}

func (r *LineSalePostgres) GetByIDForUpdate(ctx context.Context, tenantID, id uuid.UUID) (*domain.LineSale, error) {
	saleQuery := `
		SELECT id, tenant_id, customer_id, customer_name, route, salesman, note, total_amount, total_cash_in, bank_amount, collected_amount, balance, created_at, updated_at
		FROM line_sale
		WHERE tenant_id = $1 AND id = $2
		FOR UPDATE
	`
	exec := GetExecutor(ctx, r.pool)
	var sale domain.LineSale
	err := exec.QueryRow(ctx, saleQuery, tenantID, id).Scan(
		&sale.ID,
		&sale.TenantID,
		&sale.CustomerID,
		&sale.CustomerName,
		&sale.Route,
		&sale.Salesman,
		&sale.Note,
		&sale.TotalAmount,
		&sale.TotalCashIn,
		&sale.BankAmount,
		&sale.CollectedAmount,
		&sale.Balance,
		&sale.CreatedAt,
		&sale.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("line sale not found within tenant")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get line sale for update: %w", err))
	}
	return &sale, nil
}

func (r *LineSalePostgres) CreatePayment(ctx context.Context, payment *domain.LineSalePayment) error {
	pDate := payment.PaymentDate
	if pDate == "" {
		pDate = time.Now().UTC().Format("2006-01-02")
	}
	pStatus := payment.Status
	if pStatus == "" {
		pStatus = "COMPLETED"
	}
	pmQuery := `
		INSERT INTO line_sale_payments (tenant_id, line_sale_id, payment_method, bank_id, bank_name, amount, payment_date, reference_id, status, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7::date, $8, $9, $10)
		RETURNING id, payment_date::text, reference_id, status, created_at, updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, pmQuery,
		payment.TenantID,
		payment.LineSaleID,
		payment.PaymentMethod,
		payment.BankID,
		payment.BankName,
		payment.Amount,
		pDate,
		payment.ReferenceID,
		pStatus,
		payment.Note,
	).Scan(&payment.ID, &payment.PaymentDate, &payment.ReferenceID, &payment.Status, &payment.CreatedAt, &payment.UpdatedAt)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to create line sale payment: %w", err))
	}
	return nil
}

func (r *LineSalePostgres) ListPaymentsByLineSaleID(ctx context.Context, tenantID, lineSaleID uuid.UUID) ([]domain.LineSalePayment, error) {
	pmQuery := `
		SELECT id, tenant_id, line_sale_id, payment_method, bank_id, bank_name, amount,
		       COALESCE(payment_date::text, created_at::date::text) as payment_date,
		       COALESCE(reference_id, '') as reference_id,
		       COALESCE(status, 'COMPLETED') as status,
		       note, created_at, updated_at
		FROM line_sale_payments
		WHERE tenant_id = $1 AND line_sale_id = $2
		ORDER BY COALESCE(payment_date, created_at::date) ASC, created_at ASC
	`
	exec := GetExecutor(ctx, r.pool)
	rows, err := exec.Query(ctx, pmQuery, tenantID, lineSaleID)
	if err != nil {
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to list line sale payments: %w", err))
	}
	defer rows.Close()

	payments := make([]domain.LineSalePayment, 0)
	for rows.Next() {
		var p domain.LineSalePayment
		if err := rows.Scan(
			&p.ID,
			&p.TenantID,
			&p.LineSaleID,
			&p.PaymentMethod,
			&p.BankID,
			&p.BankName,
			&p.Amount,
			&p.PaymentDate,
			&p.ReferenceID,
			&p.Status,
			&p.Note,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, appErrors.NewDatabase(fmt.Errorf("failed to scan line sale payment: %w", err))
		}
		payments = append(payments, p)
	}
	return payments, nil
}

func (r *LineSalePostgres) Update(ctx context.Context, sale *domain.LineSale) error {
	query := `
		UPDATE line_sale
		SET customer_name = $1, route = $2, salesman = $3, note = $4, total_amount = $5, total_cash_in = $6, bank_amount = $7, collected_amount = $8, balance = $9, updated_at = NOW()
		WHERE tenant_id = $10 AND id = $11
		RETURNING updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query,
		sale.CustomerName,
		sale.Route,
		sale.Salesman,
		sale.Note,
		sale.TotalAmount,
		sale.TotalCashIn,
		sale.BankAmount,
		sale.CollectedAmount,
		sale.Balance,
		sale.TenantID,
		sale.ID,
	).Scan(&sale.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return appErrors.NewNotFound("line sale not found within tenant")
		}
		return appErrors.NewDatabase(fmt.Errorf("failed to update line sale: %w", err))
	}
	return nil
}

func (r *LineSalePostgres) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	query := `DELETE FROM line_sale WHERE tenant_id = $1 AND id = $2`
	exec := GetExecutor(ctx, r.pool)
	tag, err := exec.Exec(ctx, query, tenantID, id)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to delete line sale: %w", err))
	}
	if tag.RowsAffected() == 0 {
		return appErrors.NewNotFound("line sale not found within tenant")
	}
	return nil
}

func (r *LineSalePostgres) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, date string, customerID *uuid.UUID, search string) ([]domain.LineSale, int64, error) {
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
		baseWhere += fmt.Sprintf(" AND (customer_name ILIKE $%d OR route ILIKE $%d OR salesman ILIKE $%d)", argIdx, argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM line_sale %s", baseWhere)
	exec := GetExecutor(ctx, r.pool)
	var total int64
	if err := exec.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to count line sales: %w", err))
	}

	listQuery := fmt.Sprintf(`
		SELECT id, tenant_id, customer_id, customer_name, route, salesman, note, total_amount, total_cash_in, bank_amount, collected_amount, balance, created_at, updated_at
		FROM line_sale
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, baseWhere, argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := exec.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to list line sales: %w", err))
	}
	defer rows.Close()

	sales := make([]domain.LineSale, 0)
	for rows.Next() {
		var s domain.LineSale
		if err := rows.Scan(
			&s.ID,
			&s.TenantID,
			&s.CustomerID,
			&s.CustomerName,
			&s.Route,
			&s.Salesman,
			&s.Note,
			&s.TotalAmount,
			&s.TotalCashIn,
			&s.BankAmount,
			&s.CollectedAmount,
			&s.Balance,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to scan line sale: %w", err))
		}
		sales = append(sales, s)
	}

	return sales, total, nil
}

// ==========================================
// 2. CounterSalePostgres
// ==========================================

type CounterSalePostgres struct {
	pool *pgxpool.Pool
}

func NewCounterSalePostgres(pool *pgxpool.Pool) *CounterSalePostgres {
	return &CounterSalePostgres{pool: pool}
}

func (r *CounterSalePostgres) Create(ctx context.Context, sale *domain.CounterSale, payments []domain.CounterSalePayment) error {
	saleQuery := `
		INSERT INTO counter_sale (tenant_id, item, price, total_amount, payment_method, cash, bank_amount, bank_id, collected_amount, account)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, saleQuery,
		sale.TenantID,
		sale.Item,
		sale.Price,
		sale.TotalAmount,
		sale.PaymentMethod,
		sale.Cash,
		sale.BankAmount,
		sale.BankID,
		sale.CollectedAmount,
		sale.Account,
	).Scan(&sale.ID, &sale.CreatedAt, &sale.UpdatedAt)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to create counter sale: %w", err))
	}

	for i := range payments {
		payment := &payments[i]
		payment.TenantID = sale.TenantID
		payment.CounterSaleID = sale.ID
		pDate := payment.PaymentDate
		if pDate == "" {
			pDate = time.Now().UTC().Format("2006-01-02")
		}
		pStatus := payment.Status
		if pStatus == "" {
			pStatus = "COMPLETED"
		}
		pmQuery := `
			INSERT INTO counter_sale_payments (tenant_id, counter_sale_id, payment_method, bank_id, bank_name, amount, payment_date, reference_id, status, note)
			VALUES ($1, $2, $3, $4, $5, $6, $7::date, $8, $9, $10)
			RETURNING id, payment_date::text, reference_id, status, created_at, updated_at
		`
		if err := exec.QueryRow(ctx, pmQuery,
			payment.TenantID,
			payment.CounterSaleID,
			payment.PaymentMethod,
			payment.BankID,
			payment.BankName,
			payment.Amount,
			pDate,
			payment.ReferenceID,
			pStatus,
			payment.Note,
		).Scan(&payment.ID, &payment.PaymentDate, &payment.ReferenceID, &payment.Status, &payment.CreatedAt, &payment.UpdatedAt); err != nil {
			return appErrors.NewDatabase(fmt.Errorf("failed to create counter sale payment: %w", err))
		}
	}
	sale.Payments = payments

	return nil
}

func (r *CounterSalePostgres) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.CounterSale, error) {
	saleQuery := `
		SELECT id, tenant_id, item, price, total_amount, payment_method, cash, bank_amount, bank_id, collected_amount, account, created_at, updated_at
		FROM counter_sale
		WHERE tenant_id = $1 AND id = $2
	`
	exec := GetExecutor(ctx, r.pool)
	var sale domain.CounterSale
	err := exec.QueryRow(ctx, saleQuery, tenantID, id).Scan(
		&sale.ID,
		&sale.TenantID,
		&sale.Item,
		&sale.Price,
		&sale.TotalAmount,
		&sale.PaymentMethod,
		&sale.Cash,
		&sale.BankAmount,
		&sale.BankID,
		&sale.CollectedAmount,
		&sale.Account,
		&sale.CreatedAt,
		&sale.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("counter sale not found within tenant")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get counter sale: %w", err))
	}

	pmQuery := `
		SELECT id, tenant_id, counter_sale_id, payment_method, bank_id, bank_name, amount,
		       COALESCE(payment_date::text, created_at::date::text) as payment_date,
		       COALESCE(reference_id, '') as reference_id,
		       COALESCE(status, 'COMPLETED') as status,
		       note, created_at, updated_at
		FROM counter_sale_payments
		WHERE tenant_id = $1 AND counter_sale_id = $2
		ORDER BY COALESCE(payment_date, created_at::date) ASC, created_at ASC
	`
	rows, err := exec.Query(ctx, pmQuery, tenantID, id)
	if err != nil {
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get counter sale payments: %w", err))
	}
	defer rows.Close()

	sale.Payments = make([]domain.CounterSalePayment, 0)
	for rows.Next() {
		var p domain.CounterSalePayment
		if err := rows.Scan(
			&p.ID,
			&p.TenantID,
			&p.CounterSaleID,
			&p.PaymentMethod,
			&p.BankID,
			&p.BankName,
			&p.Amount,
			&p.PaymentDate,
			&p.ReferenceID,
			&p.Status,
			&p.Note,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, appErrors.NewDatabase(fmt.Errorf("failed to scan counter sale payment: %w", err))
		}
		sale.Payments = append(sale.Payments, p)
	}

	return &sale, nil
}

func (r *CounterSalePostgres) GetByIDForUpdate(ctx context.Context, tenantID, id uuid.UUID) (*domain.CounterSale, error) {
	saleQuery := `
		SELECT id, tenant_id, item, price, total_amount, payment_method, cash, bank_amount, bank_id, collected_amount, account, created_at, updated_at
		FROM counter_sale
		WHERE tenant_id = $1 AND id = $2
		FOR UPDATE
	`
	exec := GetExecutor(ctx, r.pool)
	var sale domain.CounterSale
	err := exec.QueryRow(ctx, saleQuery, tenantID, id).Scan(
		&sale.ID,
		&sale.TenantID,
		&sale.Item,
		&sale.Price,
		&sale.TotalAmount,
		&sale.PaymentMethod,
		&sale.Cash,
		&sale.BankAmount,
		&sale.BankID,
		&sale.CollectedAmount,
		&sale.Account,
		&sale.CreatedAt,
		&sale.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("counter sale not found within tenant")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get counter sale for update: %w", err))
	}
	return &sale, nil
}

func (r *CounterSalePostgres) CreatePayment(ctx context.Context, payment *domain.CounterSalePayment) error {
	pDate := payment.PaymentDate
	if pDate == "" {
		pDate = time.Now().UTC().Format("2006-01-02")
	}
	pStatus := payment.Status
	if pStatus == "" {
		pStatus = "COMPLETED"
	}
	pmQuery := `
		INSERT INTO counter_sale_payments (tenant_id, counter_sale_id, payment_method, bank_id, bank_name, amount, payment_date, reference_id, status, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7::date, $8, $9, $10)
		RETURNING id, payment_date::text, reference_id, status, created_at, updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, pmQuery,
		payment.TenantID,
		payment.CounterSaleID,
		payment.PaymentMethod,
		payment.BankID,
		payment.BankName,
		payment.Amount,
		pDate,
		payment.ReferenceID,
		pStatus,
		payment.Note,
	).Scan(&payment.ID, &payment.PaymentDate, &payment.ReferenceID, &payment.Status, &payment.CreatedAt, &payment.UpdatedAt)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to create counter sale payment: %w", err))
	}
	return nil
}

func (r *CounterSalePostgres) ListPaymentsByCounterSaleID(ctx context.Context, tenantID, counterSaleID uuid.UUID) ([]domain.CounterSalePayment, error) {
	pmQuery := `
		SELECT id, tenant_id, counter_sale_id, payment_method, bank_id, bank_name, amount,
		       COALESCE(payment_date::text, created_at::date::text) as payment_date,
		       COALESCE(reference_id, '') as reference_id,
		       COALESCE(status, 'COMPLETED') as status,
		       note, created_at, updated_at
		FROM counter_sale_payments
		WHERE tenant_id = $1 AND counter_sale_id = $2
		ORDER BY COALESCE(payment_date, created_at::date) ASC, created_at ASC
	`
	exec := GetExecutor(ctx, r.pool)
	rows, err := exec.Query(ctx, pmQuery, tenantID, counterSaleID)
	if err != nil {
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to list counter sale payments: %w", err))
	}
	defer rows.Close()

	payments := make([]domain.CounterSalePayment, 0)
	for rows.Next() {
		var p domain.CounterSalePayment
		if err := rows.Scan(
			&p.ID,
			&p.TenantID,
			&p.CounterSaleID,
			&p.PaymentMethod,
			&p.BankID,
			&p.BankName,
			&p.Amount,
			&p.PaymentDate,
			&p.ReferenceID,
			&p.Status,
			&p.Note,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, appErrors.NewDatabase(fmt.Errorf("failed to scan counter sale payment: %w", err))
		}
		payments = append(payments, p)
	}

	return payments, nil
}

func (r *CounterSalePostgres) Update(ctx context.Context, sale *domain.CounterSale) error {
	query := `
		UPDATE counter_sale
		SET item = $1, price = $2, total_amount = $3, payment_method = $4, cash = $5, bank_amount = $6, bank_id = $7, collected_amount = $8, account = $9, updated_at = NOW()
		WHERE tenant_id = $10 AND id = $11
		RETURNING updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query,
		sale.Item,
		sale.Price,
		sale.TotalAmount,
		sale.PaymentMethod,
		sale.Cash,
		sale.BankAmount,
		sale.BankID,
		sale.CollectedAmount,
		sale.Account,
		sale.TenantID,
		sale.ID,
	).Scan(&sale.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return appErrors.NewNotFound("counter sale not found within tenant")
		}
		return appErrors.NewDatabase(fmt.Errorf("failed to update counter sale: %w", err))
	}
	return nil
}

func (r *CounterSalePostgres) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	query := `DELETE FROM counter_sale WHERE tenant_id = $1 AND id = $2`
	exec := GetExecutor(ctx, r.pool)
	tag, err := exec.Exec(ctx, query, tenantID, id)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to delete counter sale: %w", err))
	}
	if tag.RowsAffected() == 0 {
		return appErrors.NewNotFound("counter sale not found within tenant")
	}
	return nil
}

func (r *CounterSalePostgres) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, date string, search string) ([]domain.CounterSale, int64, error) {
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

	if search != "" {
		baseWhere += fmt.Sprintf(" AND (item ILIKE $%d OR payment_method ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM counter_sale %s", baseWhere)
	exec := GetExecutor(ctx, r.pool)
	var total int64
	if err := exec.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to count counter sales: %w", err))
	}

	listQuery := fmt.Sprintf(`
		SELECT id, tenant_id, item, price, total_amount, payment_method, cash, bank_amount, bank_id, collected_amount, account, created_at, updated_at
		FROM counter_sale
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, baseWhere, argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := exec.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to list counter sales: %w", err))
	}
	defer rows.Close()

	sales := make([]domain.CounterSale, 0)
	for rows.Next() {
		var s domain.CounterSale
		if err := rows.Scan(
			&s.ID,
			&s.TenantID,
			&s.Item,
			&s.Price,
			&s.TotalAmount,
			&s.PaymentMethod,
			&s.Cash,
			&s.BankAmount,
			&s.BankID,
			&s.CollectedAmount,
			&s.Account,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to scan counter sale: %w", err))
		}
		sales = append(sales, s)
	}

	return sales, total, nil
}
