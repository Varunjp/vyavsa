-- 000008_purchase_customer_payable.up.sql

-- 1. Add payment_status to tenant_purchase
ALTER TABLE tenant_purchase ADD COLUMN IF NOT EXISTS payment_status VARCHAR(50) NOT NULL DEFAULT 'UNPAID';

-- 2. Populate existing payment statuses based on total_paid and total_amount
UPDATE tenant_purchase
SET payment_status = CASE
    WHEN total_pending <= 0 OR total_paid >= total_amount THEN 'PAID'
    WHEN total_paid > 0 THEN 'PARTIALLY_PAID'
    ELSE 'UNPAID'
END;

-- 3. Index for querying purchase payment status
CREATE INDEX IF NOT EXISTS idx_tenant_purchase_tenant_status ON tenant_purchase(tenant_id, payment_status);

-- 4. Add customer_id and is_settlement to tenant_purchase_payment
ALTER TABLE tenant_purchase_payment ADD COLUMN IF NOT EXISTS customer_id UUID REFERENCES tenant_customer(id) ON DELETE SET NULL;
ALTER TABLE tenant_purchase_payment ADD COLUMN IF NOT EXISTS is_settlement BOOLEAN NOT NULL DEFAULT FALSE;

-- 5. Backfill customer_id in tenant_purchase_payment from tenant_purchase
UPDATE tenant_purchase_payment p
SET customer_id = tp.customer_id
FROM tenant_purchase tp
WHERE p.purchase_id = tp.id AND p.customer_id IS NULL AND tp.customer_id IS NOT NULL;

-- 6. Index for querying customer purchase payments
CREATE INDEX IF NOT EXISTS idx_tenant_purchase_payment_tenant_customer ON tenant_purchase_payment(tenant_id, customer_id);
