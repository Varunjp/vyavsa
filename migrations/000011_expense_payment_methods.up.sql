-- Migration 000011: Support flexible expense payment methods (Cash, Bank, Cash + Bank with multi-bank splits)

-- 1. Add payment_method column to tenant_expense
ALTER TABLE tenant_expense ADD COLUMN IF NOT EXISTS payment_method VARCHAR(50) NOT NULL DEFAULT 'cash';

-- 2. Performance and lookup index for bank payments in tenant_expense_payment
CREATE INDEX IF NOT EXISTS idx_tenant_expense_payment_bank ON tenant_expense_payment(tenant_id, bank_id);

-- 3. Backfill existing expenses: ensure every existing expense has a corresponding payment record in tenant_expense_payment
INSERT INTO tenant_expense_payment (tenant_id, expense_id, payment_method, amount, created_at, updated_at)
SELECT e.tenant_id, e.id, 'cash', e.total_amount, e.created_at, e.updated_at
FROM tenant_expense e
LEFT JOIN tenant_expense_payment p ON p.expense_id = e.id
WHERE p.id IS NULL AND e.total_amount > 0;

-- 4. Sync payment_method on tenant_expense based on existing payment records
UPDATE tenant_expense e
SET payment_method = CASE
    WHEN EXISTS (SELECT 1 FROM tenant_expense_payment p WHERE p.expense_id = e.id AND p.payment_method = 'cash')
     AND EXISTS (SELECT 1 FROM tenant_expense_payment p WHERE p.expense_id = e.id AND p.payment_method = 'bank') THEN 'cash_bank'
    WHEN EXISTS (SELECT 1 FROM tenant_expense_payment p WHERE p.expense_id = e.id AND p.payment_method = 'bank') THEN 'bank'
    ELSE 'cash'
END
WHERE EXISTS (SELECT 1 FROM tenant_expense_payment p WHERE p.expense_id = e.id);
