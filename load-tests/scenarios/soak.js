/**
 * Soak (Endurance) Test Scenario
 * Evaluates system stability, memory leaks, goroutine leaks, and connection pool behavior under prolonged constant load.
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, assertSafety, THRESHOLDS } from '../config.js';
import { buildHeaders, randomChoice } from '../utils/helpers.js';
import { getTenantForVU, getUserForVU, makeCustomerPayload, makeCounterSalePayload } from '../utils/data.js';
import { getValidTokenForVU, invalidateVUToken } from '../utils/auth.js';

assertSafety();

export const options = {
  vus: parseInt(__ENV.SOAK_VUS || '35', 10),
  duration: __ENV.SOAK_DURATION || '5m', // Configurable (e.g. 10m, 30m, 1h via SOAK_DURATION)
  thresholds: {
    ...THRESHOLDS.standard,
  },
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)'],
};

export default function () {
  const user = getUserForVU(__VU);
  const token = getValidTokenForVU(__VU, BASE_URL, user);
  const headers = buildHeaders(token);

  const roll = Math.random();

  if (roll < 0.45) {
    const res = http.get(`${BASE_URL}/api/v1/tenant/profile`, {
      headers,
      tags: { name: 'GET /api/v1/tenant/profile', type: 'soak_read' },
    });
    if (res.status === 401) invalidateVUToken(__VU);
  } else if (roll < 0.8) {
    const res = http.get(`${BASE_URL}/api/v1/tenant/dashboard`, {
      headers,
      tags: { name: 'GET /api/v1/tenant/dashboard', type: 'soak_dash' },
    });
    if (res.status === 401) invalidateVUToken(__VU);
  } else {
    // 20% write mutation to continuously exercise DB write transactions
    const payload = JSON.stringify(makeCounterSalePayload());
    const res = http.post(`${BASE_URL}/api/v1/tenant/counter-sales`, payload, {
      headers,
      tags: { name: 'POST /api/v1/tenant/counter-sales', type: 'soak_write' },
    });
    if (res.status === 401) invalidateVUToken(__VU);
  }

  // Consistent pacing
  sleep(0.25);
}
