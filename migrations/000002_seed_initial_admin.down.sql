-- 000002_seed_initial_admin.down.sql
-- Rollback seeded platform admin and demo tenant

DELETE FROM tenant_user WHERE id = '22222222-2222-2222-2222-222222222222';
DELETE FROM tenants WHERE id = '11111111-1111-1111-1111-111111111111';
DELETE FROM platform_admin WHERE id = '00000000-0000-0000-0000-000000000001';
