/**
 * Vyavsa Tenant Registration Handler
 */

document.addEventListener('DOMContentLoaded', () => {
  const form = document.getElementById('register-form');
  const alertContainer = document.getElementById('alert-container');
  const submitBtn = document.getElementById('submit-btn');
  const planContainer = document.getElementById('plan-select-container');
  const passwordInput = document.getElementById('password');
  const confirmPasswordInput = document.getElementById('confirm-password');
  const strengthFeedback = document.getElementById('strength-text');
  const strengthSegments = document.querySelectorAll('.strength-bar-seg');

  let selectedPlanId = null;

  // Load plans from API dynamically
  async function loadPlans() {
    if (!planContainer) return;

    try {
      const response = await window.API.get('/plans', { skipAuth: true });
      if (response.ok && Array.isArray(response.data) && response.data.length > 0) {
        renderPlans(response.data);
      } else {
        renderFallbackPlan();
      }
    } catch (e) {
      renderFallbackPlan();
    }
  }

  function renderPlans(plans) {
    planContainer.innerHTML = '';
    const grid = document.createElement('div');
    grid.className = 'plan-select-grid';

    plans.forEach((plan, idx) => {
      const isFirst = idx === 0;
      if (isFirst) selectedPlanId = plan.id;

      const card = document.createElement('label');
      card.className = `plan-option ${isFirst ? 'selected' : ''}`;
      card.dataset.planId = plan.id;

      const priceVal = parseFloat(plan.price) || 0;
      const formattedPrice = priceVal === 0 ? 'Free' : `₹${priceVal}/mo`;

      card.innerHTML = `
        <input type="radio" name="plan_selection" value="${escapeAttr(plan.id)}" ${isFirst ? 'checked' : ''}>
        <span class="plan-option-name">${escapeText(plan.plan_name || plan.name || 'Standard')}</span>
        <span class="plan-option-price">${formattedPrice}</span>
        <span class="plan-option-desc">${escapeText(plan.note || plan.description || 'Full basic access')}</span>
      `;

      card.addEventListener('click', () => {
        grid.querySelectorAll('.plan-option').forEach(el => el.classList.remove('selected'));
        card.classList.add('selected');
        selectedPlanId = plan.id;
      });

      grid.appendChild(card);
    });

    planContainer.appendChild(grid);
  }

  function renderFallbackPlan() {
    if (!planContainer) return;
    planContainer.innerHTML = `
      <div class="plan-select-grid">
        <label class="plan-option selected">
          <input type="radio" name="plan_selection" value="" checked>
          <span class="plan-option-name">Standard Business</span>
          <span class="plan-option-price">Free Tier</span>
          <span class="plan-option-desc">Everything needed to get started</span>
        </label>
      </div>
    `;
    selectedPlanId = null;
  }

  function escapeText(str) {
    if (!str) return '';
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
  }

  function escapeAttr(str) {
    if (!str) return '';
    return str.replace(/"/g, '&quot;');
  }

  // Password strength calculator
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

  if (passwordInput) {
    passwordInput.addEventListener('input', (e) => {
      updateStrengthMeter(e.target.value);
    });
  }

  // Form submission
  if (form) {
    form.addEventListener('submit', async (e) => {
      e.preventDefault();
      window.Auth.clearAlert(alertContainer);

      const businessName = document.getElementById('name').value.trim();
      const ownerName = document.getElementById('admin_name').value.trim();
      const email = document.getElementById('email').value.trim();
      const phone = document.getElementById('phone') ? document.getElementById('phone').value.trim() : '';
      const password = passwordInput.value;
      const confirmPassword = confirmPasswordInput.value;

      // Validation
      if (!businessName || businessName.length < 2) {
        window.Auth.showAlert(alertContainer, 'Please enter a valid business or organization name (at least 2 characters).');
        document.getElementById('name').focus();
        return;
      }

      const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
      if (!email || !emailRegex.test(email)) {
        window.Auth.showAlert(alertContainer, 'Please enter a valid email address.');
        document.getElementById('email').focus();
        return;
      }

      if (!password || password.length < 6) {
        window.Auth.showAlert(alertContainer, 'Password must be at least 6 characters long.');
        passwordInput.focus();
        return;
      }

      if (password !== confirmPassword) {
        window.Auth.showAlert(alertContainer, 'Passwords do not match. Please verify your password confirmation.');
        confirmPasswordInput.focus();
        return;
      }

      // Build payload matching dto.TenantRegisterRequest
      const payload = {
        name: businessName,
        email: email,
        password: password
      };

      if (ownerName) payload.admin_name = ownerName;
      if (phone) payload.phone = phone;
      if (selectedPlanId) payload.plan_id = selectedPlanId;

      window.Auth.setButtonLoading(submitBtn, true, 'Creating your business account...');

      try {
        const response = await window.API.post('/auth/tenant/register', payload, { skipAuth: true });

        if (!response.ok) {
          window.Auth.setButtonLoading(submitBtn, false);
          let errMsg = response.error || 'Failed to create business account. Please try again.';
          if (response.details && typeof response.details === 'object') {
            const detailMsgs = Object.values(response.details).join(', ');
            if (detailMsgs) errMsg = `${errMsg}: ${detailMsgs}`;
          }
          window.Auth.showAlert(alertContainer, errMsg);
          return;
        }

        // Check if authentication tokens were provided in response
        if (response.data && response.data.tokens) {
          window.Auth.setTenantAuth(response.data.tokens);
          submitBtn.classList.remove('btn-loading');
          submitBtn.innerHTML = 'Account created! Redirecting...';
          submitBtn.classList.add('btn-success');
          setTimeout(() => {
            window.location.href = '/dashboard';
          }, 600);
        } else {
          // Redirect to login with success flag
          window.location.href = `/login?registered=true&email=${encodeURIComponent(email)}`;
        }

      } catch (err) {
        window.Auth.setButtonLoading(submitBtn, false);
        window.Auth.showAlert(alertContainer, 'A network error occurred. Please check your connection and try again.');
      }
    });
  }

  // Load plans on initialization
  loadPlans();
});
