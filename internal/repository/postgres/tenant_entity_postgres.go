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
	"github.com/shopspring/decimal"
)

// ==========================================
// 1. TenantEmployeePostgres
// ==========================================

type TenantEmployeePostgres struct {
	pool *pgxpool.Pool
}

func NewTenantEmployeePostgres(pool *pgxpool.Pool) *TenantEmployeePostgres {
	return &TenantEmployeePostgres{pool: pool}
}

func (r *TenantEmployeePostgres) Create(ctx context.Context, emp *domain.TenantEmployee) error {
	query := `
		INSERT INTO tenant_employees (tenant_id, name, phone, salary, ot_rate, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query,
		emp.TenantID,
		emp.Name,
		emp.Phone,
		emp.Salary,
		emp.OTRate,
		emp.Status,
	).Scan(&emp.ID, &emp.CreatedAt, &emp.UpdatedAt)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to create tenant employee: %w", err))
	}
	return nil
}

func (r *TenantEmployeePostgres) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantEmployee, error) {
	query := `
		SELECT id, tenant_id, name, phone, salary, ot_rate, status, created_at, updated_at
		FROM tenant_employees
		WHERE tenant_id = $1 AND id = $2
	`
	exec := GetExecutor(ctx, r.pool)
	var emp domain.TenantEmployee
	err := exec.QueryRow(ctx, query, tenantID, id).Scan(
		&emp.ID,
		&emp.TenantID,
		&emp.Name,
		&emp.Phone,
		&emp.Salary,
		&emp.OTRate,
		&emp.Status,
		&emp.CreatedAt,
		&emp.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("employee not found within tenant")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get tenant employee: %w", err))
	}
	return &emp, nil
}

func (r *TenantEmployeePostgres) Update(ctx context.Context, emp *domain.TenantEmployee) error {
	query := `
		UPDATE tenant_employees
		SET name = $1, phone = $2, salary = $3, ot_rate = $4, status = $5, updated_at = NOW()
		WHERE tenant_id = $6 AND id = $7
		RETURNING updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query,
		emp.Name,
		emp.Phone,
		emp.Salary,
		emp.OTRate,
		emp.Status,
		emp.TenantID,
		emp.ID,
	).Scan(&emp.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return appErrors.NewNotFound("employee not found within tenant")
		}
		return appErrors.NewDatabase(fmt.Errorf("failed to update employee: %w", err))
	}
	return nil
}

func (r *TenantEmployeePostgres) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	query := `DELETE FROM tenant_employees WHERE tenant_id = $1 AND id = $2`
	exec := GetExecutor(ctx, r.pool)
	tag, err := exec.Exec(ctx, query, tenantID, id)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to delete tenant employee: %w", err))
	}
	if tag.RowsAffected() == 0 {
		return appErrors.NewNotFound("employee not found within tenant")
	}
	return nil
}

func (r *TenantEmployeePostgres) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search, status string) ([]domain.TenantEmployee, int64, error) {
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

	if status != "" {
		baseWhere += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	if search != "" {
		baseWhere += fmt.Sprintf(" AND (name ILIKE $%d OR phone ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tenant_employees %s", baseWhere)
	exec := GetExecutor(ctx, r.pool)
	var total int64
	if err := exec.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to count employees: %w", err))
	}

	listQuery := fmt.Sprintf(`
		SELECT id, tenant_id, name, phone, salary, ot_rate, status, created_at, updated_at
		FROM tenant_employees
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, baseWhere, argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := exec.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to list employees: %w", err))
	}
	defer rows.Close()

	employees := make([]domain.TenantEmployee, 0)
	for rows.Next() {
		var e domain.TenantEmployee
		if err := rows.Scan(
			&e.ID,
			&e.TenantID,
			&e.Name,
			&e.Phone,
			&e.Salary,
			&e.OTRate,
			&e.Status,
			&e.CreatedAt,
			&e.UpdatedAt,
		); err != nil {
			return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to scan employee: %w", err))
		}
		employees = append(employees, e)
	}

	return employees, total, nil
}

// ==========================================
// 2. TenantCustomerPostgres
// ==========================================

type TenantCustomerPostgres struct {
	pool *pgxpool.Pool
}

func NewTenantCustomerPostgres(pool *pgxpool.Pool) *TenantCustomerPostgres {
	return &TenantCustomerPostgres{pool: pool}
}

func (r *TenantCustomerPostgres) Create(ctx context.Context, cust *domain.TenantCustomer) error {
	query := `
		INSERT INTO tenant_customer (tenant_id, customer_name, phone, opening_balance, current_balance, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query,
		cust.TenantID,
		cust.CustomerName,
		cust.Phone,
		cust.OpeningBalance,
		cust.CurrentBalance,
		cust.Status,
	).Scan(&cust.ID, &cust.CreatedAt, &cust.UpdatedAt)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to create customer: %w", err))
	}
	return nil
}

func (r *TenantCustomerPostgres) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantCustomer, error) {
	query := `
		SELECT id, tenant_id, customer_name, phone, opening_balance, current_balance, status, created_at, updated_at
		FROM tenant_customer
		WHERE tenant_id = $1 AND id = $2
	`
	exec := GetExecutor(ctx, r.pool)
	var c domain.TenantCustomer
	err := exec.QueryRow(ctx, query, tenantID, id).Scan(
		&c.ID,
		&c.TenantID,
		&c.CustomerName,
		&c.Phone,
		&c.OpeningBalance,
		&c.CurrentBalance,
		&c.Status,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("customer not found within tenant")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get customer: %w", err))
	}
	return &c, nil
}

func (r *TenantCustomerPostgres) Update(ctx context.Context, cust *domain.TenantCustomer) error {
	query := `
		UPDATE tenant_customer
		SET customer_name = $1, phone = $2, status = $3, updated_at = NOW()
		WHERE tenant_id = $4 AND id = $5
		RETURNING updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query,
		cust.CustomerName,
		cust.Phone,
		cust.Status,
		cust.TenantID,
		cust.ID,
	).Scan(&cust.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return appErrors.NewNotFound("customer not found within tenant")
		}
		return appErrors.NewDatabase(fmt.Errorf("failed to update customer: %w", err))
	}
	return nil
}

func (r *TenantCustomerPostgres) AdjustBalance(ctx context.Context, tenantID, id uuid.UUID, delta decimal.Decimal) error {
	query := `
		UPDATE tenant_customer
		SET current_balance = current_balance + $1, updated_at = NOW()
		WHERE tenant_id = $2 AND id = $3
	`
	exec := GetExecutor(ctx, r.pool)
	tag, err := exec.Exec(ctx, query, delta, tenantID, id)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to adjust customer balance: %w", err))
	}
	if tag.RowsAffected() == 0 {
		return appErrors.NewNotFound("customer not found within tenant")
	}
	return nil
}

func (r *TenantCustomerPostgres) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	query := `DELETE FROM tenant_customer WHERE tenant_id = $1 AND id = $2`
	exec := GetExecutor(ctx, r.pool)
	tag, err := exec.Exec(ctx, query, tenantID, id)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to delete customer: %w", err))
	}
	if tag.RowsAffected() == 0 {
		return appErrors.NewNotFound("customer not found within tenant")
	}
	return nil
}

func (r *TenantCustomerPostgres) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search, status string) ([]domain.TenantCustomer, int64, error) {
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

	if status != "" {
		baseWhere += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	if search != "" {
		baseWhere += fmt.Sprintf(" AND (customer_name ILIKE $%d OR phone ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tenant_customer %s", baseWhere)
	exec := GetExecutor(ctx, r.pool)
	var total int64
	if err := exec.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to count customers: %w", err))
	}

	listQuery := fmt.Sprintf(`
		SELECT id, tenant_id, customer_name, phone, opening_balance, current_balance, status, created_at, updated_at
		FROM tenant_customer
		%s
		ORDER BY customer_name ASC
		LIMIT $%d OFFSET $%d
	`, baseWhere, argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := exec.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to list customers: %w", err))
	}
	defer rows.Close()

	customers := make([]domain.TenantCustomer, 0)
	for rows.Next() {
		var c domain.TenantCustomer
		if err := rows.Scan(
			&c.ID,
			&c.TenantID,
			&c.CustomerName,
			&c.Phone,
			&c.OpeningBalance,
			&c.CurrentBalance,
			&c.Status,
			&c.CreatedAt,
			&c.UpdatedAt,
		); err != nil {
			return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to scan customer: %w", err))
		}
		customers = append(customers, c)
	}

	return customers, total, nil
}

// ==========================================
// 3. TenantBankPostgres
// ==========================================

type TenantBankPostgres struct {
	pool *pgxpool.Pool
}

func NewTenantBankPostgres(pool *pgxpool.Pool) *TenantBankPostgres {
	return &TenantBankPostgres{pool: pool}
}

func (r *TenantBankPostgres) Create(ctx context.Context, bank *domain.TenantBank) error {
	query := `
		INSERT INTO tenant_bank (tenant_id, bank_name, account_number, ifsc_or_routing, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query,
		bank.TenantID,
		bank.BankName,
		bank.AccountNumber,
		bank.IFSC,
		bank.Status,
	).Scan(&bank.ID, &bank.CreatedAt, &bank.UpdatedAt)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to create tenant bank: %w", err))
	}
	return nil
}

func (r *TenantBankPostgres) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantBank, error) {
	query := `
		SELECT id, tenant_id, bank_name, account_number, ifsc_or_routing, status, created_at, updated_at
		FROM tenant_bank
		WHERE tenant_id = $1 AND id = $2
	`
	exec := GetExecutor(ctx, r.pool)
	var b domain.TenantBank
	err := exec.QueryRow(ctx, query, tenantID, id).Scan(
		&b.ID,
		&b.TenantID,
		&b.BankName,
		&b.AccountNumber,
		&b.IFSC,
		&b.Status,
		&b.CreatedAt,
		&b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("bank account not found within tenant")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get tenant bank: %w", err))
	}
	return &b, nil
}

func (r *TenantBankPostgres) Update(ctx context.Context, bank *domain.TenantBank) error {
	query := `
		UPDATE tenant_bank
		SET bank_name = $1, account_number = $2, ifsc_or_routing = $3, status = $4, updated_at = NOW()
		WHERE tenant_id = $5 AND id = $6
		RETURNING updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query,
		bank.BankName,
		bank.AccountNumber,
		bank.IFSC,
		bank.Status,
		bank.TenantID,
		bank.ID,
	).Scan(&bank.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return appErrors.NewNotFound("bank account not found within tenant")
		}
		return appErrors.NewDatabase(fmt.Errorf("failed to update bank account: %w", err))
	}
	return nil
}

func (r *TenantBankPostgres) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	query := `DELETE FROM tenant_bank WHERE tenant_id = $1 AND id = $2`
	exec := GetExecutor(ctx, r.pool)
	tag, err := exec.Exec(ctx, query, tenantID, id)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to delete bank account: %w", err))
	}
	if tag.RowsAffected() == 0 {
		return appErrors.NewNotFound("bank account not found within tenant")
	}
	return nil
}

func (r *TenantBankPostgres) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search, status string) ([]domain.TenantBank, int64, error) {
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

	if status != "" {
		baseWhere += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	if search != "" {
		baseWhere += fmt.Sprintf(" AND (bank_name ILIKE $%d OR account_number ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tenant_bank %s", baseWhere)
	exec := GetExecutor(ctx, r.pool)
	var total int64
	if err := exec.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to count banks: %w", err))
	}

	listQuery := fmt.Sprintf(`
		SELECT id, tenant_id, bank_name, account_number, ifsc_or_routing, status, created_at, updated_at
		FROM tenant_bank
		%s
		ORDER BY bank_name ASC
		LIMIT $%d OFFSET $%d
	`, baseWhere, argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := exec.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to list banks: %w", err))
	}
	defer rows.Close()

	banks := make([]domain.TenantBank, 0)
	for rows.Next() {
		var b domain.TenantBank
		if err := rows.Scan(
			&b.ID,
			&b.TenantID,
			&b.BankName,
			&b.AccountNumber,
			&b.IFSC,
			&b.Status,
			&b.CreatedAt,
			&b.UpdatedAt,
		); err != nil {
			return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to scan bank: %w", err))
		}
		banks = append(banks, b)
	}

	return banks, total, nil
}
