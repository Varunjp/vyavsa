-- 000004_update_subscription_plans.down.sql
UPDATE platform_plans
SET plan_name = 'Free Starter',
    note = 'Essential billing and accounting for small retail shops',
    price = 0.00,
    status = 'active',
    updated_at = NOW()
WHERE id = '00000000-0000-0000-0001-000000000001';

UPDATE platform_plans
SET plan_name = 'Standard Business',
    note = 'Comprehensive features, line sales routes, and inventory tracking',
    price = 499.00,
    status = 'active',
    updated_at = NOW()
WHERE id = '00000000-0000-0000-0001-000000000002';

UPDATE platform_plans
SET status = 'active',
    updated_at = NOW()
WHERE id = '00000000-0000-0000-0001-000000000003';

UPDATE platform_subscriptions
SET current_plan_name = 'Standard Business', updated_at = NOW()
WHERE current_plan_id = '00000000-0000-0000-0001-000000000002';
