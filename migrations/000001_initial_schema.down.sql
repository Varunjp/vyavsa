-- 000001_initial_schema.down.sql
-- Down migration for Vyavsa Small Business Bill Book SaaS

DROP TABLE IF EXISTS tenant_daily_stats CASCADE;
DROP TABLE IF EXISTS employee_salary_payment CASCADE;
DROP TABLE IF EXISTS employee_salary CASCADE;
DROP TABLE IF EXISTS attendance CASCADE;
DROP TABLE IF EXISTS tenant_financial_summary CASCADE;
DROP TABLE IF EXISTS tenant_expense_payment CASCADE;
DROP TABLE IF EXISTS tenant_expense CASCADE;
DROP TABLE IF EXISTS tenant_purchase_payment CASCADE;
DROP TABLE IF EXISTS tenant_purchase CASCADE;
DROP TABLE IF EXISTS counter_sale_payments CASCADE;
DROP TABLE IF EXISTS counter_sale CASCADE;
DROP TABLE IF EXISTS line_sale_payments CASCADE;
DROP TABLE IF EXISTS line_sale CASCADE;
DROP TABLE IF EXISTS tenant_customer CASCADE;
DROP TABLE IF EXISTS tenant_employees CASCADE;
DROP TABLE IF EXISTS tenant_bank CASCADE;
DROP TABLE IF EXISTS platform_plan_transactions CASCADE;
DROP TABLE IF EXISTS platform_subscriptions CASCADE;
DROP TABLE IF EXISTS tenant_user CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;
DROP TABLE IF EXISTS platform_plans CASCADE;
DROP TABLE IF EXISTS platform_admin CASCADE;
