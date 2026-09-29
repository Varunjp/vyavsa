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

  /**
   * Core request function
   * @param {string} endpoint - API path (e.g. '/auth/tenant/login')
   * @param {object} options - Fetch options
   * @returns {Promise<{ ok: boolean, status: number, data: any, error: string|null, errors?: any }>}
   */
  async function request(endpoint, options = {}) {
    const url = endpoint.startsWith('http') ? endpoint : `${BASE_URL}${endpoint}`;
    const headers = {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
      ...(options.headers || {})
    };

    // Attach token if available and not explicitly skipped
    if (!options.skipAuth && window.Auth) {
      const token = options.isPlatform ? window.Auth.getPlatformToken() : window.Auth.getAccessToken();
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
    delete: (endpoint, options = {}) => request(endpoint, { ...options, method: 'DELETE' }),
    request
  };
})();

window.API = API;
