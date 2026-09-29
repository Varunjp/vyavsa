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
// 1. AttendancePostgres
// ==========================================

type AttendancePostgres struct {
	pool *pgxpool.Pool
}

func NewAttendancePostgres(pool *pgxpool.Pool) *AttendancePostgres {
	return &AttendancePostgres{pool: pool}
}

func (r *AttendancePostgres) Upsert(ctx context.Context, att *domain.Attendance) error {
	query := `
		INSERT INTO attendance (tenant_id, employee_id, date, status, ot, advance)
		VALUES ($1, $2, $3::date, $4, $5, $6)
		ON CONFLICT (tenant_id, employee_id, date)
		DO UPDATE SET status = EXCLUDED.status, ot = EXCLUDED.ot, advance = EXCLUDED.advance, updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query,
		att.TenantID,
		att.EmployeeID,
		att.Date,
		att.Status,
		att.OT,
		att.Advance,
	).Scan(&att.ID, &att.CreatedAt, &att.UpdatedAt)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to upsert attendance: %w", err))
	}
	return nil
}

func (r *AttendancePostgres) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Attendance, error) {
	query := `
		SELECT a.id, a.tenant_id, a.employee_id, COALESCE(e.name, '') as employee_name,
		       a.date::text, a.status, a.ot, a.advance, a.created_at, a.updated_at
		FROM attendance a
		LEFT JOIN tenant_employees e ON e.id = a.employee_id
		WHERE a.tenant_id = $1 AND a.id = $2
	`
	exec := GetExecutor(ctx, r.pool)
	var a domain.Attendance
	err := exec.QueryRow(ctx, query, tenantID, id).Scan(
		&a.ID,
		&a.TenantID,
		&a.EmployeeID,
		&a.EmployeeName,
		&a.Date,
		&a.Status,
		&a.OT,
		&a.Advance,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("attendance record not found within tenant")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get attendance: %w", err))
	}
	return &a, nil
}

func (r *AttendancePostgres) GetByEmployeeAndDate(ctx context.Context, tenantID, employeeID uuid.UUID, date string) (*domain.Attendance, error) {
	query := `
		SELECT a.id, a.tenant_id, a.employee_id, COALESCE(e.name, '') as employee_name,
		       a.date::text, a.status, a.ot, a.advance, a.created_at, a.updated_at
		FROM attendance a
		LEFT JOIN tenant_employees e ON e.id = a.employee_id
		WHERE a.tenant_id = $1 AND a.employee_id = $2 AND a.date = $3::date
	`
	exec := GetExecutor(ctx, r.pool)
	var a domain.Attendance
	err := exec.QueryRow(ctx, query, tenantID, employeeID, date).Scan(
		&a.ID,
		&a.TenantID,
		&a.EmployeeID,
		&a.EmployeeName,
		&a.Date,
		&a.Status,
		&a.OT,
		&a.Advance,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found is not an error when checking existence
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get attendance by employee and date: %w", err))
	}
	return &a, nil
}

func (r *AttendancePostgres) Update(ctx context.Context, att *domain.Attendance) error {
	query := `
		UPDATE attendance
		SET status = $1, ot = $2, advance = $3, updated_at = NOW()
		WHERE tenant_id = $4 AND id = $5
		RETURNING updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query,
		att.Status,
		att.OT,
		att.Advance,
		att.TenantID,
		att.ID,
	).Scan(&att.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return appErrors.NewNotFound("attendance record not found within tenant")
		}
		return appErrors.NewDatabase(fmt.Errorf("failed to update attendance: %w", err))
	}
	return nil
}

func (r *AttendancePostgres) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	query := `DELETE FROM attendance WHERE tenant_id = $1 AND id = $2`
	exec := GetExecutor(ctx, r.pool)
	tag, err := exec.Exec(ctx, query, tenantID, id)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to delete attendance: %w", err))
	}
	if tag.RowsAffected() == 0 {
		return appErrors.NewNotFound("attendance record not found within tenant")
	}
	return nil
}

func (r *AttendancePostgres) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, date string, employeeID *uuid.UUID) ([]domain.Attendance, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	baseWhere := "WHERE a.tenant_id = $1"
	args := []any{tenantID}
	argIdx := 2

	if date != "" {
		baseWhere += fmt.Sprintf(" AND a.date = $%d::date", argIdx)
		args = append(args, date)
		argIdx++
	}

	if employeeID != nil && *employeeID != uuid.Nil {
		baseWhere += fmt.Sprintf(" AND a.employee_id = $%d", argIdx)
		args = append(args, *employeeID)
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM attendance a %s", baseWhere)
	exec := GetExecutor(ctx, r.pool)
	var total int64
	if err := exec.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to count attendance: %w", err))
	}

	listQuery := fmt.Sprintf(`
		SELECT a.id, a.tenant_id, a.employee_id, COALESCE(e.name, '') as employee_name,
		       a.date::text, a.status, a.ot, a.advance, a.created_at, a.updated_at
		FROM attendance a
		LEFT JOIN tenant_employees e ON e.id = a.employee_id
		%s
		ORDER BY a.date DESC, e.name ASC
		LIMIT $%d OFFSET $%d
	`, baseWhere, argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := exec.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to list attendance: %w", err))
	}
	defer rows.Close()

	list := make([]domain.Attendance, 0)
	for rows.Next() {
		var a domain.Attendance
		if err := rows.Scan(
			&a.ID,
			&a.TenantID,
			&a.EmployeeID,
			&a.EmployeeName,
			&a.Date,
			&a.Status,
			&a.OT,
			&a.Advance,
			&a.CreatedAt,
			&a.UpdatedAt,
		); err != nil {
			return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to scan attendance: %w", err))
		}
		list = append(list, a)
	}

	return list, total, nil
}

// ==========================================
// 2. EmployeeSalaryPostgres
// ==========================================

type EmployeeSalaryPostgres struct {
	pool *pgxpool.Pool
}

func NewEmployeeSalaryPostgres(pool *pgxpool.Pool) *EmployeeSalaryPostgres {
	return &EmployeeSalaryPostgres{pool: pool}
}

func (r *EmployeeSalaryPostgres) GetByEmployeeID(ctx context.Context, tenantID, employeeID uuid.UUID) (*domain.EmployeeSalary, error) {
	query := `
		SELECT COALESCE(s.id, gen_random_uuid()) as id,
		       e.tenant_id, e.id as employee_id, e.name as employee_name,
		       e.salary as salary_rate, e.ot_rate,
		       COALESCE(s.balance, 0.00) as balance,
		       COALESCE(s.created_at, NOW()) as created_at,
		       COALESCE(s.updated_at, NOW()) as updated_at
		FROM tenant_employees e
		LEFT JOIN employee_salary s ON s.tenant_id = e.tenant_id AND s.employee_id = e.id
		WHERE e.tenant_id = $1 AND e.id = $2
	`
	exec := GetExecutor(ctx, r.pool)
	var s domain.EmployeeSalary
	err := exec.QueryRow(ctx, query, tenantID, employeeID).Scan(
		&s.ID,
		&s.TenantID,
		&s.EmployeeID,
		&s.EmployeeName,
		&s.SalaryRate,
		&s.OTRate,
		&s.Balance,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("employee not found within tenant")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get employee salary: %w", err))
	}
	return &s, nil
}

func (r *EmployeeSalaryPostgres) UpsertBalance(ctx context.Context, salary *domain.EmployeeSalary) error {
	query := `
		INSERT INTO employee_salary (tenant_id, employee_id, balance)
		VALUES ($1, $2, $3)
		ON CONFLICT (tenant_id, employee_id)
		DO UPDATE SET balance = EXCLUDED.balance, updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query,
		salary.TenantID,
		salary.EmployeeID,
		salary.Balance,
	).Scan(&salary.ID, &salary.CreatedAt, &salary.UpdatedAt)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to upsert employee salary: %w", err))
	}
	return nil
}

func (r *EmployeeSalaryPostgres) AdjustBalance(ctx context.Context, tenantID, employeeID uuid.UUID, delta decimal.Decimal) error {
	query := `
		INSERT INTO employee_salary (tenant_id, employee_id, balance)
		VALUES ($1, $2, $3)
		ON CONFLICT (tenant_id, employee_id)
		DO UPDATE SET balance = employee_salary.balance + EXCLUDED.balance, updated_at = NOW()
	`
	exec := GetExecutor(ctx, r.pool)
	_, err := exec.Exec(ctx, query, tenantID, employeeID, delta)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to adjust employee salary balance: %w", err))
	}
	return nil
}

func (r *EmployeeSalaryPostgres) Delete(ctx context.Context, tenantID, employeeID uuid.UUID) error {
	query := `DELETE FROM employee_salary WHERE tenant_id = $1 AND employee_id = $2`
	exec := GetExecutor(ctx, r.pool)
	tag, err := exec.Exec(ctx, query, tenantID, employeeID)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to delete employee salary: %w", err))
	}
	if tag.RowsAffected() == 0 {
		return appErrors.NewNotFound("salary record not found")
	}
	return nil
}

func (r *EmployeeSalaryPostgres) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search string) ([]domain.EmployeeSalary, int64, error) {
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

	if search != "" {
		baseWhere += fmt.Sprintf(" AND e.name ILIKE $%d", argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tenant_employees e %s", baseWhere)
	exec := GetExecutor(ctx, r.pool)
	var total int64
	if err := exec.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to count salary records: %w", err))
	}

	listQuery := fmt.Sprintf(`
		SELECT COALESCE(s.id, gen_random_uuid()) as id,
		       e.tenant_id, e.id as employee_id, e.name as employee_name,
		       e.salary as salary_rate, e.ot_rate,
		       COALESCE(s.balance, 0.00) as balance,
		       COALESCE(s.created_at, e.created_at) as created_at,
		       COALESCE(s.updated_at, e.updated_at) as updated_at
		FROM tenant_employees e
		LEFT JOIN employee_salary s ON s.tenant_id = e.tenant_id AND s.employee_id = e.id
		%s
		ORDER BY e.name ASC
		LIMIT $%d OFFSET $%d
	`, baseWhere, argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := exec.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to list salaries: %w", err))
	}
	defer rows.Close()

	salaries := make([]domain.EmployeeSalary, 0)
	for rows.Next() {
		var s domain.EmployeeSalary
		if err := rows.Scan(
			&s.ID,
			&s.TenantID,
			&s.EmployeeID,
			&s.EmployeeName,
			&s.SalaryRate,
			&s.OTRate,
			&s.Balance,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to scan salary: %w", err))
		}
		salaries = append(salaries, s)
	}

	return salaries, total, nil
}

func (r *EmployeeSalaryPostgres) ListPending(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]domain.EmployeeSalary, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	countQuery := `
		SELECT COUNT(*)
		FROM tenant_employees e
		INNER JOIN employee_salary s ON s.tenant_id = e.tenant_id AND s.employee_id = e.id
		WHERE e.tenant_id = $1 AND s.balance > 0
	`
	exec := GetExecutor(ctx, r.pool)
	var total int64
	if err := exec.QueryRow(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to count pending salaries: %w", err))
	}

	listQuery := `
		SELECT s.id, e.tenant_id, e.id as employee_id, e.name as employee_name,
		       e.salary as salary_rate, e.ot_rate, s.balance, s.created_at, s.updated_at
		FROM tenant_employees e
		INNER JOIN employee_salary s ON s.tenant_id = e.tenant_id AND s.employee_id = e.id
		WHERE e.tenant_id = $1 AND s.balance > 0
		ORDER BY s.balance DESC, e.name ASC
		LIMIT $2 OFFSET $3
	`
	rows, err := exec.Query(ctx, listQuery, tenantID, pageSize, offset)
	if err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to list pending salaries: %w", err))
	}
	defer rows.Close()

	salaries := make([]domain.EmployeeSalary, 0)
	for rows.Next() {
		var s domain.EmployeeSalary
		if err := rows.Scan(
			&s.ID,
			&s.TenantID,
			&s.EmployeeID,
			&s.EmployeeName,
			&s.SalaryRate,
			&s.OTRate,
			&s.Balance,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to scan pending salary: %w", err))
		}
		salaries = append(salaries, s)
	}

	return salaries, total, nil
}

func (r *EmployeeSalaryPostgres) CreatePayment(ctx context.Context, payment *domain.EmployeeSalaryPayment) error {
	query := `
		INSERT INTO employee_salary_payment (tenant_id, employee_id, payment_method, bank_id, bank_name, amount, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query,
		payment.TenantID,
		payment.EmployeeID,
		payment.PaymentMethod,
		payment.BankID,
		payment.BankName,
		payment.Amount,
		payment.Note,
	).Scan(&payment.ID, &payment.CreatedAt, &payment.UpdatedAt)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to create salary payment: %w", err))
	}
	return nil
}

func (r *EmployeeSalaryPostgres) ListPayments(ctx context.Context, tenantID uuid.UUID, employeeID *uuid.UUID, page, pageSize int) ([]domain.EmployeeSalaryPayment, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	baseWhere := "WHERE p.tenant_id = $1"
	args := []any{tenantID}
	argIdx := 2

	if employeeID != nil && *employeeID != uuid.Nil {
		baseWhere += fmt.Sprintf(" AND p.employee_id = $%d", argIdx)
		args = append(args, *employeeID)
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM employee_salary_payment p %s", baseWhere)
	exec := GetExecutor(ctx, r.pool)
	var total int64
	if err := exec.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to count salary payments: %w", err))
	}

	listQuery := fmt.Sprintf(`
		SELECT p.id, p.tenant_id, p.employee_id, COALESCE(e.name, '') as employee_name,
		       p.payment_method, p.bank_id, p.bank_name, p.amount, p.note, p.created_at, p.updated_at
		FROM employee_salary_payment p
		LEFT JOIN tenant_employees e ON e.id = p.employee_id
		%s
		ORDER BY p.created_at DESC
		LIMIT $%d OFFSET $%d
	`, baseWhere, argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := exec.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to list salary payments: %w", err))
	}
	defer rows.Close()

	payments := make([]domain.EmployeeSalaryPayment, 0)
	for rows.Next() {
		var p domain.EmployeeSalaryPayment
		if err := rows.Scan(
			&p.ID,
			&p.TenantID,
			&p.EmployeeID,
			&p.EmployeeName,
			&p.PaymentMethod,
			&p.BankID,
			&p.BankName,
			&p.Amount,
			&p.Note,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to scan salary payment: %w", err))
		}
		payments = append(payments, p)
	}

	return payments, total, nil
}

// ==========================================
// 3. TenantDailyStatsPostgres
// ==========================================

type TenantDailyStatsPostgres struct {
	pool *pgxpool.Pool
}

func NewTenantDailyStatsPostgres(pool *pgxpool.Pool) *TenantDailyStatsPostgres {
	return &TenantDailyStatsPostgres{pool: pool}
}

func (r *TenantDailyStatsPostgres) GetByDate(ctx context.Context, tenantID uuid.UUID, date string) (*domain.TenantDailyStats, error) {
	query := `
		SELECT id, tenant_id, date::text, line_sale_amount, counter_sale_amount, total_sales,
		       purchase_amount, expense_amount, wages_amount, advance_amount,
		       attendance_present, attendance_absent, amount_received, amount_paid,
		       credit_sale, created_at, updated_at
		FROM tenant_daily_stats
		WHERE tenant_id = $1 AND date = $2::date
	`
	exec := GetExecutor(ctx, r.pool)
	var s domain.TenantDailyStats
	err := exec.QueryRow(ctx, query, tenantID, date).Scan(
		&s.ID,
		&s.TenantID,
		&s.Date,
		&s.LineSaleAmount,
		&s.CounterSaleAmount,
		&s.TotalSales,
		&s.PurchaseAmount,
		&s.ExpenseAmount,
		&s.WagesAmount,
		&s.AdvanceAmount,
		&s.AttendancePresent,
		&s.AttendanceAbsent,
		&s.AmountReceived,
		&s.AmountPaid,
		&s.CreditSale,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // If not computed yet, return nil without error
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get tenant daily stats: %w", err))
	}
	return &s, nil
}

func (r *TenantDailyStatsPostgres) Upsert(ctx context.Context, stats *domain.TenantDailyStats) error {
	query := `
		INSERT INTO tenant_daily_stats (
			tenant_id, date, line_sale_amount, counter_sale_amount, total_sales,
			purchase_amount, expense_amount, wages_amount, advance_amount,
			attendance_present, attendance_absent, amount_received, amount_paid, credit_sale
		) VALUES (
			$1, $2::date, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13, $14
		)
		ON CONFLICT (tenant_id, date)
		DO UPDATE SET
			line_sale_amount = EXCLUDED.line_sale_amount,
			counter_sale_amount = EXCLUDED.counter_sale_amount,
			total_sales = EXCLUDED.total_sales,
			purchase_amount = EXCLUDED.purchase_amount,
			expense_amount = EXCLUDED.expense_amount,
			wages_amount = EXCLUDED.wages_amount,
			advance_amount = EXCLUDED.advance_amount,
			attendance_present = EXCLUDED.attendance_present,
			attendance_absent = EXCLUDED.attendance_absent,
			amount_received = EXCLUDED.amount_received,
			amount_paid = EXCLUDED.amount_paid,
			credit_sale = EXCLUDED.credit_sale,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query,
		stats.TenantID,
		stats.Date,
		stats.LineSaleAmount,
		stats.CounterSaleAmount,
		stats.TotalSales,
		stats.PurchaseAmount,
		stats.ExpenseAmount,
		stats.WagesAmount,
		stats.AdvanceAmount,
		stats.AttendancePresent,
		stats.AttendanceAbsent,
		stats.AmountReceived,
		stats.AmountPaid,
		stats.CreditSale,
	).Scan(&stats.ID, &stats.CreatedAt, &stats.UpdatedAt)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to upsert tenant daily stats: %w", err))
	}
	return nil
}

func (r *TenantDailyStatsPostgres) ComputeAndSyncDailyStats(ctx context.Context, tenantID uuid.UUID, date string) (*domain.TenantDailyStats, error) {
	if date == "" {
		date = time.Now().UTC().Format("2006-01-02")
	}

	exec := GetExecutor(ctx, r.pool)

	// 1. Line Sales Aggregations
	var lineSaleAmount, lineCashIn, lineCredit decimal.Decimal
	lineQuery := `
		SELECT COALESCE(SUM(total_amount), 0), COALESCE(SUM(total_cash_in), 0), COALESCE(SUM(balance), 0)
		FROM line_sale
		WHERE tenant_id = $1 AND created_at::date = $2::date
	`
	if err := exec.QueryRow(ctx, lineQuery, tenantID, date).Scan(&lineSaleAmount, &lineCashIn, &lineCredit); err != nil {
		return nil, appErrors.NewDatabase(fmt.Errorf("failed aggregating line sales: %w", err))
	}

	// 2. Counter Sales Aggregations
	var counterSaleAmount, counterCash, counterAccount decimal.Decimal
	counterQuery := `
		SELECT COALESCE(SUM(total_amount), 0), COALESCE(SUM(cash), 0), COALESCE(SUM(account), 0)
		FROM counter_sale
		WHERE tenant_id = $1 AND created_at::date = $2::date
	`
	if err := exec.QueryRow(ctx, counterQuery, tenantID, date).Scan(&counterSaleAmount, &counterCash, &counterAccount); err != nil {
		return nil, appErrors.NewDatabase(fmt.Errorf("failed aggregating counter sales: %w", err))
	}

	// Also check counter sale payments (bank/upi portion)
	var counterPaymentsBank decimal.Decimal
	cpQuery := `
		SELECT COALESCE(SUM(p.amount), 0)
		FROM counter_sale_payments p
		WHERE p.tenant_id = $1 AND p.created_at::date = $2::date
	`
	_ = exec.QueryRow(ctx, cpQuery, tenantID, date).Scan(&counterPaymentsBank)

	// 3. Purchase Aggregations
	var purchaseAmount, purchasePaid decimal.Decimal
	purchQuery := `
		SELECT COALESCE(SUM(total_amount), 0), COALESCE(SUM(total_paid), 0)
		FROM tenant_purchase
		WHERE tenant_id = $1 AND created_at::date = $2::date
	`
	if err := exec.QueryRow(ctx, purchQuery, tenantID, date).Scan(&purchaseAmount, &purchasePaid); err != nil {
		return nil, appErrors.NewDatabase(fmt.Errorf("failed aggregating purchases: %w", err))
	}

	// 4. Expense Aggregations
	var expenseAmount decimal.Decimal
	expQuery := `
		SELECT COALESCE(SUM(total_amount), 0)
		FROM tenant_expense
		WHERE tenant_id = $1 AND created_at::date = $2::date
	`
	if err := exec.QueryRow(ctx, expQuery, tenantID, date).Scan(&expenseAmount); err != nil {
		return nil, appErrors.NewDatabase(fmt.Errorf("failed aggregating expenses: %w", err))
	}

	// 5. Wages Paid
	var wagesAmount decimal.Decimal
	wagesQuery := `
		SELECT COALESCE(SUM(amount), 0)
		FROM employee_salary_payment
		WHERE tenant_id = $1 AND created_at::date = $2::date
	`
	if err := exec.QueryRow(ctx, wagesQuery, tenantID, date).Scan(&wagesAmount); err != nil {
		return nil, appErrors.NewDatabase(fmt.Errorf("failed aggregating wages: %w", err))
	}

	// 6. Attendance & Advance
	var advanceAmount decimal.Decimal
	var attPresent, attAbsent int
	attQuery := `
		SELECT COALESCE(SUM(advance), 0),
		       COUNT(CASE WHEN status = 'present' THEN 1 END),
		       COUNT(CASE WHEN status = 'absent' THEN 1 END)
		FROM attendance
		WHERE tenant_id = $1 AND date = $2::date
	`
	if err := exec.QueryRow(ctx, attQuery, tenantID, date).Scan(&advanceAmount, &attPresent, &attAbsent); err != nil {
		return nil, appErrors.NewDatabase(fmt.Errorf("failed aggregating attendance: %w", err))
	}

	totalSales := lineSaleAmount.Add(counterSaleAmount)
	amountReceived := lineCashIn.Add(counterCash).Add(counterPaymentsBank)
	amountPaid := purchasePaid.Add(expenseAmount).Add(wagesAmount).Add(advanceAmount)
	creditSale := lineCredit.Add(counterAccount)

	stats := &domain.TenantDailyStats{
		TenantID:          tenantID,
		Date:              date,
		LineSaleAmount:    lineSaleAmount,
		CounterSaleAmount: counterSaleAmount,
		TotalSales:        totalSales,
		PurchaseAmount:    purchaseAmount,
		ExpenseAmount:     expenseAmount,
		WagesAmount:       wagesAmount,
		AdvanceAmount:     advanceAmount,
		AttendancePresent: attPresent,
		AttendanceAbsent:  attAbsent,
		AmountReceived:    amountReceived,
		AmountPaid:        amountPaid,
		CreditSale:        creditSale,
	}

	if err := r.Upsert(ctx, stats); err != nil {
		return nil, err
	}

	return stats, nil
}
