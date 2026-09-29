/**
 * Vyavsa Platform Admin Login Handler
 */

document.addEventListener('DOMContentLoaded', () => {
  const form = document.getElementById('platform-login-form');
  const alertContainer = document.getElementById('alert-container');
  const submitBtn = document.getElementById('submit-btn');
  const emailInput = document.getElementById('email');
  const passwordInput = document.getElementById('password');

  // Check if platform admin already authenticated
  if (window.Auth && window.Auth.isPlatformAuthenticated()) {
    window.location.href = '/dashboard';
    return;
  }

  if (!form) return;

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    window.Auth.clearAlert(alertContainer);

    const email = emailInput.value.trim();
    const password = passwordInput.value;

    if (!email) {
      window.Auth.showAlert(alertContainer, 'Please enter administrator email.');
      emailInput.focus();
      return;
    }

    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(email)) {
      window.Auth.showAlert(alertContainer, 'Please enter a valid administrator email address.');
      emailInput.focus();
      return;
    }

    if (!password) {
      window.Auth.showAlert(alertContainer, 'Please enter password.');
      passwordInput.focus();
      return;
    }

    window.Auth.setButtonLoading(submitBtn, true, 'Authenticating...');

    try {
      const response = await window.API.post('/auth/platform/login', {
        email,
        password
      }, { skipAuth: true, isPlatform: true });

      if (!response.ok) {
        window.Auth.setButtonLoading(submitBtn, false);
        window.Auth.showAlert(alertContainer, response.error || 'Invalid administrator credentials.');
        passwordInput.value = '';
        passwordInput.focus();
        return;
      }

      // Store platform auth
      window.Auth.setPlatformAuth(response.data);

      submitBtn.classList.remove('btn-loading');
      submitBtn.innerHTML = 'Authorized! Redirecting...';
      submitBtn.classList.add('btn-success');

      setTimeout(() => {
        window.location.href = '/dashboard';
      }, 500);

    } catch (err) {
      window.Auth.setButtonLoading(submitBtn, false);
      window.Auth.showAlert(alertContainer, 'An unexpected connection error occurred.');
    }
  });
});
