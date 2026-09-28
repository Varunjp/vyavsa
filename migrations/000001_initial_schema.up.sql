-- 000001_initial_schema.up.sql
-- Initial schema for Vyavsa Small Business Bill Book SaaS

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 1. Platform Administrators
CREATE TABLE IF NOT EXISTS platform_admin (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(100) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    phone VARCHAR(30),
    status VARCHAR(30) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'suspended')),
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Platform Subscription Plans
CREATE TABLE IF NOT EXISTS platform_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_name VARCHAR(100) NOT NULL UNIQUE,
    note TEXT,
    price NUMERIC(14,2) NOT NULL DEFAULT 0.00 CHECK (price >= 0),
    status VARCHAR(30) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'archived')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Tenants (Businesses)
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    phone VARCHAR(30),
    status VARCHAR(30) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'suspended')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 4. Tenant Users (Admins & Regular Staff)
CREATE TABLE IF NOT EXISTS tenant_user (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(30) NOT NULL CHECK (role IN ('admin', 'user')),
    email VARCHAR(255) NOT NULL,
    password_hash TEXT NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'suspended')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_user_tenant_email UNIQUE (tenant_id, email)
);
CREATE INDEX IF NOT EXISTS idx_tenant_user_tenant_id ON tenant_user(tenant_id);

-- 5. Platform Subscriptions
CREATE TABLE IF NOT EXISTS platform_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    current_plan_id UUID NOT NULL REFERENCES platform_plans(id) ON DELETE RESTRICT,
    current_plan_name VARCHAR(100) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'past_due', 'cancelled', 'expired')),
    end_date TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_platform_subscriptions_tenant UNIQUE (tenant_id)
);
CREATE INDEX IF NOT EXISTS idx_subscription_tenant_id ON platform_subscriptions(tenant_id);

-- 6. Platform Plan Transactions
CREATE TABLE IF NOT EXISTS platform_plan_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    transaction_id VARCHAR(255) NOT NULL UNIQUE,
    payment_method VARCHAR(50) NOT NULL,
    plan_id UUID NOT NULL REFERENCES platform_plans(id) ON DELETE RESTRICT,
    amount NUMERIC(14,2) NOT NULL CHECK (amount >= 0),
    status VARCHAR(30) NOT NULL CHECK (status IN ('pending', 'completed', 'failed', 'refunded')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_platform_plan_transactions_tenant_created ON platform_plan_transactions(tenant_id, created_at DESC);

-- 7. Tenant Bank Accounts
CREATE TABLE IF NOT EXISTS tenant_bank (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    bank_name VARCHAR(255) NOT NULL,
    account_number VARCHAR(100),
    ifsc_or_routing VARCHAR(50),
    status VARCHAR(30) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_tenant_bank_tenant_status ON tenant_bank(tenant_id, status);

-- 8. Tenant Employees
CREATE TABLE IF NOT EXISTS tenant_employees (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(30),
    salary NUMERIC(14,2) NOT NULL DEFAULT 0.00 CHECK (salary >= 0),
    ot_rate NUMERIC(14,2) NOT NULL DEFAULT 0.00 CHECK (ot_rate >= 0),
    status VARCHAR(30) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'terminated')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_tenant_employees_tenant_status ON tenant_employees(tenant_id, status);

-- 9. Tenant Customers
CREATE TABLE IF NOT EXISTS tenant_customer (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_name VARCHAR(255) NOT NULL,
    phone VARCHAR(30),
    opening_balance NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    current_balance NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    status VARCHAR(30) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_tenant_customer_tenant_name ON tenant_customer(tenant_id, customer_name);
CREATE INDEX IF NOT EXISTS idx_tenant_customer_tenant_phone ON tenant_customer(tenant_id, phone);

-- 10. Line Sales (Field Route Sales)
CREATE TABLE IF NOT EXISTS line_sale (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL REFERENCES tenant_customer(id) ON DELETE RESTRICT,
    customer_name VARCHAR(255) NOT NULL,
    route VARCHAR(255),
    salesman VARCHAR(255),
    note TEXT,
    total_amount NUMERIC(14,2) NOT NULL DEFAULT 0.00 CHECK (total_amount >= 0),
    total_cash_in NUMERIC(14,2) NOT NULL DEFAULT 0.00 CHECK (total_cash_in >= 0),
    balance NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_line_sale_tenant_created ON line_sale(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_line_sale_tenant_customer ON line_sale(tenant_id, customer_id);

-- 11. Line Sale Payments
CREATE TABLE IF NOT EXISTS line_sale_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    line_sale_id UUID NOT NULL REFERENCES line_sale(id) ON DELETE CASCADE,
    payment_method VARCHAR(50) NOT NULL CHECK (payment_method IN ('cash', 'bank', 'cheque', 'upi', 'other')),
    bank_id UUID REFERENCES tenant_bank(id) ON DELETE SET NULL,
    bank_name VARCHAR(255),
    amount NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_line_sale_payments_tenant_sale ON line_sale_payments(tenant_id, line_sale_id);
CREATE INDEX IF NOT EXISTS idx_line_sale_payments_tenant_created ON line_sale_payments(tenant_id, created_at DESC);

-- 12. Counter Sales (Point of Sale)
CREATE TABLE IF NOT EXISTS counter_sale (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    item VARCHAR(255) NOT NULL,
    price NUMERIC(14,2) NOT NULL DEFAULT 0.00 CHECK (price >= 0),
    total_amount NUMERIC(14,2) NOT NULL DEFAULT 0.00 CHECK (total_amount >= 0),
    payment_method VARCHAR(50) NOT NULL,
    cash NUMERIC(14,2) NOT NULL DEFAULT 0.00 CHECK (cash >= 0),
    account NUMERIC(14,2) NOT NULL DEFAULT 0.00 CHECK (account >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_counter_sale_tenant_created ON counter_sale(tenant_id, created_at DESC);

-- 13. Counter Sale Payments
CREATE TABLE IF NOT EXISTS counter_sale_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    counter_sale_id UUID NOT NULL REFERENCES counter_sale(id) ON DELETE CASCADE,
    payment_method VARCHAR(50) NOT NULL,
    bank_id UUID REFERENCES tenant_bank(id) ON DELETE SET NULL,
    bank_name VARCHAR(255),
    amount NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_counter_sale_payments_tenant_sale ON counter_sale_payments(tenant_id, counter_sale_id);

-- 14. Purchases
CREATE TABLE IF NOT EXISTS tenant_purchase (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    item VARCHAR(255) NOT NULL,
    quantity INT NOT NULL DEFAULT 1 CHECK (quantity > 0),
    total_amount NUMERIC(14,2) NOT NULL DEFAULT 0.00 CHECK (total_amount >= 0),
    total_paid NUMERIC(14,2) NOT NULL DEFAULT 0.00 CHECK (total_paid >= 0),
    total_pending NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_tenant_purchase_tenant_created ON tenant_purchase(tenant_id, created_at DESC);

-- 15. Purchase Payments
CREATE TABLE IF NOT EXISTS tenant_purchase_payment (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    purchase_id UUID NOT NULL REFERENCES tenant_purchase(id) ON DELETE CASCADE,
    payment_method VARCHAR(50) NOT NULL,
    bank_id UUID REFERENCES tenant_bank(id) ON DELETE SET NULL,
    bank_name VARCHAR(255),
    amount NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_tenant_purchase_payment_tenant_purchase ON tenant_purchase_payment(tenant_id, purchase_id);

-- 16. Expenses
CREATE TABLE IF NOT EXISTS tenant_expense (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    item VARCHAR(255) NOT NULL,
    total_amount NUMERIC(14,2) NOT NULL DEFAULT 0.00 CHECK (total_amount >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_tenant_expense_tenant_created ON tenant_expense(tenant_id, created_at DESC);

-- 17. Expense Payments
CREATE TABLE IF NOT EXISTS tenant_expense_payment (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    expense_id UUID NOT NULL REFERENCES tenant_expense(id) ON DELETE CASCADE,
    payment_method VARCHAR(50) NOT NULL,
    bank_id UUID REFERENCES tenant_bank(id) ON DELETE SET NULL,
    bank_name VARCHAR(255),
    amount NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_tenant_expense_payment_tenant_expense ON tenant_expense_payment(tenant_id, expense_id);

-- 18. Financial Summary (Authoritative Tenant Balances)
CREATE TABLE IF NOT EXISTS tenant_financial_summary (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    cash_balance NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    bank_balance NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    total_receivable NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    total_payable NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_financial_summary_tenant UNIQUE (tenant_id)
);
CREATE INDEX IF NOT EXISTS idx_financial_summary_tenant_id ON tenant_financial_summary(tenant_id);

-- 19. Employee Attendance
CREATE TABLE IF NOT EXISTS attendance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES tenant_employees(id) ON DELETE CASCADE,
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    status VARCHAR(30) NOT NULL CHECK (status IN ('present', 'absent', 'half_day', 'leave')),
    ot NUMERIC(14,2) NOT NULL DEFAULT 0.00 CHECK (ot >= 0),
    advance NUMERIC(14,2) NOT NULL DEFAULT 0.00 CHECK (advance >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_attendance_tenant_emp_date UNIQUE (tenant_id, employee_id, date)
);
CREATE INDEX IF NOT EXISTS idx_attendance_tenant_date ON attendance(tenant_id, date DESC);

-- 20. Employee Salary Balances
CREATE TABLE IF NOT EXISTS employee_salary (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES tenant_employees(id) ON DELETE CASCADE,
    balance NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_employee_salary_tenant_emp UNIQUE (tenant_id, employee_id)
);
CREATE INDEX IF NOT EXISTS idx_employee_salary_tenant_emp ON employee_salary(tenant_id, employee_id);

-- 21. Employee Salary Payments
CREATE TABLE IF NOT EXISTS employee_salary_payment (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES tenant_employees(id) ON DELETE CASCADE,
    payment_method VARCHAR(50) NOT NULL,
    bank_id UUID REFERENCES tenant_bank(id) ON DELETE SET NULL,
    bank_name VARCHAR(255),
    amount NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_employee_salary_payment_tenant_emp ON employee_salary_payment(tenant_id, employee_id);
CREATE INDEX IF NOT EXISTS idx_employee_salary_payment_tenant_created ON employee_salary_payment(tenant_id, created_at DESC);

-- 22. Tenant Daily Statistics
CREATE TABLE IF NOT EXISTS tenant_daily_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    line_sale_amount NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    counter_sale_amount NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    total_sales NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    purchase_amount NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    expense_amount NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    wages_amount NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    advance_amount NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    attendance_present INT NOT NULL DEFAULT 0,
    attendance_absent INT NOT NULL DEFAULT 0,
    amount_received NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    amount_paid NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    credit_sale NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_daily_stats_tenant_date UNIQUE (tenant_id, date)
);
CREATE INDEX IF NOT EXISTS idx_tenant_daily_stats_tenant_date ON tenant_daily_stats(tenant_id, date DESC);
