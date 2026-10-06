/**
 * Mixed Workload Load Test Scenario
 * Simulates realistic small-business SaaS traffic with configurable distribution:
 * - 40% Read APIs
 * - 25% Dashboard / Financial Metrics
 * - 15% Write / Mutations (sales, customers, expenses)
 * - 10% Authentication / Login
 * - 10% Health & Readiness Probes
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, assertSafety, THRESHOLDS, MIXED_WEIGHTS } from '../config.js';
import { buildHeaders, randomChoice, randomInt } from '../utils/helpers.js';
import { getTenantForVU, getUserForVU, makeCustomerPayload, makeCounterSalePayload, makeExpensePayload } from '../utils/data.js';
import { getValidTokenForVU, invalidateVUToken, tenantLogin } from '../utils/auth.js';

assertSafety();

export const options = {
  vus: parseInt(__ENV.VUS || '25', 10),
  duration: __ENV.DURATION || '2m',
  thresholds: {
    ...THRESHOLDS.standard,
    'http_req_duration{category:read}': ['p(95)<300', 'p(99)<600'],
    'http_req_duration{category:dashboard}': ['p(95)<400', 'p(99)<800'],
    'http_req_duration{category:write}': ['p(95)<500', 'p(99)<1000'],
    'http_req_duration{category:auth}': ['p(95)<1000', 'p(99)<2000'],
    'http_req_duration{category:health}': ['p(95)<50', 'p(99)<100'],
  },
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)'],
};

// Calculate cumulative threshold boundaries for weighted random selection
const totalWeight = MIXED_WEIGHTS.read + MIXED_WEIGHTS.dashboard + MIXED_WEIGHTS.write + MIXED_WEIGHTS.auth + MIXED_WEIGHTS.health;
const boundaryRead = MIXED_WEIGHTS.read;
const boundaryDashboard = boundaryRead + MIXED_WEIGHTS.dashboard;
const boundaryWrite = boundaryDashboard + MIXED_WEIGHTS.write;
const boundaryAuth = boundaryWrite + MIXED_WEIGHTS.auth;

export default function () {
  const tenant = getTenantForVU(__VU);
  const token = getValidTokenForVU(__VU, BASE_URL, tenant.admin);
  const headers = buildHeaders(token);
  const roll = Math.random() * totalWeight;

  if (roll < boundaryRead) {
    // ----------------------------------------------------
    // Category 1: Read APIs (40%)
    // ----------------------------------------------------
    const readEndpoint = randomChoice([
      '/api/v1/tenant/profile',
      '/api/v1/tenant/customers?page=1&page_size=20',
      '/api/v1/tenant/banks',
      '/api/v1/plans',
    ]);

    const res = http.get(`${BASE_URL}${readEndpoint}`, {
      headers,
      tags: { name: `GET ${readEndpoint.split('?')[0]}`, category: 'read', operation: 'mixed_read' },
    });
    if (res.status === 401) invalidateVUToken(__VU);
    check(res, { 'read status 200': (r) => r.status === 200 });

  } else if (roll < boundaryDashboard) {
    // ----------------------------------------------------
    // Category 2: Dashboard & Statistics (25%)
    // ----------------------------------------------------
    const dashEndpoint = randomChoice([
      '/api/v1/tenant/dashboard',
      '/api/v1/tenant/financial-summary',
      '/api/v1/tenant/dashboard/today',
      '/api/v1/tenant/metrics',
    ]);

    const res = http.get(`${BASE_URL}${dashEndpoint}`, {
      headers,
      tags: { name: `GET ${dashEndpoint}`, category: 'dashboard', operation: 'mixed_dashboard' },
    });
    if (res.status === 401) invalidateVUToken(__VU);
    check(res, { 'dashboard status 200': (r) => r.status === 200 });

  } else if (roll < boundaryWrite) {
    // ----------------------------------------------------
    // Category 3: Write Mutations (15%)
    // ----------------------------------------------------
    const writeType = randomChoice(['customer', 'sale', 'expense']);

    if (writeType === 'customer') {
      const payload = JSON.stringify(makeCustomerPayload());
      const res = http.post(`${BASE_URL}/api/v1/tenant/customers`, payload, {
        headers,
        tags: { name: 'POST /api/v1/tenant/customers', category: 'write', operation: 'mixed_write' },
      });
      if (res.status === 401) invalidateVUToken(__VU);
      check(res, { 'create customer status 201': (r) => r.status === 201 || r.status === 200 });
    } else if (writeType === 'sale') {
      const payload = JSON.stringify(makeCounterSalePayload());
      const res = http.post(`${BASE_URL}/api/v1/tenant/counter-sales`, payload, {
        headers,
        tags: { name: 'POST /api/v1/tenant/counter-sales', category: 'write', operation: 'mixed_write' },
      });
      if (res.status === 401) invalidateVUToken(__VU);
      check(res, { 'create sale status 201': (r) => r.status === 201 || r.status === 200 });
    } else {
      const payload = JSON.stringify(makeExpensePayload());
      const res = http.post(`${BASE_URL}/api/v1/tenant/expenses`, payload, {
        headers,
        tags: { name: 'POST /api/v1/tenant/expenses', category: 'write', operation: 'mixed_write' },
      });
      if (res.status === 401) invalidateVUToken(__VU);
      check(res, { 'create expense status 201': (r) => r.status === 201 || r.status === 200 });
    }

  } else if (roll < boundaryAuth) {
    // ----------------------------------------------------
    // Category 4: Authentication (10%)
    // ----------------------------------------------------
    const user = getUserForVU(__VU + randomInt(1, 100));
    const authRes = tenantLogin(BASE_URL, user.email, user.password, { category: 'auth' });
    check(authRes.response, {
      'auth login status 200': (r) => r.status === 200,
    });

  } else {
    // ----------------------------------------------------
    // Category 5: Health & Readiness (10%)
    // ----------------------------------------------------
    const headers = buildHeaders();
    const endpoint = Math.random() < 0.5 ? '/health' : '/ready';
    const res = http.get(`${BASE_URL}${endpoint}`, {
      headers,
      tags: { name: `GET ${endpoint}`, category: 'health', operation: 'mixed_health' },
    });
    check(res, { 'health status 200': (r) => r.status === 200 });
  }

  // Realistic user pacing between requests (100ms - 300ms)
  sleep(0.15);
}
