/**
 * Gradual Ramp-Up Load Test Scenario
 * Evaluates application throughput and latency degradation inflection points under escalating concurrency.
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, assertSafety, THRESHOLDS } from '../config.js';
import { buildHeaders, randomChoice } from '../utils/helpers.js';
import { getTenantForVU, getUserForVU } from '../utils/data.js';
import { getValidTokenForVU, invalidateVUToken } from '../utils/auth.js';

assertSafety();

const stageMultiplier = parseFloat(__ENV.STAGE_DURATION_SCALE || '1.0');

export const options = {
  stages: [
    { duration: `${Math.round(60 * stageMultiplier)}s`, target: 25 },   // Ramp to 25 VUs
    { duration: `${Math.round(90 * stageMultiplier)}s`, target: 50 },   // Step to 50 VUs
    { duration: `${Math.round(120 * stageMultiplier)}s`, target: 100 }, // Step to 100 VUs
    { duration: `${Math.round(120 * stageMultiplier)}s`, target: 150 }, // Step to 150 VUs
    { duration: `${Math.round(60 * stageMultiplier)}s`, target: 0 },    // Ramp down to 0
  ],
  thresholds: {
    ...THRESHOLDS.standard,
  },
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)'],
};

export default function () {
  const user = getUserForVU(__VU);
  const token = getValidTokenForVU(__VU, BASE_URL, user);
  const headers = buildHeaders(token);

  // Mix of read, dashboard, and health probes
  const target = randomChoice(['profile', 'dashboard', 'financial-summary', 'health']);

  if (target === 'profile') {
    const res = http.get(`${BASE_URL}/api/v1/tenant/profile`, {
      headers,
      tags: { name: 'GET /api/v1/tenant/profile', type: 'read' },
    });
    if (res.status === 401) invalidateVUToken(__VU);
  } else if (target === 'dashboard') {
    const res = http.get(`${BASE_URL}/api/v1/tenant/dashboard`, {
      headers,
      tags: { name: 'GET /api/v1/tenant/dashboard', type: 'dashboard' },
    });
    if (res.status === 401) invalidateVUToken(__VU);
  } else if (target === 'financial-summary') {
    const res = http.get(`${BASE_URL}/api/v1/tenant/financial-summary`, {
      headers,
      tags: { name: 'GET /api/v1/tenant/financial-summary', type: 'dashboard' },
    });
    if (res.status === 401) invalidateVUToken(__VU);
  } else {
    http.get(`${BASE_URL}/health`, {
      headers,
      tags: { name: 'GET /health', type: 'health' },
    });
  }

  sleep(0.1);
}
