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

  function isAuthenticated() {
    return !!getAccessToken();
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
    return !!getPlatformToken();
  }

  function clearPlatformAuth() {
    safeStorageRemove(STORAGE_KEYS.PLATFORM_ACCESS_TOKEN);
    safeStorageRemove(STORAGE_KEYS.PLATFORM_REFRESH_TOKEN);
    safeStorageRemove(STORAGE_KEYS.PLATFORM_USER);
    try { sessionStorage.clear(); } catch(e) {}
  }

  // Session Invalidation
  async function logout() {
    try {
      if (window.API && isAuthenticated()) {
        await window.API.post('/auth/logout', {});
      }
    } catch (e) {
      // Proceed with local cleanup regardless
    } finally {
      clearTenantAuth();
      clearPlatformAuth();
      window.location.replace('/login');
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
    logout,
    showAlert,
    clearAlert,
    setButtonLoading,
    setupPasswordToggles
  };
})();

window.Auth = Auth;
