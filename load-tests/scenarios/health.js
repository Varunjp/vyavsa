/**
 * Baseline Health & Readiness Load Test Scenario
 * Evaluates core framework latency, Gin router throughput, and operational health probes.
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, assertSafety, THRESHOLDS } from '../config.js';
import { buildHeaders } from '../utils/helpers.js';

assertSafety();

export const options = {
  vus: parseInt(__ENV.VUS || '10', 10),
  duration: __ENV.DURATION || '60s',
  thresholds: {
    ...THRESHOLDS.health,
    'http_req_duration{endpoint:health}': ['p(95)<30', 'p(99)<60'],
    'http_req_duration{endpoint:ready}': ['p(95)<60', 'p(99)<120'],
  },
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)'],
};

export default function () {
  const headers = buildHeaders();

  // 1. Health Probe (Process Liveness)
  const healthRes = http.get(`${BASE_URL}/health`, {
    headers,
    tags: { name: 'GET /health', endpoint: 'health' },
  });

  check(healthRes, {
    'health status is 200': (r) => r.status === 200,
    'health body contains UP': (r) => r.body && r.body.includes('UP'),
  });

  // 2. Readiness Probe (PostgreSQL & Redis Dependency Health)
  const readyRes = http.get(`${BASE_URL}/ready`, {
    headers,
    tags: { name: 'GET /ready', endpoint: 'ready' },
  });

  check(readyRes, {
    'ready status is 200': (r) => r.status === 200,
    'ready body contains READY': (r) => r.body && r.body.includes('READY'),
  });

  sleep(0.1);
}
