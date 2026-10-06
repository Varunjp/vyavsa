/**
 * Authentication helper routines for Vyavsa load tests.
 */

import http from 'k6/http';
import { buildHeaders, parseJSON } from './helpers.js';

/**
 * Authenticates a tenant user (admin or staff) via POST /api/v1/auth/tenant/login.
 */
export function tenantLogin(baseUrl, email, password, customTags = {}) {
  const payload = JSON.stringify({ email, password });
  const headers = buildHeaders();

  const res = http.post(`${baseUrl}/api/v1/auth/tenant/login`, payload, {
    headers,
    tags: { name: 'POST /api/v1/auth/tenant/login', operation: 'tenant_login', ...customTags },
  });

  const parsed = parseJSON(res.body);
  const success = res.status === 200 && parsed && parsed.success && parsed.data && parsed.data.access_token;

  return {
    response: res,
    success: !!success,
    status: res.status,
    accessToken: success ? parsed.data.access_token : null,
    refreshToken: success ? parsed.data.refresh_token : null,
    tenantId: success && parsed.data.user ? parsed.data.user.tenant_id : null,
    userId: success && parsed.data.user ? parsed.data.user.id : null,
  };
}

/**
 * Authenticates a platform administrator via POST /api/v1/auth/platform/login.
 */
export function platformLogin(baseUrl, identifier, password) {
  const payload = JSON.stringify({ identifier, password });
  const headers = buildHeaders();

  const res = http.post(`${baseUrl}/api/v1/auth/platform/login`, payload, {
    headers,
    tags: { name: 'POST /api/v1/auth/platform/login', operation: 'platform_login' },
  });

  const parsed = parseJSON(res.body);
  const success = res.status === 200 && parsed && parsed.success && parsed.data && parsed.data.access_token;

  return {
    response: res,
    success: !!success,
    status: res.status,
    accessToken: success ? parsed.data.access_token : null,
  };
}

/**
 * Exchanges a refresh token for a new token pair via POST /api/v1/auth/refresh.
 */
export function refreshAuthToken(baseUrl, refreshToken) {
  const payload = JSON.stringify({ refresh_token: refreshToken });
  const headers = buildHeaders();

  const res = http.post(`${baseUrl}/api/v1/auth/refresh`, payload, {
    headers,
    tags: { name: 'POST /api/v1/auth/refresh', operation: 'token_refresh' },
  });

  const parsed = parseJSON(res.body);
  return {
    response: res,
    success: res.status === 200 && parsed && parsed.success,
    accessToken: parsed && parsed.data ? parsed.data.access_token : null,
  };
}

/**
 * Logs out an authenticated user, revoking the token in Redis via POST /api/v1/auth/logout.
 */
export function logout(baseUrl, accessToken) {
  const headers = buildHeaders(accessToken);

  return http.post(`${baseUrl}/api/v1/auth/logout`, '{}', {
    headers,
    tags: { name: 'POST /api/v1/auth/logout', operation: 'logout' },
  });
}

// In-memory token cache keyed by VU index to support long runs and automatic re-authentication
const vuTokenCache = {};

/**
 * Returns a valid cached access token for the given VU or logs in on the fly.
 */
export function getValidTokenForVU(vuIndex, baseUrl, user) {
  if (vuTokenCache[vuIndex]) {
    return vuTokenCache[vuIndex];
  }

  // Authenticate dynamically to ensure fresh, non-expired token
  if (user && user.email && user.password) {
    const authRes = tenantLogin(baseUrl, user.email, user.password);
    if (authRes.success) {
      vuTokenCache[vuIndex] = authRes.accessToken;
      return authRes.accessToken;
    }
  }

  // Fallback to pre-seeded token if login failed or credentials missing
  if (user && user.accessToken) {
    return user.accessToken;
  }
  if (user && user.access_token) {
    return user.access_token;
  }

  return null;
}

/**
 * Invalidates cached token for a VU (e.g. on 401 Unauthorized), forcing re-login on next request.
 */
export function invalidateVUToken(vuIndex) {
  delete vuTokenCache[vuIndex];
}
