/**
 * Vyavsa API Client
 * Standardized client for backend API communication
 */

const API = (() => {
  const BASE_URL = '/api/v1';

  /**
   * Helper to parse error responses from backend
   * @param {Response} response
   * @param {any} data
   * @returns {string}
   */
  function extractErrorMessage(response, data) {
    if (!data) {
      if (response.status === 404) return 'The requested resource was not found.';
      if (response.status === 401) return 'Invalid credentials or session expired.';
      if (response.status === 403) return 'You do not have permission to perform this action.';
      if (response.status === 429) return 'Too many requests. Please slow down and try again later.';
      if (response.status >= 500) return 'A server error occurred. Please try again later.';
      return `Request failed with status ${response.status}.`;
    }

    if (typeof data === 'string') return data;
    if (data.error) {
      if (typeof data.error === 'string') return data.error;
      if (typeof data.error.message === 'string') return data.error.message;
    }
    if (data.message) return data.message;
    if (Array.isArray(data.errors) && data.errors.length > 0) {
      return data.errors.join(', ');
    }

    return 'An unexpected error occurred. Please try again.';
  }

  // Single-flight refresh locks for concurrent requests
  const refreshLocks = {
    tenant: null,
    platform: null
  };

  /**
   * Check if endpoint is an authentication endpoint that must not trigger refresh
   * @param {string} endpoint
   * @returns {boolean}
   */
  function isExcludedEndpoint(endpoint) {
    if (!endpoint) return false;
    const excluded = [
      '/auth/refresh',
      '/auth/tenant/login',
      '/auth/platform/login',
      '/auth/tenant/register',
      '/auth/forgot-password',
      '/auth/verify-reset-otp',
      '/auth/reset-password',
      '/auth/revoke',
      '/auth/logout'
    ];
    return excluded.some(p => endpoint.includes(p));
  }

  /**
   * Core request function with transparent token refresh & retry interceptor
   * @param {string} endpoint - API path (e.g. '/tenant/employees')
   * @param {object} options - Fetch options
   * @returns {Promise<{ ok: boolean, status: number, data: any, error: string|null, details?: any, errors?: any }>}
   */
  async function request(endpoint, options = {}) {
    const isPlatform = !!(options.isPlatform || (typeof window !== 'undefined' && window.location && window.location.pathname.startsWith('/platform')));
    const lockKey = isPlatform ? 'platform' : 'tenant';

    // 1. If another request is currently refreshing this token type, await completion before sending
    if (!options.skipAuth && !options.skipRefresh && !options._retry && !isExcludedEndpoint(endpoint)) {
      if (refreshLocks[lockKey]) {
        try {
          await refreshLocks[lockKey];
        } catch (e) {
          // If refresh failed, current request will proceed and fail with proper error
        }
      }

      // 2. Proactive refresh: if token is already expired and refresh token is available, refresh before sending
      if (window.Auth && window.Auth.isTokenExpired) {
        const curToken = isPlatform ? window.Auth.getPlatformToken() : window.Auth.getAccessToken();
        const hasRefresh = isPlatform ? !!window.Auth.getPlatformRefreshToken() : !!window.Auth.getRefreshToken();
        if ((!curToken || window.Auth.isTokenExpired(curToken)) && hasRefresh) {
          if (!refreshLocks[lockKey]) {
            refreshLocks[lockKey] = window.Auth.refreshToken(isPlatform).finally(() => {
              refreshLocks[lockKey] = null;
            });
          }
          try {
            await refreshLocks[lockKey];
          } catch (e) {
            // Let the request fail naturally with standard handling
          }
        }
      }
    }

    const url = endpoint.startsWith('http') ? endpoint : `${BASE_URL}${endpoint}`;
    const headers = {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
      ...(options.headers || {})
    };

    // Attach token if available and not explicitly skipped
    if (!options.skipAuth && window.Auth) {
      const token = isPlatform ? window.Auth.getPlatformToken() : window.Auth.getAccessToken();
      if (token) {
        headers['Authorization'] = `Bearer ${token}`;
      }
    }

    const config = {
      ...options,
      headers
    };

    if (options.body && typeof options.body === 'object') {
      config.body = JSON.stringify(options.body);
    }

    try {
      const response = await fetch(url, config);
      const isJson = (response.headers.get('content-type') || '').includes('application/json');
      const data = isJson ? await response.json() : await response.text();

      // 3. Reactive 401 Interceptor: detect expired access token, refresh, and retry exactly once
      if (response.status === 401 && !options._retry && !options.skipAuth && !options.skipRefresh && !isExcludedEndpoint(endpoint) && window.Auth) {
        const hasRefresh = isPlatform ? !!window.Auth.getPlatformRefreshToken() : !!window.Auth.getRefreshToken();
        if (hasRefresh) {
          // Initiate single-flight refresh if not already in progress
          if (!refreshLocks[lockKey]) {
            refreshLocks[lockKey] = window.Auth.refreshToken(isPlatform).finally(() => {
              refreshLocks[lockKey] = null;
            });
          }

          let newToken = null;
          try {
            newToken = await refreshLocks[lockKey];
          } catch (refreshErr) {
            // Refresh failed: invalid/expired refresh token
            if (window.Auth.handleAuthFailure) {
              window.Auth.handleAuthFailure(isPlatform);
            }
            const errorMsg = extractErrorMessage(response, data);
            return {
              ok: false,
              status: 401,
              data: null,
              error: errorMsg || 'Session expired. Please log in again.'
            };
          }

          if (newToken) {
            // Retry the original request exactly once with new access token
            const retryHeaders = {
              ...(options.headers || {}),
              'Authorization': `Bearer ${newToken}`
            };
            return await request(endpoint, {
              ...options,
              _retry: true,
              headers: retryHeaders
            });
          }
        } else {
          // No refresh token available
          if (window.Auth.handleAuthFailure) {
            window.Auth.handleAuthFailure(isPlatform);
          }
        }
      }

      if (!response.ok) {
        const errorMsg = extractErrorMessage(response, data);
        const details = data && data.error && data.error.details ? data.error.details : null;
        return {
          ok: false,
          status: response.status,
          data: null,
          error: errorMsg,
          details: details,
          errors: data && typeof data === 'object' ? data.errors : null
        };
      }

      // Check for standard Vyavsa API response wrapper
      const payload = (data && typeof data === 'object' && 'data' in data) ? data.data : data;
      const message = (data && typeof data === 'object' && data.message) ? data.message : null;

      return {
        ok: true,
        status: response.status,
        data: payload,
        message: message,
        raw: data,
        error: null
      };
    } catch (err) {
      return {
        ok: false,
        status: 0,
        data: null,
        error: 'Unable to connect to the server. Please check your internet connection and try again.'
      };
    }
  }

  return {
    get: (endpoint, options = {}) => request(endpoint, { ...options, method: 'GET' }),
    post: (endpoint, body, options = {}) => request(endpoint, { ...options, method: 'POST', body }),
    put: (endpoint, body, options = {}) => request(endpoint, { ...options, method: 'PUT', body }),
    patch: (endpoint, body, options = {}) => request(endpoint, { ...options, method: 'PATCH', body }),
    delete: (endpoint, options = {}) => request(endpoint, { ...options, method: 'DELETE' }),
    request
  };

})();

window.API = API;
