/**
 * Vyavsa OTP Verification Handler
 * Handles 6-digit verification code input, auto-focus, paste, countdown timer, and resend
 */

document.addEventListener('DOMContentLoaded', () => {
  const form = document.getElementById('otp-form');
  const alertContainer = document.getElementById('alert-container');
  const submitBtn = document.getElementById('verify-btn');
  const resendBtn = document.getElementById('resend-btn');
  const countdownText = document.getElementById('countdown-text');
  const emailDisplay = document.getElementById('email-display');
  const digitInputs = document.querySelectorAll('.otp-digit');

  const urlParams = new URLSearchParams(window.location.search);
  const email = urlParams.get('email') || '';
  const flowType = urlParams.get('type') || 'password_reset';

  if (emailDisplay) {
    emailDisplay.textContent = email || 'your registered email';
  }

  // Setup auto-focus, paste, and navigation for the 6 digit inputs
  if (digitInputs.length > 0) {
    // Focus first input on page load
    setTimeout(() => {
      digitInputs[0].focus();
    }, 100);

    digitInputs.forEach((input, index) => {
      // Input event
      input.addEventListener('input', (e) => {
        const val = e.target.value;

        // Ensure single digit
        if (val.length > 1) {
          e.target.value = val.slice(-1);
        }

        // Clean non-numeric
        if (!/^\d$/.test(e.target.value)) {
          e.target.value = '';
          e.target.classList.remove('has-value');
          return;
        }

        e.target.classList.add('has-value');
        e.target.classList.remove('is-invalid');

        // Auto advance to next input
        if (index < digitInputs.length - 1) {
          digitInputs[index + 1].focus();
          digitInputs[index + 1].select();
        }

        // If all 6 filled, check if ready to submit
        checkFullCode();
      });

      // Keydown navigation (Backspace, arrows)
      input.addEventListener('keydown', (e) => {
        if (e.key === 'Backspace') {
          if (!input.value && index > 0) {
            digitInputs[index - 1].focus();
            digitInputs[index - 1].value = '';
            digitInputs[index - 1].classList.remove('has-value');
          } else {
            input.value = '';
            input.classList.remove('has-value');
          }
        } else if (e.key === 'ArrowLeft' && index > 0) {
          e.preventDefault();
          digitInputs[index - 1].focus();
        } else if (e.key === 'ArrowRight' && index < digitInputs.length - 1) {
          e.preventDefault();
          digitInputs[index + 1].focus();
        }
      });

      // Paste handling
      input.addEventListener('paste', (e) => {
        e.preventDefault();
        const pastedData = (e.clipboardData || window.clipboardData).getData('text');
        const digits = pastedData.replace(/\D/g, '').slice(0, digitInputs.length);

        if (digits.length > 0) {
          digits.split('').forEach((d, i) => {
            if (digitInputs[i]) {
              digitInputs[i].value = d;
              digitInputs[i].classList.add('has-value');
              digitInputs[i].classList.remove('is-invalid');
            }
          });

          const nextIndex = Math.min(digits.length, digitInputs.length - 1);
          digitInputs[nextIndex].focus();

          if (digits.length === digitInputs.length) {
            checkFullCode();
          }
        }
      });
    });
  }

  function getOTPValue() {
    return Array.from(digitInputs).map(i => i.value.trim()).join('');
  }

  function checkFullCode() {
    const code = getOTPValue();
    if (code.length === digitInputs.length && form) {
      // Auto trigger form submit when 6 digits entered
      form.requestSubmit();
    }
  }

  // 60-Second Countdown Timer for Resend
  let timerSeconds = 60;
  let timerInterval = null;

  function startCountdown() {
    timerSeconds = 60;
    if (resendBtn) {
      resendBtn.disabled = true;
    }
    updateTimerDisplay();

    if (timerInterval) clearInterval(timerInterval);

    timerInterval = setInterval(() => {
      timerSeconds--;
      updateTimerDisplay();

      if (timerSeconds <= 0) {
        clearInterval(timerInterval);
        if (resendBtn) {
          resendBtn.disabled = false;
        }
        if (countdownText) {
          countdownText.textContent = 'Didn\'t receive code?';
        }
      }
    }, 1000);
  }

  function updateTimerDisplay() {
    if (countdownText && timerSeconds > 0) {
      countdownText.textContent = `Resend available in ${timerSeconds}s`;
    }
  }

  // Start timer on page load
  startCountdown();

  // Resend OTP button handler
  if (resendBtn) {
    resendBtn.addEventListener('click', async () => {
      if (resendBtn.disabled) return;
      if (!email) {
        window.Auth.showAlert(alertContainer, 'Missing email address for OTP resend.');
        return;
      }

      window.Auth.clearAlert(alertContainer);
      resendBtn.disabled = true;

      try {
        const response = await window.API.post('/auth/forgot-password', { email }, { skipAuth: true });
        if (response.ok) {
          window.Auth.showAlert(alertContainer, 'A new verification code has been dispatched to your email.', 'success');
          // Clear previous input
          digitInputs.forEach(input => {
            input.value = '';
            input.classList.remove('has-value', 'is-invalid');
          });
          digitInputs[0].focus();
          startCountdown();
        } else {
          window.Auth.showAlert(alertContainer, response.error || 'Failed to resend code. Please try again.');
          resendBtn.disabled = false;
        }
      } catch (err) {
        window.Auth.showAlert(alertContainer, 'Unable to connect to server to resend code.');
        resendBtn.disabled = false;
      }
    });
  }

  // Verify form submission
  if (form) {
    form.addEventListener('submit', async (e) => {
      e.preventDefault();
      window.Auth.clearAlert(alertContainer);

      const otp = getOTPValue();

      if (otp.length !== 6) {
        window.Auth.showAlert(alertContainer, 'Please enter all 6 digits of the verification code.');
        return;
      }

      if (!email) {
        window.Auth.showAlert(alertContainer, 'Email context is missing. Please restart the recovery flow.');
        return;
      }

      window.Auth.setButtonLoading(submitBtn, true, 'Verifying...');

      try {
        // Backend endpoint: POST /api/v1/auth/verify-reset-otp
        const response = await window.API.post('/auth/verify-reset-otp', {
          email: email,
          otp: otp
        }, { skipAuth: true });

        if (!response.ok) {
          window.Auth.setButtonLoading(submitBtn, false);
          digitInputs.forEach(input => input.classList.add('is-invalid'));
          window.Auth.showAlert(alertContainer, response.error || 'Invalid or expired verification code.');
          return;
        }

        // Success: Extract reset token
        const resetToken = response.data && response.data.reset_token;
        if (resetToken) {
          sessionStorage.setItem('vyavsa_reset_token', resetToken);
        }

        submitBtn.classList.remove('btn-loading');
        submitBtn.innerHTML = 'Verified! Redirecting...';
        submitBtn.classList.add('btn-success');

        setTimeout(() => {
          if (flowType === 'password_reset') {
            window.location.href = `/reset-password?email=${encodeURIComponent(email)}`;
          } else {
            window.location.href = '/dashboard';
          }
        }, 500);

      } catch (err) {
        window.Auth.setButtonLoading(submitBtn, false);
        window.Auth.showAlert(alertContainer, 'A network error occurred while verifying the code.');
      }
    });
  }
});
