-- 000002_seed_initial_admin.up.sql
-- Seed default Platform Administrator and Demo Tenant

-- 1. Default Platform Administrator (Password: Admin@12345)
INSERT INTO platform_admin (id, username, email, phone, status, password_hash)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'platform_admin',
    'admin@vyavsa.com',
    '+919876543210',
    'active',
    '$2b$12$4KHlciG0A9OWuLDO1W7Lf.UvJ65fKYWmCsitEOkmQaq1Gcsi06Bmi'
) ON CONFLICT (email) DO NOTHING;

-- 2. Default Demo Business Tenant
INSERT INTO tenants (id, name, email, phone, status)
VALUES (
    '11111111-1111-1111-1111-111111111111',
    'Demo Store',
    'contact@demostore.com',
    '+919876543211',
    'active'
) ON CONFLICT (email) DO NOTHING;

-- 3. Default Tenant Administrator User (Password: Store@12345)
INSERT INTO tenant_user (id, tenant_id, name, role, email, password_hash, status)
VALUES (
    '22222222-2222-2222-2222-222222222222',
    '11111111-1111-1111-1111-111111111111',
    'Demo Store Admin',
    'admin',
    'admin@demostore.com',
    '$2b$12$OW4euk/eQrmuwjso8hGa7eS8D8diiIx/jyRvYJKGYqvH4/fM2364.',
    'active'
) ON CONFLICT (tenant_id, email) DO NOTHING;
