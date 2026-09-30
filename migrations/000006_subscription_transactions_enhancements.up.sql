-- 000006_subscription_transactions_enhancements.up.sql

-- 1. Add failure_reason and plan_name to platform_plan_transactions
ALTER TABLE platform_plan_transactions ADD COLUMN IF NOT EXISTS failure_reason TEXT;
ALTER TABLE platform_plan_transactions ADD COLUMN IF NOT EXISTS plan_name VARCHAR(100);

-- 2. Add start_date to platform_subscriptions
ALTER TABLE platform_subscriptions ADD COLUMN IF NOT EXISTS start_date TIMESTAMPTZ DEFAULT NOW();
UPDATE platform_subscriptions SET start_date = created_at WHERE start_date IS NULL;
