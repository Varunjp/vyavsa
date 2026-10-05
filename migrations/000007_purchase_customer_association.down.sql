-- 000007_purchase_customer_association.down.sql

DROP INDEX IF EXISTS idx_tenant_purchase_tenant_customer;
ALTER TABLE tenant_purchase DROP COLUMN IF EXISTS customer_name;
ALTER TABLE tenant_purchase DROP COLUMN IF EXISTS customer_id;
