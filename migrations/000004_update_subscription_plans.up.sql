-- 000004_update_subscription_plans.up.sql
-- Configure standard subscription plans:
-- 1. Default Free Trial (1 Month) with full feature access (Price: 0.00)
-- 2. Standard Monthly Plan (Price: 500.00)
-- 3. Archive any other obsolete plans

-- 1. Upsert / Update 1 Month Free Trial
UPDATE platform_plans
SET plan_name = '1 Month Free Trial',
    note = '1 month free trial with full access to all features',
    price = 0.00,
    status = 'active',
    updated_at = NOW()
WHERE id = '00000000-0000-0000-0001-000000000001' OR plan_name = 'Free Starter';

INSERT INTO platform_plans (id, plan_name, note, price, status)
VALUES (
    '00000000-0000-0000-0001-000000000001',
    '1 Month Free Trial',
    '1 month free trial with full access to all features',
    0.00,
    'active'
) ON CONFLICT (plan_name) DO UPDATE
SET note = EXCLUDED.note, price = EXCLUDED.price, status = 'active', updated_at = NOW();

-- 2. Upsert / Update Monthly Plan at ₹500.00
UPDATE platform_plans
SET plan_name = 'Monthly Plan',
    note = 'Full access to all business features at ₹500/month',
    price = 500.00,
    status = 'active',
    updated_at = NOW()
WHERE id = '00000000-0000-0000-0001-000000000002' OR plan_name = 'Standard Business';

INSERT INTO platform_plans (id, plan_name, note, price, status)
VALUES (
    '00000000-0000-0000-0001-000000000002',
    'Monthly Plan',
    'Full access to all business features at ₹500/month',
    500.00,
    'active'
) ON CONFLICT (plan_name) DO UPDATE
SET note = EXCLUDED.note, price = EXCLUDED.price, status = 'active', updated_at = NOW();

-- 3. Update existing subscriptions referencing ID 2 or old name
UPDATE platform_subscriptions
SET current_plan_name = 'Monthly Plan', updated_at = NOW()
WHERE current_plan_id = '00000000-0000-0000-0001-000000000002' OR current_plan_name = 'Standard Business';

-- 4. Archive any other plan so only the 2 required plans are active
UPDATE platform_plans
SET status = 'archived', updated_at = NOW()
WHERE id NOT IN (
    '00000000-0000-0000-0001-000000000001',
    '00000000-0000-0000-0001-000000000002'
) AND plan_name NOT IN ('1 Month Free Trial', 'Monthly Plan');
