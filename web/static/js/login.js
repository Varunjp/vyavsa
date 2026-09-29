/**
 * Vyavsa Tenant Login Handler
 */

document.addEventListener('DOMContentLoaded', () => {
  const form = document.getElementById('tenant-login-form');
  const alertContainer = document.getElementById('alert-container');
  const submitBtn = document.getElementById('submit-btn');
  const emailInput = document.getElementById('email');
  const passwordInput = document.getElementById('password');

  // If already authenticated as tenant, redirect to dashboard
  if (window.Auth && window.Auth.isAuthenticated()) {
    window.location.href = '/dashboard';
    return;
  }

  // Pre-fill email from query param if available
  const urlParams = new URLSearchParams(window.location.search);
  const emailParam = urlParams.get('email');
  if (emailParam && emailInput) {
    emailInput.value = emailParam;
  }

  // Check for registration success message
  if (urlParams.get('registered') === 'true') {
    window.Auth.showAlert(alertContainer, 'Account created successfully! Please sign in with your credentials.', 'success');
  } else if (urlParams.get('reset') === 'true') {
    window.Auth.showAlert(alertContainer, 'Password reset successfully! Please sign in with your new password.', 'success');
  }

  if (!form) return;

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    window.Auth.clearAlert(alertContainer);

    const email = emailInput.value.trim();
    const password = passwordInput.value;

    // Client-side validations
    if (!email) {
      window.Auth.showAlert(alertContainer, 'Please enter your email address.');
      emailInput.focus();
      return;
    }

    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(email)) {
      window.Auth.showAlert(alertContainer, 'Please enter a valid email address.');
      emailInput.focus();
      return;
    }

    if (!password) {
      window.Auth.showAlert(alertContainer, 'Please enter your password.');
      passwordInput.focus();
      return;
    }

    // Set loading state
    window.Auth.setButtonLoading(submitBtn, true, 'Signing in...');

    try {
      const response = await window.API.post('/auth/tenant/login', {
        email,
        password
      }, { skipAuth: true });

      if (!response.ok) {
        window.Auth.setButtonLoading(submitBtn, false);
        window.Auth.showAlert(alertContainer, response.error || 'Invalid email or password.');
        passwordInput.value = '';
        passwordInput.focus();
        return;
      }

      // Successful authentication
      window.Auth.setTenantAuth(response.data);
      
      // Visual success state before redirect
      submitBtn.classList.remove('btn-loading');
      submitBtn.innerHTML = 'Success! Redirecting...';
      submitBtn.classList.add('btn-success');

      setTimeout(() => {
        const redirectUrl = urlParams.get('redirect') || '/dashboard';
        window.location.href = redirectUrl;
      }, 500);

    } catch (err) {
      window.Auth.setButtonLoading(submitBtn, false);
      window.Auth.showAlert(alertContainer, 'An unexpected error occurred. Please try again.');
    }
  });
});
