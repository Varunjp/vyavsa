-- 000006_subscription_transactions_enhancements.down.sql

ALTER TABLE platform_subscriptions DROP COLUMN IF EXISTS start_date;
ALTER TABLE platform_plan_transactions DROP COLUMN IF EXISTS plan_name;
ALTER TABLE platform_plan_transactions DROP COLUMN IF EXISTS failure_reason;
