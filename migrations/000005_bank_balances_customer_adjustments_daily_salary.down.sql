-- 000005_bank_balances_customer_adjustments_daily_salary.down.sql

ALTER TABLE tenant_daily_stats DROP COLUMN IF EXISTS today_employee_salary;
ALTER TABLE counter_sale DROP COLUMN IF EXISTS collected_amount, DROP COLUMN IF EXISTS bank_id, DROP COLUMN IF EXISTS bank_amount;
ALTER TABLE line_sale DROP COLUMN IF EXISTS collected_amount, DROP COLUMN IF EXISTS bank_amount;
ALTER TABLE attendance DROP COLUMN IF EXISTS daily_salary;
DROP TABLE IF EXISTS customer_balance_adjustments;
DROP TABLE IF EXISTS tenant_bank_transactions;
ALTER TABLE tenant_bank DROP COLUMN IF EXISTS current_balance;
