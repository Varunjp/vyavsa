-- 000012_sync_attendance_daily_salary_and_indexes.up.sql

-- 1. Index attendance by tenant and employee for faster salary and balance aggregation
CREATE INDEX IF NOT EXISTS idx_attendance_tenant_emp ON attendance(tenant_id, employee_id);

-- 2. Backfill daily_salary for existing attendance records where it was 0 but status was present or half_day
UPDATE attendance a
SET daily_salary = CASE
    WHEN a.status = 'present' THEN e.salary
    WHEN a.status = 'half_day' THEN ROUND(e.salary / 2, 2)
    ELSE 0.00
END
FROM tenant_employees e
WHERE a.tenant_id = e.tenant_id
  AND a.employee_id = e.id
  AND a.daily_salary = 0.00
  AND a.status IN ('present', 'half_day');

-- 3. Synchronize employee_salary balances based on complete attendance, overtime, advances, and payments history
INSERT INTO employee_salary (tenant_id, employee_id, balance)
SELECT
    e.tenant_id,
    e.id,
    (COALESCE((SELECT SUM(a.daily_salary) FROM attendance a WHERE a.tenant_id = e.tenant_id AND a.employee_id = e.id), 0.00) +
     COALESCE((SELECT SUM(o.amount) FROM employee_overtime o WHERE o.tenant_id = e.tenant_id AND o.employee_id = e.id), 0.00) -
     COALESCE((SELECT SUM(adv.amount) FROM employee_advances adv WHERE adv.tenant_id = e.tenant_id AND adv.employee_id = e.id), 0.00) -
     COALESCE((SELECT SUM(p.amount) FROM employee_salary_payment p WHERE p.tenant_id = e.tenant_id AND p.employee_id = e.id), 0.00)
    )
FROM tenant_employees e
ON CONFLICT (tenant_id, employee_id)
DO UPDATE SET balance = EXCLUDED.balance, updated_at = NOW();
