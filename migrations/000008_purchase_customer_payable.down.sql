-- 000008_purchase_customer_payable.down.sql

DROP INDEX IF EXISTS idx_tenant_purchase_payment_tenant_customer;
ALTER TABLE tenant_purchase_payment DROP COLUMN IF EXISTS is_settlement;
ALTER TABLE tenant_purchase_payment DROP COLUMN IF EXISTS customer_id;

DROP INDEX IF EXISTS idx_tenant_purchase_tenant_status;
ALTER TABLE tenant_purchase DROP COLUMN IF EXISTS payment_status;
