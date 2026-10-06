/**
 * Vyavsa Load Testing Configuration
 * Defines environment variables, safety gates, thresholds, and execution options.
 */

export const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
export const TEST_EMAIL = __ENV.TEST_EMAIL || 'admin@demostore.com';
export const TEST_PASSWORD = __ENV.TEST_PASSWORD || 'Store@12345';
export const TEST_TENANT_ID = __ENV.TEST_TENANT_ID || '11111111-1111-1111-1111-111111111111';
export const ACCESS_TOKEN = __ENV.ACCESS_TOKEN || '';
export const ENVIRONMENT = (__ENV.ENVIRONMENT || __ENV.APP_ENV || 'development').toLowerCase();
export const LOAD_TEST_ENABLED = (__ENV.LOAD_TEST_ENABLED || 'true').toLowerCase();

export const TENANT_COUNT = parseInt(__ENV.TENANT_COUNT || '50', 10);
export const USERS_PER_TENANT = parseInt(__ENV.USERS_PER_TENANT || '2', 10);

// Assert safety to prevent accidental test runs against production
export function assertSafety() {
  if (ENVIRONMENT === 'production') {
    throw new Error(`[SAFETY GUARD] Load tests must NEVER run against production! Detected ENVIRONMENT=${ENVIRONMENT}`);
  }
  if (LOAD_TEST_ENABLED !== 'true' && LOAD_TEST_ENABLED !== '1') {
    throw new Error(`[SAFETY GUARD] Load testing disabled. Set LOAD_TEST_ENABLED=true to confirm execution.`);
  }
}

// Configurable Mixed Workload Percentages
export const MIXED_WEIGHTS = {
  read: parseInt(__ENV.MIX_READ || '40', 10),
  dashboard: parseInt(__ENV.MIX_DASHBOARD || '25', 10),
  write: parseInt(__ENV.MIX_WRITE || '15', 10),
  auth: parseInt(__ENV.MIX_AUTH || '10', 10),
  health: parseInt(__ENV.MIX_HEALTH || '10', 10),
};

// Common SLO Thresholds
export const THRESHOLDS = {
  standard: {
    http_req_failed: ['rate<0.01'], // Error rate < 1%
    http_req_duration: ['p(95)<500', 'p(99)<1000'], // p95 < 500ms, p99 < 1000ms
  },
  health: {
    http_req_failed: ['rate<0.001'], // 99.9% success
    http_req_duration: ['p(95)<50', 'p(99)<100'], // p95 < 50ms, p99 < 100ms
  },
  auth: {
    http_req_failed: ['rate<0.02'], // Error rate < 2%
    http_req_duration: ['p(95)<1200', 'p(99)<2500'], // Bcrypt is CPU-bound
  },
  stress: {
    http_req_failed: ['rate<0.05'], // Keep under 5% during stress test
    http_req_duration: ['p(95)<1500'],
  },
};
