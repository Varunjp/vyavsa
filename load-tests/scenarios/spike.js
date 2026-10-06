/**
 * Traffic Spike Test Scenario
 * Evaluates elasticity and recovery time when traffic abruptly spikes from baseline to heavy load and returns to normal.
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, assertSafety } from '../config.js';
import { buildHeaders } from '../utils/helpers.js';
import { getTenantForVU, getUserForVU } from '../utils/data.js';
import { getValidTokenForVU, invalidateVUToken } from '../utils/auth.js';

assertSafety();

const spikeTarget = parseInt(__ENV.SPIKE_VUS || '250', 10);

export const options = {
  stages: [
    { duration: '15s', target: 10 },          // Warm-up at 10 VUs
    { duration: '10s', target: spikeTarget },  // Fast spike to 250 VUs
    { duration: '25s', target: spikeTarget },  // Sustain the burst
    { duration: '15s', target: 10 },          // Drop back to 10 VUs
    { duration: '20s', target: 10 },          // Recovery observation period
    { duration: '5s', target: 0 },            // Ramp down
  ],
  thresholds: {
    http_req_failed: ['rate<0.05'], // Max 5% fail during extreme spike
  },
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)'],
};

export default function () {
  const user = getUserForVU(__VU);
  const token = getValidTokenForVU(__VU, BASE_URL, user);
  const headers = buildHeaders(token);

  // Mixed queries during spike
  if (Math.random() < 0.6) {
    const res = http.get(`${BASE_URL}/api/v1/tenant/dashboard`, {
      headers,
      tags: { name: 'GET /api/v1/tenant/dashboard', spike_phase: 'query' },
    });
    if (res.status === 401) {
      invalidateVUToken(__VU);
    }
  } else {
    http.get(`${BASE_URL}/health`, {
      headers,
      tags: { name: 'GET /health', spike_phase: 'health' },
    });
  }

  sleep(0.1);
}
