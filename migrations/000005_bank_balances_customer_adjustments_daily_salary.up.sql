-- 000005_bank_balances_customer_adjustments_daily_salary.up.sql

-- 1. Add current_balance to tenant_bank
ALTER TABLE tenant_bank ADD COLUMN IF NOT EXISTS current_balance NUMERIC(14,2) NOT NULL DEFAULT 0.00;

-- 2. Create tenant_bank_transactions table for individual bank ledger tracking
CREATE TABLE IF NOT EXISTS tenant_bank_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    bank_id UUID NOT NULL REFERENCES tenant_bank(id) ON DELETE CASCADE,
    amount NUMERIC(14,2) NOT NULL,
    transaction_type VARCHAR(50) NOT NULL, -- 'credit', 'debit'
    reason VARCHAR(255) NOT NULL,
    sale_type VARCHAR(50), -- 'line_sale', 'counter_sale', or NULL
    sale_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_tenant_bank_transactions_bank ON tenant_bank_transactions(tenant_id, bank_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tenant_bank_transactions_sale ON tenant_bank_transactions(tenant_id, sale_id);

-- 3. Create customer_balance_adjustments table for manual customer balance modifications
CREATE TABLE IF NOT EXISTS customer_balance_adjustments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL REFERENCES tenant_customer(id) ON DELETE CASCADE,
    previous_balance NUMERIC(14,2) NOT NULL,
    new_balance NUMERIC(14,2) NOT NULL,
    adjustment_amount NUMERIC(14,2) NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_customer_adjustments_tenant_cust ON customer_balance_adjustments(tenant_id, customer_id, created_at DESC);

-- 4. Add daily_salary to attendance table
ALTER TABLE attendance ADD COLUMN IF NOT EXISTS daily_salary NUMERIC(14,2) NOT NULL DEFAULT 0.00;

-- 5. Add bank_amount and collected_amount to line_sale table
ALTER TABLE line_sale ADD COLUMN IF NOT EXISTS bank_amount NUMERIC(14,2) NOT NULL DEFAULT 0.00;
ALTER TABLE line_sale ADD COLUMN IF NOT EXISTS collected_amount NUMERIC(14,2) NOT NULL DEFAULT 0.00;

-- 6. Add bank_amount, bank_id, and collected_amount to counter_sale table
ALTER TABLE counter_sale ADD COLUMN IF NOT EXISTS bank_amount NUMERIC(14,2) NOT NULL DEFAULT 0.00;
ALTER TABLE counter_sale ADD COLUMN IF NOT EXISTS bank_id UUID REFERENCES tenant_bank(id) ON DELETE SET NULL;
ALTER TABLE counter_sale ADD COLUMN IF NOT EXISTS collected_amount NUMERIC(14,2) NOT NULL DEFAULT 0.00;

-- 7. Add today_employee_salary to tenant_daily_stats table
ALTER TABLE tenant_daily_stats ADD COLUMN IF NOT EXISTS today_employee_salary NUMERIC(14,2) NOT NULL DEFAULT 0.00;
