-- 000009_financial_history_overtime_advances.up.sql

-- 1. Attendance overtime amount support
ALTER TABLE attendance ADD COLUMN IF NOT EXISTS ot_amount NUMERIC(14,2) NOT NULL DEFAULT 0.00 CHECK (ot_amount >= 0);

-- Backfill ot_amount from historical ot hours and employee ot_rate
UPDATE attendance a
SET ot_amount = ROUND(a.ot * COALESCE(e.ot_rate, 0.00), 2)
FROM tenant_employees e
WHERE a.employee_id = e.id AND a.ot > 0 AND a.ot_amount = 0.00;

-- 2. Employee advances transaction ledger
CREATE TABLE IF NOT EXISTS employee_advances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES tenant_employees(id) ON DELETE CASCADE,
    expense_id UUID REFERENCES tenant_expense(id) ON DELETE SET NULL,
    amount NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    payment_method VARCHAR(50) NOT NULL DEFAULT 'cash',
    bank_id UUID REFERENCES tenant_bank(id) ON DELETE SET NULL,
    bank_name VARCHAR(255),
    reference_id VARCHAR(100),
    advance_date DATE NOT NULL DEFAULT CURRENT_DATE,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_employee_advances_tenant_emp ON employee_advances(tenant_id, employee_id, advance_date DESC);
CREATE INDEX IF NOT EXISTS idx_employee_advances_tenant_date ON employee_advances(tenant_id, advance_date DESC);

-- Backfill employee advances from existing attendance records
INSERT INTO employee_advances (tenant_id, employee_id, amount, advance_date, payment_method, notes, created_at, updated_at)
SELECT a.tenant_id, a.employee_id, a.advance, a.date, 'cash', 'Historical attendance advance', a.created_at, a.updated_at
FROM attendance a
WHERE a.advance > 0;

-- 3. Employee overtime transaction ledger
CREATE TABLE IF NOT EXISTS employee_overtime (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES tenant_employees(id) ON DELETE CASCADE,
    amount NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    overtime_date DATE NOT NULL DEFAULT CURRENT_DATE,
    reference_id VARCHAR(100),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_employee_overtime_tenant_emp ON employee_overtime(tenant_id, employee_id, overtime_date DESC);
CREATE INDEX IF NOT EXISTS idx_employee_overtime_tenant_date ON employee_overtime(tenant_id, overtime_date DESC);

-- Backfill employee overtime from existing attendance records
INSERT INTO employee_overtime (tenant_id, employee_id, amount, overtime_date, notes, created_at, updated_at)
SELECT a.tenant_id, a.employee_id, a.ot_amount, a.date, 'Historical attendance overtime', a.created_at, a.updated_at
FROM attendance a
WHERE a.ot_amount > 0;

-- 4. Expenses: Add category and employee association for employee advances
ALTER TABLE tenant_expense ADD COLUMN IF NOT EXISTS category VARCHAR(100) NOT NULL DEFAULT 'general';
ALTER TABLE tenant_expense ADD COLUMN IF NOT EXISTS employee_id UUID REFERENCES tenant_employees(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_tenant_expense_employee ON tenant_expense(tenant_id, employee_id);

-- 5. Salary Payment enhancements for payment history & reference
ALTER TABLE employee_salary_payment ADD COLUMN IF NOT EXISTS payment_date DATE NOT NULL DEFAULT CURRENT_DATE;
ALTER TABLE employee_salary_payment ADD COLUMN IF NOT EXISTS reference_id VARCHAR(100);
ALTER TABLE employee_salary_payment ADD COLUMN IF NOT EXISTS status VARCHAR(50) NOT NULL DEFAULT 'COMPLETED';
CREATE INDEX IF NOT EXISTS idx_employee_salary_payment_date ON employee_salary_payment(tenant_id, payment_date DESC);

-- 6. Purchase Payment enhancements for payment history & reference
ALTER TABLE tenant_purchase_payment ADD COLUMN IF NOT EXISTS payment_date DATE NOT NULL DEFAULT CURRENT_DATE;
ALTER TABLE tenant_purchase_payment ADD COLUMN IF NOT EXISTS reference_id VARCHAR(100);
ALTER TABLE tenant_purchase_payment ADD COLUMN IF NOT EXISTS status VARCHAR(50) NOT NULL DEFAULT 'COMPLETED';
CREATE INDEX IF NOT EXISTS idx_tenant_purchase_payment_date ON tenant_purchase_payment(tenant_id, payment_date DESC);

-- 7. Line Sale and Counter Sale Payments enhancements
ALTER TABLE line_sale_payments ADD COLUMN IF NOT EXISTS payment_date DATE NOT NULL DEFAULT CURRENT_DATE;
ALTER TABLE line_sale_payments ADD COLUMN IF NOT EXISTS reference_id VARCHAR(100);
ALTER TABLE line_sale_payments ADD COLUMN IF NOT EXISTS status VARCHAR(50) NOT NULL DEFAULT 'COMPLETED';
CREATE INDEX IF NOT EXISTS idx_line_sale_payments_date ON line_sale_payments(tenant_id, payment_date DESC);

ALTER TABLE counter_sale_payments ADD COLUMN IF NOT EXISTS payment_date DATE NOT NULL DEFAULT CURRENT_DATE;
ALTER TABLE counter_sale_payments ADD COLUMN IF NOT EXISTS reference_id VARCHAR(100);
ALTER TABLE counter_sale_payments ADD COLUMN IF NOT EXISTS status VARCHAR(50) NOT NULL DEFAULT 'COMPLETED';
CREATE INDEX IF NOT EXISTS idx_counter_sale_payments_date ON counter_sale_payments(tenant_id, payment_date DESC);
