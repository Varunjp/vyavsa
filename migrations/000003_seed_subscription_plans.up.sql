-- 000003_seed_subscription_plans.up.sql
-- Seed standard subscription plans and initialize state for demo tenant

-- 1. Standard Subscription Plans
INSERT INTO platform_plans (id, plan_name, note, price, status)
VALUES 
    (
        '00000000-0000-0000-0001-000000000001',
        'Free Starter',
        'Essential billing and accounting for small retail shops',
        0.00,
        'active'
    ),
    (
        '00000000-0000-0000-0001-000000000002',
        'Standard Business',
        'Comprehensive features, line sales routes, and inventory tracking',
        499.00,
        'active'
    ),
    (
        '00000000-0000-0000-0001-000000000003',
        'Enterprise Premium',
        'Unlimited routes, multi-user access, advanced reports, and dedicated support',
        1499.00,
        'active'
    )
ON CONFLICT (plan_name) DO NOTHING;

-- 2. Initialize Financial Summary for Demo Store
INSERT INTO tenant_financial_summary (id, tenant_id, cash_balance, bank_balance, total_receivable, total_payable)
VALUES (
    '33333333-3333-3333-3333-333333333333',
    '11111111-1111-1111-1111-111111111111',
    0.00,
    0.00,
    0.00,
    0.00
) ON CONFLICT (tenant_id) DO NOTHING;

-- 3. Initialize Subscription for Demo Store
INSERT INTO platform_subscriptions (id, tenant_id, current_plan_id, current_plan_name, status, end_date)
VALUES (
    '44444444-4444-4444-4444-444444444444',
    '11111111-1111-1111-1111-111111111111',
    '00000000-0000-0000-0001-000000000002',
    'Standard Business',
    'active',
    NOW() + INTERVAL '30 days'
) ON CONFLICT (tenant_id) DO NOTHING;
