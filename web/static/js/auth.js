/**
 * Vyavsa Authentication & Session Manager
 * Manages tokens, user context, and route protection
 */

const Auth = (() => {
  const STORAGE_KEYS = {
    ACCESS_TOKEN: 'vyavsa_access_token',
    REFRESH_TOKEN: 'vyavsa_refresh_token',
    USER: 'vyavsa_user',
    PLATFORM_ACCESS_TOKEN: 'vyavsa_platform_access_token',
    PLATFORM_REFRESH_TOKEN: 'vyavsa_platform_refresh_token',
    PLATFORM_USER: 'vyavsa_platform_user'
  };

  function safeStorageGet(key) {
    try {
      return localStorage.getItem(key);
    } catch (e) {
      return null;
    }
  }

  function safeStorageSet(key, val) {
    try {
      localStorage.setItem(key, val);
    } catch (e) {
      // Storage unavailable/disabled
    }
  }

  function safeStorageRemove(key) {
    try {
      localStorage.removeItem(key);
    } catch (e) {}
  }

  // Tenant Session Management
  function setTenantAuth(tokenResp) {
    if (!tokenResp) return;
    if (tokenResp.access_token) safeStorageSet(STORAGE_KEYS.ACCESS_TOKEN, tokenResp.access_token);
    if (tokenResp.refresh_token) safeStorageSet(STORAGE_KEYS.REFRESH_TOKEN, tokenResp.refresh_token);
    if (tokenResp.user) safeStorageSet(STORAGE_KEYS.USER, JSON.stringify(tokenResp.user));
  }

  function getAccessToken() {
    return safeStorageGet(STORAGE_KEYS.ACCESS_TOKEN);
  }

  function getRefreshToken() {
    return safeStorageGet(STORAGE_KEYS.REFRESH_TOKEN);
  }

  function getUser() {
    const raw = safeStorageGet(STORAGE_KEYS.USER);
    if (!raw) return null;
    try {
      return JSON.parse(raw);
    } catch (e) {
      return null;
    }
  }

  /**
   * Parse JWT payload and check expiration
   * @param {string} token
   * @param {number} thresholdSeconds - Safety buffer in seconds (default 5)
   * @returns {boolean} true if missing, malformed, or expired
   */
  function isTokenExpired(token, thresholdSeconds = 5) {
    if (!token || typeof token !== 'string') return true;
    try {
      const parts = token.split('.');
      if (parts.length !== 3) return true;
      const base64Url = parts[1];
      let base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
      while (base64.length % 4 !== 0) {
        base64 += '=';
      }
      const jsonPayload = decodeURIComponent(
        Array.prototype.map.call(atob(base64), function(c) {
          return '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2);
        }).join('')
      );
      const payload = JSON.parse(jsonPayload);
      if (!payload || typeof payload.exp !== 'number') return false;
      const nowSeconds = Math.floor(Date.now() / 1000);
      return payload.exp <= (nowSeconds + thresholdSeconds);
    } catch (e) {
      return true;
    }
  }

  function isAuthenticated() {
    const at = getAccessToken();
    if (at && !isTokenExpired(at)) return true;
    const rt = getRefreshToken();
    if (rt && !isTokenExpired(rt)) return true;
    return false;
  }

  function clearTenantAuth() {
    safeStorageRemove(STORAGE_KEYS.ACCESS_TOKEN);
    safeStorageRemove(STORAGE_KEYS.REFRESH_TOKEN);
    safeStorageRemove(STORAGE_KEYS.USER);
    try { sessionStorage.clear(); } catch(e) {}
  }

  // Platform Admin Session Management
  function setPlatformAuth(tokenResp) {
    if (!tokenResp) return;
    if (tokenResp.access_token) safeStorageSet(STORAGE_KEYS.PLATFORM_ACCESS_TOKEN, tokenResp.access_token);
    if (tokenResp.refresh_token) safeStorageSet(STORAGE_KEYS.PLATFORM_REFRESH_TOKEN, tokenResp.refresh_token);
    if (tokenResp.user) safeStorageSet(STORAGE_KEYS.PLATFORM_USER, JSON.stringify(tokenResp.user));
  }

  function getPlatformToken() {
    return safeStorageGet(STORAGE_KEYS.PLATFORM_ACCESS_TOKEN);
  }

  function isPlatformAuthenticated() {
    const at = getPlatformToken();
    if (at && !isTokenExpired(at)) return true;
    const rt = getPlatformRefreshToken();
    if (rt && !isTokenExpired(rt)) return true;
    return false;
  }

  function getPlatformRefreshToken() {
    return safeStorageGet(STORAGE_KEYS.PLATFORM_REFRESH_TOKEN);
  }

  function clearPlatformAuth() {
    safeStorageRemove(STORAGE_KEYS.PLATFORM_ACCESS_TOKEN);
    safeStorageRemove(STORAGE_KEYS.PLATFORM_REFRESH_TOKEN);
    safeStorageRemove(STORAGE_KEYS.PLATFORM_USER);
    try { sessionStorage.clear(); } catch(e) {}
  }

  /**
   * Request a fresh access token from backend using refresh token
   * @param {boolean} isPlatform
   * @returns {Promise<string>} new access token
   */
  async function refreshToken(isPlatform = false) {
    const rf = isPlatform ? getPlatformRefreshToken() : getRefreshToken();
    if (!rf) {
      throw new Error('No refresh token available');
    }

    const response = await fetch('/api/v1/auth/refresh', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json'
      },
      body: JSON.stringify({ refresh_token: rf })
    });

    const isJson = (response.headers.get('content-type') || '').includes('application/json');
    const data = isJson ? await response.json() : await response.text();

    if (!response.ok) {
      let errorMsg = 'Failed to refresh token';
      if (data && typeof data === 'object') {
        if (data.error) {
          errorMsg = typeof data.error === 'string' ? data.error : (data.error.message || errorMsg);
        } else if (data.message) {
          errorMsg = data.message;
        }
      }
      throw new Error(errorMsg);
    }

    const tokenResp = (data && typeof data === 'object' && data.data) ? data.data : data;
    if (!tokenResp || !tokenResp.access_token) {
      throw new Error('Malformed token response from server');
    }

    if (isPlatform) {
      setPlatformAuth(tokenResp);
    } else {
      setTenantAuth(tokenResp);
    }

    return tokenResp.access_token;
  }

  /**
   * Ensure an active, unexpired access token is available.
   * If access token is expired or missing but refresh token is valid, refreshes session.
   * @param {boolean} isPlatform
   * @returns {Promise<string>} active access token
   */
  async function ensureSession(isPlatform = false) {
    const at = isPlatform ? getPlatformToken() : getAccessToken();
    if (at && !isTokenExpired(at)) {
      return at;
    }
    const rf = isPlatform ? getPlatformRefreshToken() : getRefreshToken();
    if (rf && !isTokenExpired(rf)) {
      return await refreshToken(isPlatform);
    }
    throw new Error('No valid session credentials available');
  }

  /**
   * Centralized authentication failure handler
   * Clears stored credentials and redirects to appropriate login
   */
  function handleAuthFailure(isPlatform = false) {
    if (isPlatform) {
      clearPlatformAuth();
      if (typeof window !== 'undefined' && window.location && !window.location.pathname.startsWith('/platform/login')) {
        window.location.replace('/platform/login');
      }
    } else {
      clearTenantAuth();
      if (typeof window !== 'undefined' && window.location && !window.location.pathname.startsWith('/login')) {
        window.location.replace('/login');
      }
    }
  }

  // Session Invalidation
  async function logout() {
    try {
      const rf = getRefreshToken();
      if (rf && window.API) {
        await window.API.post('/auth/revoke', { refresh_token: rf }, { skipRefresh: true, skipAuth: true });
      }
    } catch (e) {
      // Proceed with local cleanup regardless of network error
    } finally {
      clearTenantAuth();
      clearPlatformAuth();
      window.location.replace('/login');
    }
  }

  async function platformLogout() {
    try {
      const rf = getPlatformRefreshToken();
      if (rf && window.API) {
        await window.API.post('/auth/revoke', { refresh_token: rf }, { skipRefresh: true, skipAuth: true, isPlatform: true });
      }
    } catch (e) {
      // Proceed with local cleanup regardless of network error
    } finally {
      clearPlatformAuth();
      window.location.replace('/platform/login');
    }
  }

  // UI Helper: Alerts inside form containers
  function showAlert(container, message, type = 'error') {
    if (!container) return;
    container.innerHTML = `
      <div class="alert alert-${type} alert-dismissible" role="alert">
        <svg class="alert-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          ${type === 'success' 
            ? '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>' 
            : '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>'
          }
        </svg>
        <div class="alert-content">${escapeHTML(message)}</div>
      </div>
    `;
    container.style.display = 'block';
  }

  function clearAlert(container) {
    if (!container) return;
    container.innerHTML = '';
    container.style.display = 'none';
  }

  function escapeHTML(str) {
    if (!str) return '';
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
  }

  // UI Helper: Loading state for submit buttons
  function setButtonLoading(button, isLoading, loadingText = 'Please wait...') {
    if (!button) return;
    if (isLoading) {
      button.dataset.originalText = button.innerHTML;
      button.disabled = true;
      button.classList.add('btn-loading');
      button.innerHTML = `
        <span class="btn-spinner" aria-hidden="true"></span>
        <span>${escapeHTML(loadingText)}</span>
      `;
    } else {
      button.disabled = false;
      button.classList.remove('btn-loading');
      if (button.dataset.originalText) {
        button.innerHTML = button.dataset.originalText;
      }
    }
  }

  // Setup password show/hide buttons
  function setupPasswordToggles() {
    document.querySelectorAll('.password-toggle-btn').forEach(btn => {
      btn.addEventListener('click', () => {
        const input = btn.closest('.password-input-wrap')?.querySelector('input');
        if (!input) return;
        const isPassword = input.getAttribute('type') === 'password';
        input.setAttribute('type', isPassword ? 'text' : 'password');
        btn.setAttribute('aria-label', isPassword ? 'Hide password' : 'Show password');
        btn.innerHTML = isPassword 
          ? `<svg fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l18 18"/></svg>`
          : `<svg fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/></svg>`;
      });
    });
  }

  // Initialize common UI behaviors
  document.addEventListener('DOMContentLoaded', () => {
    setupPasswordToggles();
  });

  return {
    setTenantAuth,
    getAccessToken,
    getRefreshToken,
    getUser,
    isAuthenticated,
    clearTenantAuth,
    setPlatformAuth,
    getPlatformToken,
    isPlatformAuthenticated,
    clearPlatformAuth,
    getPlatformRefreshToken,
    logout,
    platformLogout,
    showAlert,
    clearAlert,
    setButtonLoading,
    setupPasswordToggles,
    isTokenExpired,
    refreshToken,
    ensureSession,
    handleAuthFailure
  };
})();

window.Auth = Auth;
