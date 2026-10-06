/**
 * Authenticated Multi-Tenant Load Test Scenario
 * Evaluates tenant-scoped read APIs, dashboard calculations, write mutations, and tenant data isolation.
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, assertSafety, THRESHOLDS } from '../config.js';
import { buildHeaders, parseJSON } from '../utils/helpers.js';
import { getTenantForVU, makeCustomerPayload, makeCounterSalePayload, makeExpensePayload } from '../utils/data.js';
import { getValidTokenForVU, invalidateVUToken } from '../utils/auth.js';

assertSafety();

export const options = {
  vus: parseInt(__ENV.VUS || '20', 10),
  duration: __ENV.DURATION || '60s',
  thresholds: {
    ...THRESHOLDS.standard,
    'http_req_duration{operation:read}': ['p(95)<300', 'p(99)<600'],
    'http_req_duration{operation:dashboard}': ['p(95)<400', 'p(99)<800'],
    'http_req_duration{operation:write}': ['p(95)<500', 'p(99)<1000'],
  },
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)'],
};

export default function () {
  const tenant = getTenantForVU(__VU);
  const token = getValidTokenForVU(__VU, BASE_URL, tenant.admin);

  if (!token) {
    check(null, { 'token acquired for tenant': () => false });
    sleep(1);
    return;
  }

  const headers = buildHeaders(token);

  // 1. Read: Tenant Profile (Also validates multi-tenant isolation)
  const profileRes = http.get(`${BASE_URL}/api/v1/tenant/profile`, {
    headers,
    tags: { name: 'GET /api/v1/tenant/profile', operation: 'read', tenant_id: tenant.tenant_id },
  });

  if (profileRes.status === 401) {
    invalidateVUToken(__VU);
  }

  const parsedProfile = parseJSON(profileRes.body);
  check(profileRes, {
    'profile status is 200': (r) => r.status === 200,
    'tenant isolation intact': () => {
      if (!parsedProfile || !parsedProfile.data) return false;
      return parsedProfile.data.id === tenant.tenant_id;
    },
  });

  // 2. Read: List Customers
  http.get(`${BASE_URL}/api/v1/tenant/customers?page=1&page_size=20`, {
    headers,
    tags: { name: 'GET /api/v1/tenant/customers', operation: 'read', tenant_id: tenant.tenant_id },
  });

  // 3. Read: List Banks
  http.get(`${BASE_URL}/api/v1/tenant/banks`, {
    headers,
    tags: { name: 'GET /api/v1/tenant/banks', operation: 'read', tenant_id: tenant.tenant_id },
  });

  // 4. Dashboard: Financial Summary
  http.get(`${BASE_URL}/api/v1/tenant/financial-summary`, {
    headers,
    tags: { name: 'GET /api/v1/tenant/financial-summary', operation: 'dashboard', tenant_id: tenant.tenant_id },
  });

  // 5. Dashboard: Comprehensive Metrics
  http.get(`${BASE_URL}/api/v1/tenant/dashboard`, {
    headers,
    tags: { name: 'GET /api/v1/tenant/dashboard', operation: 'dashboard', tenant_id: tenant.tenant_id },
  });

  // 6. Mutating Write: Create Customer (30% probability)
  if (Math.random() < 0.3) {
    const custPayload = JSON.stringify(makeCustomerPayload());
    http.post(`${BASE_URL}/api/v1/tenant/customers`, custPayload, {
      headers,
      tags: { name: 'POST /api/v1/tenant/customers', operation: 'write', tenant_id: tenant.tenant_id },
    });
  }

  // 7. Mutating Write: Record Counter Sale (20% probability)
  if (Math.random() < 0.2) {
    const salePayload = JSON.stringify(makeCounterSalePayload());
    http.post(`${BASE_URL}/api/v1/tenant/counter-sales`, salePayload, {
      headers,
      tags: { name: 'POST /api/v1/tenant/counter-sales', operation: 'write', tenant_id: tenant.tenant_id },
    });
  }

  // 8. Mutating Write: Record Store Expense (10% probability)
  if (Math.random() < 0.1) {
    const expPayload = JSON.stringify(makeExpensePayload());
    http.post(`${BASE_URL}/api/v1/tenant/expenses`, expPayload, {
      headers,
      tags: { name: 'POST /api/v1/tenant/expenses', operation: 'write', tenant_id: tenant.tenant_id },
    });
  }

  sleep(0.2);
}
