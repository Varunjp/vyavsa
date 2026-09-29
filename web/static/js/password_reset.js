/**
 * Vyavsa Password Reset Flow Handlers
 * Handles /forgot-password and /reset-password logic
 */

document.addEventListener('DOMContentLoaded', () => {
  // ----------------------------------------------------
  // 1. Forgot Password Form Handler
  // ----------------------------------------------------
  const forgotForm = document.getElementById('forgot-password-form');
  if (forgotForm) {
    const alertContainer = document.getElementById('alert-container');
    const submitBtn = document.getElementById('submit-btn');
    const emailInput = document.getElementById('email');

    forgotForm.addEventListener('submit', async (e) => {
      e.preventDefault();
      window.Auth.clearAlert(alertContainer);

      const email = emailInput.value.trim();
      const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

      if (!email || !emailRegex.test(email)) {
        window.Auth.showAlert(alertContainer, 'Please enter a valid registered email address.');
        emailInput.focus();
        return;
      }

      window.Auth.setButtonLoading(submitBtn, true, 'Sending verification code...');

      try {
        const response = await window.API.post('/auth/forgot-password', { email }, { skipAuth: true });

        // Whether email exists or not, backend returns generic message to avoid enumeration
        submitBtn.classList.remove('btn-loading');
        submitBtn.innerHTML = 'Code Sent! Redirecting...';
        submitBtn.classList.add('btn-success');

        setTimeout(() => {
          window.location.href = `/verify-otp?email=${encodeURIComponent(email)}&type=password_reset`;
        }, 600);

      } catch (err) {
        window.Auth.setButtonLoading(submitBtn, false);
        window.Auth.showAlert(alertContainer, 'Unable to connect to the server. Please try again.');
      }
    });
  }

  // ----------------------------------------------------
  // 2. Set New Password Form Handler
  // ----------------------------------------------------
  const resetForm = document.getElementById('reset-password-form');
  if (resetForm) {
    const alertContainer = document.getElementById('alert-container');
    const submitBtn = document.getElementById('submit-btn');
    const newPasswordInput = document.getElementById('new-password');
    const confirmPasswordInput = document.getElementById('confirm-password');
    const successCard = document.getElementById('reset-success-card');
    const strengthFeedback = document.getElementById('strength-text');
    const strengthSegments = document.querySelectorAll('.strength-bar-seg');

    // Retrieve reset token from storage or query param
    const urlParams = new URLSearchParams(window.location.search);
    const resetToken = sessionStorage.getItem('vyavsa_reset_token') || urlParams.get('token');

    if (!resetToken) {
      window.Auth.showAlert(
        alertContainer, 
        'Reset session is invalid or has expired. Please initiate the password recovery process again.'
      );
      if (submitBtn) submitBtn.disabled = true;
      if (newPasswordInput) newPasswordInput.disabled = true;
      if (confirmPasswordInput) confirmPasswordInput.disabled = true;
    }

    // Password strength logic
    function evaluatePasswordStrength(pwd) {
      if (!pwd) return 0;
      let score = 0;
      if (pwd.length >= 6) score++;
      if (pwd.length >= 8) score++;
      if (/[A-Z]/.test(pwd) && /[a-z]/.test(pwd)) score++;
      if (/\d/.test(pwd) && /[^A-Za-z0-9]/.test(pwd)) score++;
      return score;
    }

    function updateStrengthMeter(pwd) {
      if (!strengthSegments || strengthSegments.length === 0) return;
      const score = evaluatePasswordStrength(pwd);

      strengthSegments.forEach(seg => {
        seg.className = 'strength-bar-seg';
      });

      const labels = ['Weak', 'Fair', 'Good', 'Strong'];
      const classes = ['active-weak', 'active-fair', 'active-good', 'active-strong'];

      if (score > 0) {
        for (let i = 0; i < score; i++) {
          strengthSegments[i].classList.add(classes[score - 1]);
        }
        if (strengthFeedback) {
          strengthFeedback.textContent = labels[score - 1];
          strengthFeedback.style.color = score <= 1 ? 'var(--color-error)' : score === 2 ? 'var(--color-warning)' : 'var(--color-success)';
        }
      } else {
        if (strengthFeedback) {
          strengthFeedback.textContent = 'Minimum 6 characters';
          strengthFeedback.style.color = 'var(--color-text-muted)';
        }
      }
    }

    if (newPasswordInput) {
      newPasswordInput.addEventListener('input', (e) => {
        updateStrengthMeter(e.target.value);
      });
    }

    resetForm.addEventListener('submit', async (e) => {
      e.preventDefault();
      window.Auth.clearAlert(alertContainer);

      if (!resetToken) {
        window.Auth.showAlert(alertContainer, 'Session expired. Please request a new verification code.');
        return;
      }

      const newPassword = newPasswordInput.value;
      const confirmPassword = confirmPasswordInput.value;

      if (!newPassword || newPassword.length < 6) {
        window.Auth.showAlert(alertContainer, 'Password must be at least 6 characters long.');
        newPasswordInput.focus();
        return;
      }

      if (newPassword !== confirmPassword) {
        window.Auth.showAlert(alertContainer, 'Passwords do not match. Please re-enter your password.');
        confirmPasswordInput.focus();
        return;
      }

      window.Auth.setButtonLoading(submitBtn, true, 'Updating password...');

      try {
        const response = await window.API.post('/auth/reset-password', {
          reset_token: resetToken,
          new_password: newPassword
        }, { skipAuth: true });

        if (!response.ok) {
          window.Auth.setButtonLoading(submitBtn, false);
          window.Auth.showAlert(alertContainer, response.error || 'Failed to reset password. The link or token may have expired.');
          return;
        }

        // Clean up token from storage
        sessionStorage.removeItem('vyavsa_reset_token');

        // Display success card
        if (resetForm && successCard) {
          resetForm.style.display = 'none';
          successCard.style.display = 'block';
        }

      } catch (err) {
        window.Auth.setButtonLoading(submitBtn, false);
        window.Auth.showAlert(alertContainer, 'A network error occurred while updating your password.');
      }
    });
  }
});
