/**
 * Vyavsa Landing Page Interactions
 * Mobile navigation, dynamic pricing plans loader, and auth state adaptation
 */

document.addEventListener('DOMContentLoaded', () => {
  // 1. Mobile Menu Toggle
  const menuToggle = document.getElementById('mobile-menu-toggle');
  const navMenu = document.getElementById('nav-menu');

  if (menuToggle && navMenu) {
    menuToggle.addEventListener('click', () => {
      const isOpen = navMenu.classList.toggle('open');
      menuToggle.setAttribute('aria-expanded', isOpen);
      menuToggle.innerHTML = isOpen
        ? `<svg fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>`
        : `<svg fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"/></svg>`;
    });

    // Close menu when clicking navigation link
    navMenu.querySelectorAll('.nav-link').forEach(link => {
      link.addEventListener('click', () => {
        navMenu.classList.remove('open');
        menuToggle.setAttribute('aria-expanded', 'false');
      });
    });
  }

  // 2. Dynamic Nav Links based on Auth State
  if (window.Auth && window.Auth.isAuthenticated()) {
    const loginNavLinks = document.querySelectorAll('a[href="/login"]');
    loginNavLinks.forEach(link => {
      link.textContent = 'Dashboard';
      link.href = '/dashboard';
    });

    const getStartedBtns = document.querySelectorAll('.hero-actions .btn-primary, .cta-actions .btn-primary');
    getStartedBtns.forEach(btn => {
      btn.textContent = 'Go to Dashboard';
      btn.href = '/dashboard';
    });
  }

  // 3. Dynamic Pricing Plans from Backend API
  const pricingContainer = document.getElementById('pricing-grid');
  if (pricingContainer) {
    loadPricingPlans(pricingContainer);
  }

  async function loadPricingPlans(container) {
    try {
      const response = await window.API.get('/plans', { skipAuth: true });
      if (response.ok && Array.isArray(response.data) && response.data.length > 0) {
        renderDynamicPlans(container, response.data);
      }
    } catch (e) {
      // Keep static graceful fallback cards
    }
  }

  function renderDynamicPlans(container, plans) {
    container.innerHTML = '';

    plans.forEach((plan, idx) => {
      const card = document.createElement('div');
      const priceVal = parseFloat(plan.price) || 0;
      const isPopular = priceVal > 0 || idx === 1 || plan.is_popular;
      card.className = `pricing-card ${isPopular ? 'pricing-card-popular' : ''}`;

      const displayPrice = priceVal === 0 ? 'Free' : `₹${priceVal}`;
      const period = priceVal === 0 ? 'for 1 month' : '/month';
      const ctaText = priceVal === 0 ? 'Start Free Trial' : 'Choose Monthly Plan';
      const badgeText = isPopular ? 'Standard Plan' : '';

      card.innerHTML = `
        ${badgeText ? `<div class="pricing-badge">${badgeText}</div>` : ''}
        <h3 class="pricing-card-title">${escapeHTML(plan.plan_name || plan.name || 'Business Plan')}</h3>
        <p class="pricing-card-desc">${escapeHTML(plan.note || plan.description || 'Full access to all business features.')}</p>
        <div class="pricing-card-price">
          <span class="price-val">${displayPrice}</span>
          <span class="price-period">${period}</span>
        </div>
        <ul class="pricing-features-list">
          <li class="pricing-feature-item">
            <svg class="check-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
            Full access to all features
          </li>
          <li class="pricing-feature-item">
            <svg class="check-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
            Unlimited transaction bookkeeping
          </li>
          <li class="pricing-feature-item">
            <svg class="check-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
            Customer dues & receivables tracking
          </li>
          <li class="pricing-feature-item">
            <svg class="check-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
            Employee records & salary management
          </li>
          <li class="pricing-feature-item">
            <svg class="check-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
            Real-time financial status & reports
          </li>
          ${isPopular ? `
          <li class="pricing-feature-item">
            <svg class="check-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
            Priority support & automatic data backup
          </li>` : ''}
        </ul>
        <a href="/register" class="btn ${isPopular ? 'btn-primary' : 'btn-outline'} btn-block">${ctaText}</a>
      `;

      container.appendChild(card);
    });
  }

  function escapeHTML(str) {
    if (!str) return '';
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
  }
});
