-- 000009_financial_history_overtime_advances.down.sql

DROP INDEX IF EXISTS idx_counter_sale_payments_date;
ALTER TABLE counter_sale_payments DROP COLUMN IF EXISTS status;
ALTER TABLE counter_sale_payments DROP COLUMN IF EXISTS reference_id;
ALTER TABLE counter_sale_payments DROP COLUMN IF EXISTS payment_date;

DROP INDEX IF EXISTS idx_line_sale_payments_date;
ALTER TABLE line_sale_payments DROP COLUMN IF EXISTS status;
ALTER TABLE line_sale_payments DROP COLUMN IF EXISTS reference_id;
ALTER TABLE line_sale_payments DROP COLUMN IF EXISTS payment_date;

DROP INDEX IF EXISTS idx_tenant_purchase_payment_date;
ALTER TABLE tenant_purchase_payment DROP COLUMN IF EXISTS status;
ALTER TABLE tenant_purchase_payment DROP COLUMN IF EXISTS reference_id;
ALTER TABLE tenant_purchase_payment DROP COLUMN IF EXISTS payment_date;

DROP INDEX IF EXISTS idx_employee_salary_payment_date;
ALTER TABLE employee_salary_payment DROP COLUMN IF EXISTS status;
ALTER TABLE employee_salary_payment DROP COLUMN IF EXISTS reference_id;
ALTER TABLE employee_salary_payment DROP COLUMN IF EXISTS payment_date;

DROP INDEX IF EXISTS idx_tenant_expense_employee;
ALTER TABLE tenant_expense DROP COLUMN IF EXISTS employee_id;
ALTER TABLE tenant_expense DROP COLUMN IF EXISTS category;

DROP TABLE IF EXISTS employee_overtime;
DROP TABLE IF EXISTS employee_advances;

ALTER TABLE attendance DROP COLUMN IF EXISTS ot_amount;
