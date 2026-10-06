-- Migration 000010: Essential performance indexes for financial write path and aggregations

-- 1. Index on counter_sale_payments(tenant_id, created_at DESC) to optimize date range aggregations in daily stats
CREATE INDEX IF NOT EXISTS idx_counter_sale_payments_tenant_created ON counter_sale_payments(tenant_id, created_at DESC);
