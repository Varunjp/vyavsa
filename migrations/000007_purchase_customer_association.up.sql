-- 000007_purchase_customer_association.up.sql

-- 1. Add customer_id and customer_name to tenant_purchase
ALTER TABLE tenant_purchase ADD COLUMN IF NOT EXISTS customer_id UUID REFERENCES tenant_customer(id) ON DELETE SET NULL;
ALTER TABLE tenant_purchase ADD COLUMN IF NOT EXISTS customer_name VARCHAR(255);

-- 2. Create index for tenant customer purchases
CREATE INDEX IF NOT EXISTS idx_tenant_purchase_tenant_customer ON tenant_purchase(tenant_id, customer_id);
