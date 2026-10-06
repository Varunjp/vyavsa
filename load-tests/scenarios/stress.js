/**
 * Stress Test Scenario
 * Aggressively pushes concurrency to locate the system breaking point, maximum stable RPS, and saturation thresholds.
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, assertSafety, THRESHOLDS } from '../config.js';
import { buildHeaders, randomChoice } from '../utils/helpers.js';
import { getTenantForVU, getUserForVU, makeCustomerPayload, makeCounterSalePayload } from '../utils/data.js';
import { getValidTokenForVU, invalidateVUToken } from '../utils/auth.js';

assertSafety();

const stageMultiplier = parseFloat(__ENV.STAGE_DURATION_SCALE || '1.0');

export const options = {
  stages: [
    { duration: `${Math.round(45 * stageMultiplier)}s`, target: 50 },  // 50 VUs
    { duration: `${Math.round(60 * stageMultiplier)}s`, target: 100 }, // 100 VUs
    { duration: `${Math.round(60 * stageMultiplier)}s`, target: 200 }, // 200 VUs
    { duration: `${Math.round(60 * stageMultiplier)}s`, target: 350 }, // 350 VUs
    { duration: `${Math.round(45 * stageMultiplier)}s`, target: 0 },   // Cool-down
  ],
  thresholds: {
    ...THRESHOLDS.stress,
  },
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)'],
};

export default function () {
  const user = getUserForVU(__VU);
  const token = getValidTokenForVU(__VU, BASE_URL, user);
  const headers = buildHeaders(token);

  // Exercise read, dashboard, and transactional endpoints
  const roll = Math.random();

  if (roll < 0.5) {
    const res = http.get(`${BASE_URL}/api/v1/tenant/dashboard`, {
      headers,
      tags: { name: 'GET /api/v1/tenant/dashboard', type: 'stress_dash' },
    });
    if (res.status === 401) invalidateVUToken(__VU);
  } else if (roll < 0.8) {
    const res = http.get(`${BASE_URL}/api/v1/tenant/profile`, {
      headers,
      tags: { name: 'GET /api/v1/tenant/profile', type: 'stress_read' },
    });
    if (res.status === 401) invalidateVUToken(__VU);
  } else {
    // 20% write mutation under stress
    const payload = JSON.stringify(makeCounterSalePayload());
    http.post(`${BASE_URL}/api/v1/tenant/counter-sales`, payload, {
      headers,
      tags: { name: 'POST /api/v1/tenant/counter-sales', type: 'stress_write' },
    });
  }

  sleep(0.05);
}
