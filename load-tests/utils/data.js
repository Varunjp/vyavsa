/**
 * Test data provider and synthetic generator for Vyavsa load tests.
 */

import { randomInt, randomString, randomPhone, randomChoice } from './helpers.js';
import { TEST_EMAIL, TEST_PASSWORD, TEST_TENANT_ID, ACCESS_TOKEN } from '../config.js';

let dataset = null;

try {
  // Attempt to load synthetic pre-seeded multi-tenant dataset
  const raw = open('../data/test-users.json');
  if (raw && raw.length > 0) {
    dataset = JSON.parse(raw);
  }
} catch (e) {
  // test-users.json not present; fallback to default configured tenant credentials
}

/**
 * Returns available tenants list or a single fallback tenant.
 */
export function getTenants() {
  if (dataset && dataset.tenants && dataset.tenants.length > 0) {
    return dataset.tenants;
  }

  // Fallback single-tenant configuration
  return [
    {
      tenant_id: TEST_TENANT_ID,
      name: 'Default Test Store',
      email: TEST_EMAIL,
      admin: {
        id: '22222222-2222-2222-2222-222222222222',
        email: TEST_EMAIL,
        password: TEST_PASSWORD,
        role: 'admin',
        access_token: ACCESS_TOKEN,
      },
      users: [],
      sample_customer_id: null,
      sample_bank_id: null,
      sample_employee_id: null,
    },
  ];
}

/**
 * Returns a specific tenant for a given VU index to distribute multi-tenant load.
 */
export function getTenantForVU(vuIndex) {
  const tenants = getTenants();
  return tenants[vuIndex % tenants.length];
}

/**
 * Returns a user credential (admin or staff) for login testing.
 */
export function getUserForVU(vuIndex) {
  const tenants = getTenants();
  const tenant = tenants[vuIndex % tenants.length];
  if (tenant.users && tenant.users.length > 0 && vuIndex % 2 === 1) {
    return {
      email: tenant.users[0].email,
      password: tenant.users[0].password,
      tenantId: tenant.tenant_id,
      role: tenant.users[0].role,
      accessToken: tenant.users[0].access_token,
    };
  }
  return {
    email: tenant.admin.email,
    password: tenant.admin.password,
    tenantId: tenant.tenant_id,
    role: tenant.admin.role,
    accessToken: tenant.admin.access_token,
  };
}

/**
 * Generates valid payload for POST /api/v1/tenant/customers
 */
export function makeCustomerPayload() {
  return {
    customer_name: `Customer ${randomString(6)}`,
    phone: randomPhone(),
    opening_balance: `${randomInt(100, 5000)}.00`,
    status: 'active',
  };
}

/**
 * Generates valid payload for POST /api/v1/tenant/line-sales
 */
export function makeLineSalePayload(customerId) {
  const amount = randomInt(500, 5000);
  const cash = randomInt(100, amount);
  return {
    customer_id: customerId,
    route: `Route-${randomInt(1, 10)}`,
    salesman: `Salesman-${randomInt(1, 5)}`,
    note: `Load test line sale ${randomString(4)}`,
    total_amount: `${amount}.00`,
    total_cash_in: `${cash}.00`,
  };
}

/**
 * Generates valid payload for POST /api/v1/tenant/counter-sales
 */
export function makeCounterSalePayload() {
  const amount = randomInt(50, 1500);
  return {
    item: `Product-${randomChoice(['Biscuits', 'Wheat', 'Rice', 'Oil', 'Soap', 'Tea'])}-${randomInt(1, 100)}`,
    price: `${amount}.00`,
    total_amount: `${amount}.00`,
    payment_method: 'cash',
    cash: `${amount}.00`,
  };
}

/**
 * Generates valid payload for POST /api/v1/tenant/expenses
 */
export function makeExpensePayload() {
  const amount = randomInt(50, 500);
  return {
    item: `Store Expense ${randomString(5)}`,
    category: randomChoice(['tea_coffee', 'cleaning', 'utilities', 'stationery', 'transport']),
    total_amount: `${amount}.00`,
    payment_method: 'cash',
  };
}

/**
 * Generates valid payload for POST /api/v1/tenant/attendance
 */
export function makeAttendancePayload(employeeId) {
  const today = new Date().toISOString().split('T')[0];
  return {
    employee_id: employeeId,
    date: today,
    status: randomChoice(['present', 'present', 'present', 'half_day']),
  };
}
