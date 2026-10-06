/**
 * Authentication Load Test Scenario
 * Evaluates tenant user login, Bcrypt hashing throughput, token generation, refresh, and logout.
 */

import { check, sleep } from 'k6';
import { Trend, Counter } from 'k6/metrics';
import { BASE_URL, assertSafety, THRESHOLDS } from '../config.js';
import { getUserForVU } from '../utils/data.js';
import { tenantLogin, refreshAuthToken, logout } from '../utils/auth.js';

assertSafety();

const loginDuration = new Trend('auth_login_duration', true);
const loginSuccesses = new Counter('auth_login_success_total');
const loginFailures = new Counter('auth_login_failure_total');

export const options = {
  vus: parseInt(__ENV.VUS || '15', 10),
  duration: __ENV.DURATION || '60s',
  thresholds: {
    ...THRESHOLDS.auth,
    auth_login_duration: ['p(95)<1000', 'p(99)<2000'],
  },
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)'],
};

export default function () {
  const user = getUserForVU(__VU);

  // 1. Tenant User Login
  const loginRes = tenantLogin(BASE_URL, user.email, user.password);
  loginDuration.add(loginRes.response.timings.duration);

  const loginOk = check(loginRes.response, {
    'login status is 200': (r) => r.status === 200,
    'access token returned': () => !!loginRes.accessToken,
  });

  if (loginOk) {
    loginSuccesses.add(1);

    // 2. Refresh Token validation (20% sample of successful logins)
    if (loginRes.refreshToken && Math.random() < 0.2) {
      const refreshRes = refreshAuthToken(BASE_URL, loginRes.refreshToken);
      check(refreshRes.response, {
        'refresh status is 200': (r) => r.status === 200,
        'new access token received': () => !!refreshRes.accessToken,
      });
    }

    // 3. Logout & Token Blacklist in Redis (10% sample)
    if (loginRes.accessToken && Math.random() < 0.1) {
      const logoutRes = logout(BASE_URL, loginRes.accessToken);
      check(logoutRes, {
        'logout status is 200': (r) => r.status === 200,
      });
    }
  } else {
    loginFailures.add(1);
  }

  // Small pacing to avoid instantaneous CPU thrashing on bcrypt
  sleep(0.5);
}
