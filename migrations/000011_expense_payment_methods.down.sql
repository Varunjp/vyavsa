-- Migration 000011 down
DROP INDEX IF EXISTS idx_tenant_expense_payment_bank;
ALTER TABLE tenant_expense DROP COLUMN IF EXISTS payment_method;
