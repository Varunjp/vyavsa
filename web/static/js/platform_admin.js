/**
 * Vyavsa Platform Admin Portal Manager
 * Handles routing, telemetry, tenant management, subscription plans, and billing
 */

(function () {
  'use strict';

  // State Management
  const state = {
    currentView: 'dashboard',
    tenants: {
      page: 1,
      pageSize: 15,
      status: '',
      search: '',
      total: 0
    },
    plans: {
      page: 1,
      pageSize: 50,
      status: 'active',
      cached: []
    },
    subscriptions: {
      page: 1,
      pageSize: 15,
      status: '',
      total: 0
    },
    transactions: {
      page: 1,
      pageSize: 15,
      status: '',
      total: 0
    }
  };

  // Utility: Currency Formatter
  function formatCurrency(amount) {
    if (amount === undefined || amount === null || isNaN(amount)) return '₹0.00';
    const num = parseFloat(amount);
    return new Intl.NumberFormat('en-IN', {
      style: 'currency',
      currency: 'INR',
      minimumFractionDigits: 2,
      maximumFractionDigits: 2
    }).format(num);
  }

  // Utility: Date Formatter
  function formatDate(isoStr) {
    if (!isoStr) return '—';
    try {
      const d = new Date(isoStr);
      if (isNaN(d.getTime())) return '—';
      return d.toLocaleDateString('en-IN', {
        year: 'numeric',
        month: 'short',
        day: 'numeric'
      });
    } catch (e) {
      return '—';
    }
  }

  function formatDateTime(isoStr) {
    if (!isoStr) return '—';
    try {
      const d = new Date(isoStr);
      if (isNaN(d.getTime())) return '—';
      return d.toLocaleString('en-IN', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
      });
    } catch (e) {
      return '—';
    }
  }

  // Utility: Toast Notification
  function showToast(message, type = 'info') {
    const container = document.getElementById('platform-toast-container');
    if (!container) return;

    const toast = document.createElement('div');
    toast.className = `platform-toast toast-${type}`;
    
    let iconSvg = '';
    if (type === 'success') {
      iconSvg = '<svg width="18" height="18" fill="none" viewBox="0 0 24 24" stroke="#10b981"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>';
    } else if (type === 'error') {
      iconSvg = '<svg width="18" height="18" fill="none" viewBox="0 0 24 24" stroke="#ef4444"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>';
    } else {
      iconSvg = '<svg width="18" height="18" fill="none" viewBox="0 0 24 24" stroke="#4f46e5"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>';
    }

    toast.innerHTML = `
      <div style="flex-shrink: 0;">${iconSvg}</div>
      <div style="flex: 1; word-break: break-word;">${escapeHTML(message)}</div>
    `;

    container.appendChild(toast);
    // Trigger animation
    requestAnimationFrame(() => toast.classList.add('show'));

    setTimeout(() => {
      toast.classList.remove('show');
      setTimeout(() => toast.remove(), 250);
    }, 3500);
  }

  function escapeHTML(str) {
    if (!str) return '';
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
  }

  // Modals Management
  function openModal(modalId) {
    const modal = document.getElementById(modalId);
    if (!modal) return;
    modal.classList.add('active');
    document.body.style.overflow = 'hidden';
  }

  function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    if (!modal) return;
    modal.classList.remove('active');
    if (!document.querySelector('.modal.active')) {
      document.body.style.overflow = '';
    }
  }

  function closeAllModals() {
    document.querySelectorAll('.modal.active').forEach(m => m.classList.remove('active'));
    document.body.style.overflow = '';
  }

  function confirmDialog(title, message, onConfirm, confirmBtnText = 'Confirm', isDestructive = false) {
    const modal = document.getElementById('confirm-action-modal');
    const titleEl = document.getElementById('confirm-modal-title');
    const msgEl = document.getElementById('confirm-modal-message');
    const actionBtn = document.getElementById('confirm-action-btn');

    if (titleEl) titleEl.textContent = title;
    if (msgEl) msgEl.textContent = message;
    if (actionBtn) {
      actionBtn.textContent = confirmBtnText;
      actionBtn.className = isDestructive ? 'btn btn-danger' : 'btn btn-primary';

      // Replace click handler cleanly
      const newBtn = actionBtn.cloneNode(true);
      actionBtn.parentNode.replaceChild(newBtn, actionBtn);
      newBtn.addEventListener('click', async () => {
        closeModal('confirm-action-modal');
        if (typeof onConfirm === 'function') await onConfirm();
      });
    }
    openModal('confirm-action-modal');
  }

  // View Navigation Router
  function navigateTo(path, updateHistory = true) {
    // Close any open modals on route transition
    closeAllModals();
    window.scrollTo(0, 0);

    // Determine view and potential parameters
    let view = 'dashboard';
    let tenantIdParam = null;

    if (path.startsWith('/platform/tenants/')) {
      view = 'tenants';
      tenantIdParam = path.replace('/platform/tenants/', '').trim();
    } else if (path.startsWith('/platform/tenants')) {
      view = 'tenants';
    } else if (path.startsWith('/platform/plans')) {
      view = 'plans';
    } else if (path.startsWith('/platform/subscriptions')) {
      view = 'subscriptions';
    } else if (path.startsWith('/platform/transactions')) {
      view = 'transactions';
    } else if (path.startsWith('/platform/metrics')) {
      view = 'metrics';
    } else {
      view = 'dashboard';
      path = '/platform/dashboard';
    }

    state.currentView = view;

    // Update Browser History
    if (updateHistory && window.location.pathname !== path) {
      window.history.pushState({ path }, '', path);
    }

    // Toggle active view container
    document.querySelectorAll('.platform-view').forEach(el => el.classList.remove('active'));
    const targetViewEl = document.getElementById(`view-${view}`);
    if (targetViewEl) targetViewEl.classList.add('active');

    // Toggle active sidebar links
    document.querySelectorAll('.sidebar-link').forEach(link => {
      const linkView = link.getAttribute('data-view');
      if (linkView === view) {
        link.classList.add('active');
      } else {
        link.classList.remove('active');
      }
    });

    // Close mobile drawer if open
    closeMobileSidebar();

    // Trigger View Data Loading
    if (view === 'dashboard') {
      loadDashboardMetrics();
    } else if (view === 'tenants') {
      loadTenants();
      if (tenantIdParam) {
        showTenantDetails(tenantIdParam);
      }
    } else if (view === 'plans') {
      loadPlans();
    } else if (view === 'subscriptions') {
      loadSubscriptions();
    } else if (view === 'transactions') {
      loadTransactions();
    } else if (view === 'metrics') {
      setupMetricsView();
    }
  }

  function openMobileSidebar() {
    const sidebar = document.getElementById('platform-sidebar');
    const backdrop = document.getElementById('platform-sidebar-backdrop');
    if (sidebar) {
      sidebar.classList.add('open', 'active');
    }
    if (backdrop) {
      backdrop.classList.add('active', 'open');
    }
    document.body.style.overflow = 'hidden';
  }

  function closeMobileSidebar() {
    const sidebar = document.getElementById('platform-sidebar');
    const backdrop = document.getElementById('platform-sidebar-backdrop');
    if (sidebar) {
      sidebar.classList.remove('open', 'active');
    }
    if (backdrop) {
      backdrop.classList.remove('active', 'open');
    }
    if (!document.querySelector('.modal.active')) {
      document.body.style.overflow = '';
    }
  }

  function toggleMobileSidebar() {
    const sidebar = document.getElementById('platform-sidebar');
    if (sidebar && (sidebar.classList.contains('open') || sidebar.classList.contains('active'))) {
      closeMobileSidebar();
    } else {
      openMobileSidebar();
    }
  }

  // =========================================================================
  // View 1: Dashboard Telemetry & SVG Chart
  // =========================================================================
  async function loadDashboardMetrics() {
    const activeEl = document.getElementById('metric-active-tenants');
    const totalEl = document.getElementById('metric-total-tenants-sub');
    const incomeEl = document.getElementById('metric-monthly-income');
    const regEl = document.getElementById('metric-recent-registrations');
    const chartWrapper = document.getElementById('revenue-chart-wrapper');

    if (activeEl) activeEl.innerHTML = '<span class="btn-spinner" style="width: 1.25rem; height: 1.25rem;"></span>';

    try {
      const res = await window.API.get('/platform/dashboard/metrics');
      if (!res.ok) {
        showToast(res.error || 'Failed to load telemetry metrics', 'error');
        if (activeEl) activeEl.textContent = '—';
        if (incomeEl) incomeEl.textContent = '—';
        if (regEl) regEl.textContent = '—';
        if (chartWrapper) {
          chartWrapper.innerHTML = `
            <div class="table-empty-box" style="height: 100%; display: flex; flex-direction: column; align-items: center; justify-content: center;">
              <div style="color: var(--color-error); font-weight: 600; margin-bottom: 0.75rem;">${escapeHTML(res.error || 'Failed to load telemetry metrics')}</div>
              <button type="button" class="btn btn-secondary btn-sm" id="retry-metrics-btn">Retry Telemetry</button>
            </div>
          `;
          const rBtn = document.getElementById('retry-metrics-btn');
          if (rBtn) rBtn.addEventListener('click', () => loadDashboardMetrics());
        }
        return;
      }

      const data = res.data || {};
      if (activeEl) activeEl.textContent = data.active_tenants || '0';
      if (totalEl) {
        totalEl.innerHTML = `<span>of <strong>${data.total_tenants || '0'}</strong> total onboarded organizations</span>`;
      }
      if (incomeEl) incomeEl.textContent = formatCurrency(data.monthly_received_income || 0);
      if (regEl) regEl.textContent = data.recent_registrations_7d || '0';

      // Render Dynamic SVG Revenue Trend Chart
      renderRevenueTrendChart(data.revenue_trend || [], chartWrapper);

    } catch (err) {
      if (activeEl) activeEl.textContent = '—';
      if (incomeEl) incomeEl.textContent = '—';
      if (regEl) regEl.textContent = '—';
      if (chartWrapper) {
        chartWrapper.innerHTML = `
          <div class="table-empty-box" style="height: 100%; display: flex; flex-direction: column; align-items: center; justify-content: center;">
            <div style="color: var(--color-error); font-weight: 600; margin-bottom: 0.75rem;">Connection error while fetching telemetry metrics.</div>
            <button type="button" class="btn btn-secondary btn-sm" id="retry-metrics-btn">Retry Telemetry</button>
          </div>
        `;
        const rBtn = document.getElementById('retry-metrics-btn');
        if (rBtn) rBtn.addEventListener('click', () => loadDashboardMetrics());
      }
      showToast('Connection error while fetching telemetry metrics.', 'error');
    }
  }

  function renderRevenueTrendChart(trendData, container) {
    if (!container) return;

    if (!trendData || trendData.length === 0) {
      container.innerHTML = `
        <div class="table-empty-box" style="height: 100%; display: flex; flex-direction: column; align-items: center; justify-content: center;">
          <svg width="40" height="40" fill="none" viewBox="0 0 24 24" stroke="currentColor" class="table-empty-icon">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 12l3-3 3 3 4-4M8 21l4-4 4 4M3 4h18M4 4h16v12a1 1 0 01-1 1H5a1 1 0 01-1-1V4z"/>
          </svg>
          <div>No transaction revenue data recorded yet.</div>
        </div>
      `;
      return;
    }

    const width = 800;
    const height = 240;
    const paddingLeft = 60;
    const paddingRight = 20;
    const paddingTop = 25;
    const paddingBottom = 35;

    const chartW = width - paddingLeft - paddingRight;
    const chartH = height - paddingTop - paddingBottom;

    // Calculate maximum revenue for scaling
    let maxVal = 0;
    trendData.forEach(item => {
      const v = parseFloat(item.revenue) || 0;
      if (v > maxVal) maxVal = v;
    });
    // Ensure reasonable min height range
    if (maxVal === 0) maxVal = 1000;
    // Round up maxVal to a clean ceiling
    maxVal = Math.ceil(maxVal * 1.15);

    const stepX = chartW / (trendData.length > 1 ? (trendData.length - 1) : 1);

    // Calculate Coordinates
    const points = trendData.map((d, i) => {
      const val = parseFloat(d.revenue) || 0;
      const x = paddingLeft + (i * stepX);
      const y = paddingTop + chartH - ((val / maxVal) * chartH);
      return { x, y, val, month: d.month, count: d.count || 0 };
    });

    // Build SVG Path
    let pathD = '';
    points.forEach((p, idx) => {
      if (idx === 0) {
        pathD += `M ${p.x} ${p.y}`;
      } else {
        // Smooth cubic bezier curve
        const prev = points[idx - 1];
        const cpX1 = prev.x + (p.x - prev.x) / 2;
        const cpY1 = prev.y;
        const cpX2 = prev.x + (p.x - prev.x) / 2;
        const cpY2 = p.y;
        pathD += ` C ${cpX1} ${cpY1}, ${cpX2} ${cpY2}, ${p.x} ${p.y}`;
      }
    });

    // Area fill path
    const lastP = points[points.length - 1];
    const firstP = points[0];
    const areaD = `${pathD} L ${lastP.x} ${paddingTop + chartH} L ${firstP.x} ${paddingTop + chartH} Z`;

    // Horizontal Grid Lines & Y-axis labels
    let gridLinesSvg = '';
    const gridTicks = 4;
    for (let t = 0; t <= gridTicks; t++) {
      const tickVal = (maxVal / gridTicks) * t;
      const yPos = paddingTop + chartH - ((tickVal / maxVal) * chartH);
      gridLinesSvg += `
        <line x1="${paddingLeft}" y1="${yPos}" x2="${width - paddingRight}" y2="${yPos}" stroke="#e2e8f0" stroke-dasharray="3,3" stroke-width="1"/>
        <text x="${paddingLeft - 8}" y="${yPos + 4}" fill="#94a3b8" font-size="10" text-anchor="end" font-family="'JetBrains Mono', monospace">
          ${tickVal >= 1000 ? (tickVal / 1000).toFixed(1) + 'k' : Math.round(tickVal)}
        </text>
      `;
    }

    // X-axis Month Labels & Interactive Data Circles
    let pointsAndLabelsSvg = '';
    points.forEach((p) => {
      pointsAndLabelsSvg += `
        <!-- X Axis Label -->
        <text x="${p.x}" y="${height - 8}" fill="#64748b" font-size="11" font-weight="600" text-anchor="middle" font-family="'Plus Jakarta Sans', sans-serif">
          ${p.month}
        </text>

        <!-- Interactive Point -->
        <g class="chart-point-group" tabindex="0">
          <circle cx="${p.x}" cy="${p.y}" r="5" fill="#4f46e5" stroke="#ffffff" stroke-width="2.5" style="cursor: pointer; transition: all 150ms;">
            <title>${p.month}: ${formatCurrency(p.val)} (${p.count} transactions)</title>
          </circle>
          <circle cx="${p.x}" cy="${p.y}" r="14" fill="transparent" style="cursor: pointer;">
            <title>${p.month}: ${formatCurrency(p.val)} (${p.count} transactions)</title>
          </circle>
        </g>
      `;
    });

    container.innerHTML = `
      <svg class="chart-svg" viewBox="0 0 ${width} ${height}" preserveAspectRatio="none">
        <defs>
          <linearGradient id="revenueGradient" x1="0%" y1="0%" x2="0%" y2="100%">
            <stop offset="0%" stop-color="#4f46e5" stop-opacity="0.32"/>
            <stop offset="100%" stop-color="#4f46e5" stop-opacity="0.0"/>
          </linearGradient>
        </defs>

        <!-- Grid Lines -->
        ${gridLinesSvg}

        <!-- Gradient Area Fill -->
        <path d="${areaD}" fill="url(#revenueGradient)"/>

        <!-- Line Stroke -->
        <path d="${pathD}" fill="none" stroke="#4f46e5" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"/>

        <!-- Points and Labels -->
        ${pointsAndLabelsSvg}
      </svg>
    `;
  }

  // =========================================================================
  // View 2: Tenants Management
  // =========================================================================
  async function loadTenants() {
    const tbody = document.getElementById('tenants-table-body');
    const pageInfo = document.getElementById('tenants-page-info');
    const prevBtn = document.getElementById('tenants-prev-btn');
    const nextBtn = document.getElementById('tenants-next-btn');

    if (!tbody) return;
    tbody.innerHTML = `<tr><td colspan="6" class="table-empty-box"><span class="btn-spinner" style="margin: 0 auto 0.5rem; display: block;"></span>Loading tenant organizations...</td></tr>`;

    try {
      const q = new URLSearchParams({
        page: state.tenants.page,
        page_size: state.tenants.pageSize
      });
      if (state.tenants.status) q.set('status', state.tenants.status);
      if (state.tenants.search) q.set('search', state.tenants.search);

      const res = await window.API.get(`/platform/tenants?${q.toString()}`);
      if (!res.ok) {
        tbody.innerHTML = `
          <tr>
            <td colspan="6" class="table-empty-box" style="color: var(--color-error);">
              <div style="margin-bottom: 0.5rem;">${escapeHTML(res.error || 'Failed to retrieve tenants.')}</div>
              <button type="button" class="btn btn-secondary btn-sm" id="retry-tenants-btn">Retry</button>
            </td>
          </tr>
        `;
        const rBtn = document.getElementById('retry-tenants-btn');
        if (rBtn) rBtn.addEventListener('click', () => loadTenants());
        return;
      }

      const tenants = res.data || [];
      const pagination = res.raw && res.raw.pagination ? res.raw.pagination : {};
      state.tenants.total = pagination.total_items || tenants.length;

      if (tenants.length === 0) {
        tbody.innerHTML = `
          <tr>
            <td colspan="6" class="table-empty-box">
              <svg width="36" height="36" fill="none" viewBox="0 0 24 24" stroke="currentColor" class="table-empty-icon">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"/>
              </svg>
              <div>No tenant businesses found matching current criteria.</div>
            </td>
          </tr>
        `;
        if (pageInfo) pageInfo.textContent = 'Showing 0 tenants';
        if (prevBtn) prevBtn.disabled = true;
        if (nextBtn) nextBtn.disabled = true;
        return;
      }

      // Populate Table Rows
      tbody.innerHTML = tenants.map(t => `
        <tr>
          <td>
            <div style="font-weight: 700; color: var(--color-text-main);">${escapeHTML(t.name)}</div>
            <div style="font-size: 0.75rem; color: var(--color-text-muted); font-family: var(--font-mono);">ID: ${escapeHTML(t.id)}</div>
          </td>
          <td>
            <a href="mailto:${escapeHTML(t.email)}" style="color: var(--color-text-main); font-weight: 500;">${escapeHTML(t.email)}</a>
          </td>
          <td>${t.phone ? escapeHTML(t.phone) : '<span style="color: var(--color-text-subtle);">None</span>'}</td>
          <td>
            <span class="badge badge-${t.status || 'inactive'}">
              <span class="badge-dot"></span>
              <span>${t.status || 'unknown'}</span>
            </span>
          </td>
          <td style="color: var(--color-text-muted); font-size: 0.8125rem;">
            ${formatDate(t.created_at)}
          </td>
          <td style="text-align: right; white-space: nowrap;">
            <div class="table-actions" style="justify-content: flex-end;">
              <button type="button" class="btn btn-secondary btn-sm action-view-tenant" data-id="${t.id}" title="View complete details">
                Inspect
              </button>
              <button type="button" class="btn btn-secondary btn-sm action-edit-tenant" data-tenant='${escapeHTML(JSON.stringify(t))}' title="Edit tenant details">
                Edit
              </button>
              <button type="button" class="btn btn-secondary btn-sm action-status-tenant" data-id="${t.id}" data-status="${t.status}" title="Update status">
                Status
              </button>
              <button type="button" class="btn btn-secondary btn-sm action-plan-tenant" data-id="${t.id}" data-name="${escapeHTML(t.name)}" title="Change subscription plan">
                Plan
              </button>
            </div>
          </td>
        </tr>
      `).join('');

      // Update Pagination Controls
      const totalPages = pagination.total_pages || Math.ceil(state.tenants.total / state.tenants.pageSize) || 1;
      if (pageInfo) {
        pageInfo.textContent = `Page ${state.tenants.page} of ${totalPages} (${state.tenants.total} organizations)`;
      }
      if (prevBtn) prevBtn.disabled = state.tenants.page <= 1;
      if (nextBtn) nextBtn.disabled = state.tenants.page >= totalPages;

    } catch (err) {
      tbody.innerHTML = `
        <tr>
          <td colspan="6" class="table-empty-box" style="color: var(--color-error);">
            <div style="margin-bottom: 0.5rem;">An error occurred while communicating with the server.</div>
            <button type="button" class="btn btn-secondary btn-sm" id="retry-tenants-btn">Retry</button>
          </td>
        </tr>
      `;
      const rBtn = document.getElementById('retry-tenants-btn');
      if (rBtn) rBtn.addEventListener('click', () => loadTenants());
    }
  }

  // Show Full Tenant Details Modal
  async function showTenantDetails(id) {
    const titleEl = document.getElementById('tenant-detail-title');
    const bodyEl = document.getElementById('tenant-detail-body');
    const footerEl = document.getElementById('tenant-detail-footer');

    if (bodyEl) {
      bodyEl.innerHTML = `<div style="padding: 3rem; text-align: center;"><span class="btn-spinner" style="margin: 0 auto 0.75rem; display: block;"></span>Loading details...</div>`;
    }
    openModal('tenant-detail-modal');

    try {
      const res = await window.API.get(`/platform/tenants/${id}`);
      if (!res.ok) {
        if (bodyEl) bodyEl.innerHTML = `<div class="alert alert-error">${escapeHTML(res.error || 'Failed to load tenant details.')}</div>`;
        return;
      }

      const d = res.data;
      const tenant = d.tenant || {};
      const admin = d.admin_user || {};
      const sub = d.subscription || {};
      const fin = d.financial_summary || {};

      if (titleEl) titleEl.textContent = `${tenant.name} — Overview`;

      if (bodyEl) {
        bodyEl.innerHTML = `
          <!-- Organization Details Card -->
          <div class="card" style="margin-bottom: 1.25rem;">
            <h3 style="font-size: 0.95rem; font-weight: 700; margin-bottom: 1rem; color: var(--color-primary);">
              Business Organization
            </h3>
            <div class="detail-grid">
              <div class="detail-item">
                <span class="detail-label">Business Name</span>
                <span class="detail-val">${escapeHTML(tenant.name)}</span>
              </div>
              <div class="detail-item">
                <span class="detail-label">Status</span>
                <span>
                  <span class="badge badge-${tenant.status}">
                    <span class="badge-dot"></span>
                    <span>${tenant.status}</span>
                  </span>
                </span>
              </div>
              <div class="detail-item">
                <span class="detail-label">Email</span>
                <span class="detail-val">${escapeHTML(tenant.email)}</span>
              </div>
              <div class="detail-item">
                <span class="detail-label">Phone</span>
                <span class="detail-val">${tenant.phone ? escapeHTML(tenant.phone) : '—'}</span>
              </div>
              <div class="detail-item">
                <span class="detail-label">Organization ID</span>
                <span class="detail-val mono-code" style="font-size: 0.75rem;">${escapeHTML(tenant.id)}</span>
              </div>
              <div class="detail-item">
                <span class="detail-label">Registered On</span>
                <span class="detail-val">${formatDateTime(tenant.created_at)}</span>
              </div>
            </div>
          </div>

          <!-- Subscription Plan Card -->
          <div class="card" style="margin-bottom: 1.25rem;">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
              <h3 style="font-size: 0.95rem; font-weight: 700; color: var(--color-primary); margin: 0;">
                Current Subscription
              </h3>
              <button type="button" class="btn btn-secondary btn-sm action-plan-tenant" data-id="${tenant.id}" data-name="${escapeHTML(tenant.name)}">
                Change Plan
              </button>
            </div>
            <div class="detail-grid">
              <div class="detail-item">
                <span class="detail-label">Plan Name</span>
                <span class="detail-val" style="font-weight: 800; font-size: 1.05rem;">${escapeHTML(sub.current_plan_name || 'None')}</span>
              </div>
              <div class="detail-item">
                <span class="detail-label">Plan Status</span>
                <span>
                  <span class="badge badge-${sub.status || 'expired'}">
                    <span class="badge-dot"></span>
                    <span>${sub.status || 'none'}</span>
                  </span>
                </span>
              </div>
              <div class="detail-item">
                <span class="detail-label">Term Start Date</span>
                <span class="detail-val">${formatDate(sub.start_date || sub.created_at)}</span>
              </div>
              <div class="detail-item">
                <span class="detail-label">Term End Date</span>
                <span class="detail-val">${sub.end_date ? formatDate(sub.end_date) : 'Continuous / Ongoing'}</span>
              </div>
            </div>
          </div>

          <!-- Primary Administrator Card -->
          <div class="card" style="margin-bottom: 1.25rem;">
            <h3 style="font-size: 0.95rem; font-weight: 700; margin-bottom: 1rem; color: var(--color-primary);">
              Designated Primary Admin
            </h3>
            <div class="detail-grid">
              <div class="detail-item">
                <span class="detail-label">Administrator Name</span>
                <span class="detail-val">${admin.name ? escapeHTML(admin.name) : '—'}</span>
              </div>
              <div class="detail-item">
                <span class="detail-label">Admin Email</span>
                <span class="detail-val">${admin.email ? escapeHTML(admin.email) : '—'}</span>
              </div>
              <div class="detail-item">
                <span class="detail-label">Role</span>
                <span class="detail-val">${admin.role ? escapeHTML(admin.role) : '—'}</span>
              </div>
              <div class="detail-item">
                <span class="detail-label">Account Status</span>
                <span class="detail-val">${admin.status ? escapeHTML(admin.status) : '—'}</span>
              </div>
            </div>
          </div>

          <!-- Financial Snapshot Card -->
          <div class="card">
            <h3 style="font-size: 0.95rem; font-weight: 700; margin-bottom: 1rem; color: var(--color-primary);">
              Tenant Bookkeeping Balances
            </h3>
            <div class="detail-grid">
              <div class="detail-item">
                <span class="detail-label">Cash Ledger Balance</span>
                <span class="detail-val">${formatCurrency(fin.cash_balance || 0)}</span>
              </div>
              <div class="detail-item">
                <span class="detail-label">Total Bank Balances</span>
                <span class="detail-val">${formatCurrency(fin.bank_balance || 0)}</span>
              </div>
              <div class="detail-item">
                <span class="detail-label">Customer Receivables</span>
                <span class="detail-val" style="color: var(--color-success);">${formatCurrency(fin.total_receivable || 0)}</span>
              </div>
              <div class="detail-item">
                <span class="detail-label">Vendor Payables</span>
                <span class="detail-val" style="color: var(--color-error);">${formatCurrency(fin.total_payable || 0)}</span>
              </div>
            </div>
          </div>
        `;
      }

      if (footerEl) {
        footerEl.innerHTML = `
          <button type="button" class="btn btn-secondary" data-close-modal="tenant-detail-modal">Close</button>
          <button type="button" class="btn btn-primary action-edit-tenant" data-tenant='${escapeHTML(JSON.stringify(tenant))}'>Edit Organization</button>
        `;
      }

    } catch (err) {
      if (bodyEl) bodyEl.innerHTML = `<div class="alert alert-error">An error occurred while loading tenant details.</div>`;
    }
  }

  // =========================================================================
  // View 3: Subscription Plans Management
  // =========================================================================
  async function loadPlans() {
    const tbody = document.getElementById('plans-table-body');
    if (!tbody) return;

    tbody.innerHTML = `<tr><td colspan="6" class="table-empty-box"><span class="btn-spinner" style="margin: 0 auto 0.5rem; display: block;"></span>Loading subscription plans...</td></tr>`;

    try {
      const q = new URLSearchParams({
        page: state.plans.page,
        page_size: state.plans.pageSize
      });
      if (state.plans.status) q.set('status', state.plans.status);

      const res = await window.API.get(`/platform/plans?${q.toString()}`);
      if (!res.ok) {
        tbody.innerHTML = `
          <tr>
            <td colspan="6" class="table-empty-box" style="color: var(--color-error);">
              <div style="margin-bottom: 0.5rem;">${escapeHTML(res.error || 'Failed to load plans.')}</div>
              <button type="button" class="btn btn-secondary btn-sm" id="retry-plans-btn">Retry</button>
            </td>
          </tr>
        `;
        const rBtn = document.getElementById('retry-plans-btn');
        if (rBtn) rBtn.addEventListener('click', () => loadPlans());
        return;
      }

      const plans = res.data || [];
      state.plans.cached = plans;

      if (plans.length === 0) {
        tbody.innerHTML = `
          <tr>
            <td colspan="6" class="table-empty-box">
              <svg width="36" height="36" fill="none" viewBox="0 0 24 24" stroke="currentColor" class="table-empty-icon">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"/>
              </svg>
              <div>No subscription plans found for this filter.</div>
            </td>
          </tr>
        `;
        return;
      }

      tbody.innerHTML = plans.map(p => `
        <tr>
          <td>
            <div style="font-weight: 700; color: var(--color-text-main); font-size: 1rem;">${escapeHTML(p.plan_name)}</div>
            <div style="font-size: 0.75rem; color: var(--color-text-muted); font-family: var(--font-mono);">ID: ${escapeHTML(p.id)}</div>
          </td>
          <td>
            <span style="font-size: 1.15rem; font-weight: 800; color: var(--color-primary);">${formatCurrency(p.price)}</span>
            <span style="font-size: 0.75rem; color: var(--color-text-muted);">/mo</span>
          </td>
          <td style="max-width: 280px; color: var(--color-text-muted); font-size: 0.85rem;">
            ${p.note ? escapeHTML(p.note) : '<span style="color: var(--color-text-subtle);">Standard tier entitlements</span>'}
          </td>
          <td>
            <span class="badge badge-${p.status}">
              <span class="badge-dot"></span>
              <span>${p.status}</span>
            </span>
          </td>
          <td style="color: var(--color-text-muted); font-size: 0.8125rem;">
            ${formatDate(p.created_at)}
          </td>
          <td style="text-align: right; white-space: nowrap;">
            <div class="table-actions" style="justify-content: flex-end;">
              <button type="button" class="btn btn-secondary btn-sm action-edit-plan" data-plan='${escapeHTML(JSON.stringify(p))}'>
                Edit Plan
              </button>
              ${p.status !== 'archived' ? `
                <button type="button" class="btn btn-secondary btn-sm action-archive-plan" data-id="${p.id}" data-name="${escapeHTML(p.plan_name)}" style="color: var(--color-error);">
                  Archive
                </button>
              ` : ''}
            </div>
          </td>
        </tr>
      `).join('');

    } catch (err) {
      tbody.innerHTML = `
        <tr>
          <td colspan="6" class="table-empty-box" style="color: var(--color-error);">
            <div style="margin-bottom: 0.5rem;">Failed to connect to plans service.</div>
            <button type="button" class="btn btn-secondary btn-sm" id="retry-plans-btn">Retry</button>
          </td>
        </tr>
      `;
      const rBtn = document.getElementById('retry-plans-btn');
      if (rBtn) rBtn.addEventListener('click', () => loadPlans());
    }
  }

  // Populate Plans in Dropdowns
  async function populatePlanSelect(selectEl, selectedId = null) {
    if (!selectEl) return;
    selectEl.innerHTML = '<option value="">Loading plans...</option>';

    try {
      const res = await window.API.get('/platform/plans?status=active&page_size=50');
      if (!res.ok) {
        selectEl.innerHTML = '<option value="">Failed to load plans</option>';
        return;
      }

      const plans = res.data || [];
      state.plans.cached = plans;

      if (plans.length === 0) {
        selectEl.innerHTML = '<option value="">No active plans available</option>';
        return;
      }

      selectEl.innerHTML = '<option value="">Select Plan...</option>' + plans.map(p => `
        <option value="${p.id}" ${selectedId === p.id ? 'selected' : ''}>
          ${escapeHTML(p.plan_name)} — ${formatCurrency(p.price)}/mo
        </option>
      `).join('');

    } catch (e) {
      selectEl.innerHTML = '<option value="">Error loading plans</option>';
    }
  }

  // =========================================================================
  // View 4: Subscribed Tenants Management
  // =========================================================================
  async function loadSubscriptions() {
    const tbody = document.getElementById('subs-table-body');
    const pageInfo = document.getElementById('subs-page-info');
    const prevBtn = document.getElementById('subs-prev-btn');
    const nextBtn = document.getElementById('subs-next-btn');

    if (!tbody) return;
    tbody.innerHTML = `<tr><td colspan="7" class="table-empty-box"><span class="btn-spinner" style="margin: 0 auto 0.5rem; display: block;"></span>Loading subscribed tenants...</td></tr>`;

    try {
      const q = new URLSearchParams({
        page: state.subscriptions.page,
        page_size: state.subscriptions.pageSize
      });
      if (state.subscriptions.status) q.set('status', state.subscriptions.status);

      const res = await window.API.get(`/platform/subscriptions?${q.toString()}`);
      if (!res.ok) {
        tbody.innerHTML = `
          <tr>
            <td colspan="7" class="table-empty-box" style="color: var(--color-error);">
              <div style="margin-bottom: 0.5rem;">${escapeHTML(res.error || 'Failed to load subscriptions.')}</div>
              <button type="button" class="btn btn-secondary btn-sm" id="retry-subs-btn">Retry</button>
            </td>
          </tr>
        `;
        const rBtn = document.getElementById('retry-subs-btn');
        if (rBtn) rBtn.addEventListener('click', () => loadSubscriptions());
        return;
      }

      const subs = res.data || [];
      const pagination = res.raw && res.raw.pagination ? res.raw.pagination : {};
      state.subscriptions.total = pagination.total_items || subs.length;

      if (subs.length === 0) {
        tbody.innerHTML = `
          <tr>
            <td colspan="7" class="table-empty-box">
              <svg width="36" height="36" fill="none" viewBox="0 0 24 24" stroke="currentColor" class="table-empty-icon">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"/>
              </svg>
              <div>No subscribed tenants recorded under this status.</div>
            </td>
          </tr>
        `;
        if (pageInfo) pageInfo.textContent = 'Showing 0 subscriptions';
        if (prevBtn) prevBtn.disabled = true;
        if (nextBtn) nextBtn.disabled = true;
        return;
      }

      tbody.innerHTML = subs.map(s => `
        <tr>
          <td>
            <div style="font-weight: 700; color: var(--color-text-main);">${escapeHTML(s.tenant_name || 'Organization')}</div>
            <div style="font-size: 0.75rem; color: var(--color-text-muted);">${escapeHTML(s.tenant_email || '')}</div>
          </td>
          <td>
            <span style="font-weight: 700; color: var(--color-primary);">${escapeHTML(s.current_plan_name || 'Standard')}</span>
          </td>
          <td>
            <span class="badge badge-${s.status}">
              <span class="badge-dot"></span>
              <span>${s.status}</span>
            </span>
          </td>
          <td style="font-size: 0.8125rem;">${formatDate(s.start_date || s.created_at)}</td>
          <td style="font-size: 0.8125rem;">${s.end_date ? formatDate(s.end_date) : '<span style="color: var(--color-text-subtle);">Perpetual</span>'}</td>
          <td style="font-size: 0.8125rem; color: var(--color-text-muted);">${formatDate(s.created_at)}</td>
          <td style="text-align: right; white-space: nowrap;">
            <div class="table-actions" style="justify-content: flex-end;">
              <button type="button" class="btn btn-secondary btn-sm action-view-tenant" data-id="${s.tenant_id}" title="Inspect tenant details">
                Details
              </button>
              <button type="button" class="btn btn-secondary btn-sm action-plan-tenant" data-id="${s.tenant_id}" data-name="${escapeHTML(s.tenant_name)}" title="Change subscription plan">
                Plan
              </button>
              <button type="button" class="btn btn-secondary btn-sm action-status-sub" data-tenant-id="${s.tenant_id}" data-status="${s.status}" title="Update subscription lifecycle status">
                Status
              </button>
            </div>
          </td>
        </tr>
      `).join('');

      const totalPages = pagination.total_pages || Math.ceil(state.subscriptions.total / state.subscriptions.pageSize) || 1;
      if (pageInfo) {
        pageInfo.textContent = `Page ${state.subscriptions.page} of ${totalPages} (${state.subscriptions.total} subscriptions)`;
      }
      if (prevBtn) prevBtn.disabled = state.subscriptions.page <= 1;
      if (nextBtn) nextBtn.disabled = state.subscriptions.page >= totalPages;

    } catch (err) {
      tbody.innerHTML = `
        <tr>
          <td colspan="7" class="table-empty-box" style="color: var(--color-error);">
            <div style="margin-bottom: 0.5rem;">An error occurred while loading subscriptions.</div>
            <button type="button" class="btn btn-secondary btn-sm" id="retry-subs-btn">Retry</button>
          </td>
        </tr>
      `;
      const rBtn = document.getElementById('retry-subs-btn');
      if (rBtn) rBtn.addEventListener('click', () => loadSubscriptions());
    }
  }

  // =========================================================================
  // View 5: Platform Transactions Management
  // =========================================================================
  async function loadTransactions() {
    const tbody = document.getElementById('txns-table-body');
    const pageInfo = document.getElementById('txns-page-info');
    const prevBtn = document.getElementById('txns-prev-btn');
    const nextBtn = document.getElementById('txns-next-btn');

    if (!tbody) return;
    tbody.innerHTML = `<tr><td colspan="8" class="table-empty-box"><span class="btn-spinner" style="margin: 0 auto 0.5rem; display: block;"></span>Loading transactions ledger...</td></tr>`;

    try {
      const q = new URLSearchParams({
        page: state.transactions.page,
        page_size: state.transactions.pageSize
      });
      if (state.transactions.status) q.set('status', state.transactions.status);

      const res = await window.API.get(`/platform/transactions?${q.toString()}`);
      if (!res.ok) {
        tbody.innerHTML = `
          <tr>
            <td colspan="8" class="table-empty-box" style="color: var(--color-error);">
              <div style="margin-bottom: 0.5rem;">${escapeHTML(res.error || 'Failed to load transactions.')}</div>
              <button type="button" class="btn btn-secondary btn-sm" id="retry-txns-btn">Retry</button>
            </td>
          </tr>
        `;
        const rBtn = document.getElementById('retry-txns-btn');
        if (rBtn) rBtn.addEventListener('click', () => loadTransactions());
        return;
      }

      const txns = res.data || [];
      const pagination = res.raw && res.raw.pagination ? res.raw.pagination : {};
      state.transactions.total = pagination.total_items || txns.length;

      if (txns.length === 0) {
        tbody.innerHTML = `
          <tr>
            <td colspan="8" class="table-empty-box">
              <svg width="36" height="36" fill="none" viewBox="0 0 24 24" stroke="currentColor" class="table-empty-icon">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
              </svg>
              <div>No transactions recorded matching this filter.</div>
            </td>
          </tr>
        `;
        if (pageInfo) pageInfo.textContent = 'Showing 0 transactions';
        if (prevBtn) prevBtn.disabled = true;
        if (nextBtn) nextBtn.disabled = true;
        return;
      }

      tbody.innerHTML = txns.map(tx => `
        <tr>
          <td>
            <span class="mono-code" title="${escapeHTML(tx.transaction_id)}">${escapeHTML(tx.transaction_id)}</span>
          </td>
          <td>
            <div style="font-weight: 700; color: var(--color-text-main);">${escapeHTML(tx.tenant_name || 'Tenant')}</div>
            <div style="font-size: 0.75rem; color: var(--color-text-muted);">${escapeHTML(tx.tenant_email || '')}</div>
          </td>
          <td style="font-weight: 600;">${escapeHTML(tx.plan_name || 'Standard')}</td>
          <td>
            <span style="font-weight: 800; color: var(--color-text-main); font-size: 1.05rem;">
              ${formatCurrency(tx.amount)}
            </span>
          </td>
          <td>
            <span class="badge" style="background: var(--color-surface-hover); color: var(--color-text-main); border: 1px solid var(--color-border); text-transform: uppercase;">
              ${escapeHTML(tx.payment_method)}
            </span>
          </td>
          <td>
            <span class="badge badge-${tx.status}">
              <span class="badge-dot"></span>
              <span>${tx.status}</span>
            </span>
            ${tx.failure_reason ? `<div style="font-size: 0.72rem; color: var(--color-error); margin-top: 0.2rem;">${escapeHTML(tx.failure_reason)}</div>` : ''}
          </td>
          <td style="font-size: 0.8125rem; color: var(--color-text-muted); white-space: nowrap;">
            ${formatDateTime(tx.created_at)}
          </td>
          <td style="text-align: right; white-space: nowrap;">
            <div class="table-actions" style="justify-content: flex-end;">
              <button type="button" class="btn btn-secondary btn-sm action-view-tenant" data-id="${tx.tenant_id}" title="Inspect tenant details">
                Tenant
              </button>
            </div>
          </td>
        </tr>
      `).join('');

      const totalPages = pagination.total_pages || Math.ceil(state.transactions.total / state.transactions.pageSize) || 1;
      if (pageInfo) {
        pageInfo.textContent = `Page ${state.transactions.page} of ${totalPages} (${state.transactions.total} transactions)`;
      }
      if (prevBtn) prevBtn.disabled = state.transactions.page <= 1;
      if (nextBtn) nextBtn.disabled = state.transactions.page >= totalPages;

    } catch (err) {
      tbody.innerHTML = `
        <tr>
          <td colspan="8" class="table-empty-box" style="color: var(--color-error);">
            <div style="margin-bottom: 0.5rem;">An error occurred while loading transactions.</div>
            <button type="button" class="btn btn-secondary btn-sm" id="retry-txns-btn">Retry</button>
          </td>
        </tr>
      `;
      const rBtn = document.getElementById('retry-txns-btn');
      if (rBtn) rBtn.addEventListener('click', () => loadTransactions());
    }
  }

  // =========================================================================
  // View 6: Application Metrics & Grafana
  // =========================================================================
  function setupMetricsView() {
    const grafanaBtn = document.getElementById('open-grafana-btn');
    if (grafanaBtn) {
      const hostname = window.location.hostname || 'localhost';
      grafanaBtn.href = `http://${hostname}:3000`;
    }
  }

  // =========================================================================
  // Event Bindings and Initializers
  // =========================================================================
  function initEventHandlers() {
    // 1. Navigation Clicks
    document.addEventListener('click', (e) => {
      // Sidebar Links
      const sidebarLink = e.target.closest('.sidebar-link[data-view]');
      if (sidebarLink) {
        e.preventDefault();
        const href = sidebarLink.getAttribute('href');
        navigateTo(href);
        return;
      }

      // Generic data-nav elements
      const navTarget = e.target.closest('[data-nav]');
      if (navTarget) {
        e.preventDefault();
        const targetPath = navTarget.getAttribute('data-nav');
        navigateTo(targetPath);
        return;
      }

      // Backdrop click on modal overlay closes it
      if (e.target.classList && e.target.classList.contains('modal') && e.target.classList.contains('active')) {
        closeModal(e.target.id);
        return;
      }

      // Close modal buttons
      const closeBtn = e.target.closest('[data-close-modal]');
      if (closeBtn) {
        const modalId = closeBtn.getAttribute('data-close-modal');
        closeModal(modalId);
        return;
      }

      // View Tenant action buttons
      const viewTenantBtn = e.target.closest('.action-view-tenant');
      if (viewTenantBtn) {
        const id = viewTenantBtn.getAttribute('data-id');
        showTenantDetails(id);
        return;
      }

      // Edit Tenant action buttons
      const editTenantBtn = e.target.closest('.action-edit-tenant');
      if (editTenantBtn) {
        const raw = editTenantBtn.getAttribute('data-tenant');
        try {
          const tenant = JSON.parse(raw);
          closeModal('tenant-detail-modal');
          document.getElementById('edit-tenant-id').value = tenant.id || '';
          document.getElementById('edit-tenant-name').value = tenant.name || '';
          document.getElementById('edit-tenant-email').value = tenant.email || '';
          document.getElementById('edit-tenant-phone').value = tenant.phone || '';
          document.getElementById('edit-tenant-status').value = tenant.status || 'active';
          openModal('edit-tenant-modal');
        } catch (e) {}
        return;
      }

      // Change Tenant Status action buttons
      const statusTenantBtn = e.target.closest('.action-status-tenant');
      if (statusTenantBtn) {
        const id = statusTenantBtn.getAttribute('data-id');
        const currStatus = statusTenantBtn.getAttribute('data-status') || 'active';
        document.getElementById('status-tenant-id').value = id;
        document.getElementById('status-tenant-select').value = currStatus;
        openModal('tenant-status-modal');
        return;
      }

      // Change Tenant Plan action buttons
      const planTenantBtn = e.target.closest('.action-plan-tenant');
      if (planTenantBtn) {
        const id = planTenantBtn.getAttribute('data-id');
        closeModal('tenant-detail-modal');
        document.getElementById('plan-tenant-id').value = id;
        const planSelect = document.getElementById('change-plan-select');
        populatePlanSelect(planSelect);
        openModal('tenant-plan-modal');
        return;
      }

      // Change Subscription Status action buttons
      const statusSubBtn = e.target.closest('.action-status-sub');
      if (statusSubBtn) {
        const tenantId = statusSubBtn.getAttribute('data-tenant-id');
        const currStatus = statusSubBtn.getAttribute('data-status') || 'active';
        const hiddenId = document.getElementById('status-sub-tenant-id');
        const selectEl = document.getElementById('status-sub-select');
        if (hiddenId) hiddenId.value = tenantId;
        if (selectEl) selectEl.value = currStatus;
        openModal('sub-status-modal');
        return;
      }

      // Edit Plan action buttons
      const editPlanBtn = e.target.closest('.action-edit-plan');
      if (editPlanBtn) {
        const raw = editPlanBtn.getAttribute('data-plan');
        try {
          const plan = JSON.parse(raw);
          document.getElementById('edit-plan-id').value = plan.id || '';
          document.getElementById('edit-plan-name').value = plan.plan_name || '';
          document.getElementById('edit-plan-price').value = plan.price || '0';
          document.getElementById('edit-plan-note').value = plan.note || '';
          document.getElementById('edit-plan-status').value = plan.status || 'active';
          openModal('edit-plan-modal');
        } catch (e) {}
        return;
      }

      // Archive Plan action buttons
      const archivePlanBtn = e.target.closest('.action-archive-plan');
      if (archivePlanBtn) {
        const id = archivePlanBtn.getAttribute('data-id');
        const name = archivePlanBtn.getAttribute('data-name');
        confirmDialog(
          'Archive Subscription Plan',
          `Are you sure you want to archive "${name}"? It will no longer be available for new tenant subscriptions. Existing subscriptions will continue until renewal.`,
          async () => {
            try {
              const res = await window.API.delete(`/platform/plans/${id}`);
              if (!res.ok) {
                showToast(res.error || 'Failed to archive plan', 'error');
                return;
              }
              showToast(`Plan "${name}" archived successfully.`, 'success');
              loadPlans();
            } catch (e) {
              showToast('Connection error while archiving plan.', 'error');
            }
          },
          'Archive Plan',
          true
        );
        return;
      }
    });

    // 2. Escape Key Listener to Close Modals & Mobile Sidebar
    document.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') {
        closeMobileSidebar();
        closeAllModals();
      }
    });

    // 3. Browser History Back/Forward Popstate Event
    window.addEventListener('popstate', (e) => {
      const path = (e.state && e.state.path) ? e.state.path : window.location.pathname;
      navigateTo(path, false);
    });

    // 4. Window Resize Cleanup
    window.addEventListener('resize', () => {
      if (window.innerWidth > 992) {
        closeMobileSidebar();
      }
      if (!document.querySelector('.modal.active') && (!document.getElementById('platform-sidebar') || !document.getElementById('platform-sidebar').classList.contains('open'))) {
        document.body.style.overflow = '';
      }
    });

    // 5. Mobile Hamburger, Close Button & Backdrop
    const toggleBtn = document.getElementById('platform-sidebar-toggle');
    const closeBtn = document.getElementById('platform-sidebar-close');
    const backdrop = document.getElementById('platform-sidebar-backdrop');
    if (toggleBtn) toggleBtn.addEventListener('click', toggleMobileSidebar);
    if (closeBtn) closeBtn.addEventListener('click', closeMobileSidebar);
    if (backdrop) backdrop.addEventListener('click', closeMobileSidebar);

    // 6. Sign Out Buttons (Topbar and Sidebar)
    const handleLogout = () => {
      confirmDialog(
        'Platform Sign Out',
        'Are you sure you want to end your administrative superadmin session?',
        async () => {
          try {
            if (window.Auth && typeof window.Auth.platformLogout === 'function') {
              await window.Auth.platformLogout();
            } else if (window.API) {
              await window.API.post('/auth/logout', {});
              if (window.Auth) window.Auth.clearPlatformAuth();
            }
          } catch (e) {
            if (window.Auth) window.Auth.clearPlatformAuth();
          } finally {
            window.location.replace('/platform/login');
          }
        },
        'Sign Out',
        true
      );
    };

    const topbarLogout = document.getElementById('topbar-logout-btn');
    const sidebarLogout = document.getElementById('sidebar-logout-btn');
    if (topbarLogout) topbarLogout.addEventListener('click', handleLogout);
    if (sidebarLogout) sidebarLogout.addEventListener('click', handleLogout);

    // 7. Dashboard Refresh
    const dashRefresh = document.getElementById('dashboard-refresh-btn');
    if (dashRefresh) dashRefresh.addEventListener('click', () => {
      loadDashboardMetrics();
      showToast('Dashboard telemetry refreshed.', 'info');
    });

    // 6. Tenants Filtering & Search
    const tenantSearch = document.getElementById('tenants-search-input');
    const tenantStatus = document.getElementById('tenants-status-filter');
    const tenantPrev = document.getElementById('tenants-prev-btn');
    const tenantNext = document.getElementById('tenants-next-btn');

    let searchTimer = null;
    if (tenantSearch) {
      tenantSearch.addEventListener('input', () => {
        clearTimeout(searchTimer);
        searchTimer = setTimeout(() => {
          state.tenants.search = tenantSearch.value.trim();
          state.tenants.page = 1;
          loadTenants();
        }, 300);
      });
    }

    if (tenantStatus) {
      tenantStatus.addEventListener('change', () => {
        state.tenants.status = tenantStatus.value === 'all' ? '' : tenantStatus.value;
        state.tenants.page = 1;
        loadTenants();
      });
    }

    if (tenantPrev) {
      tenantPrev.addEventListener('click', () => {
        if (state.tenants.page > 1) {
          state.tenants.page--;
          loadTenants();
        }
      });
    }

    if (tenantNext) {
      tenantNext.addEventListener('click', () => {
        state.tenants.page++;
        loadTenants();
      });
    }

    // 7. Onboard Tenant Modal & Submission
    const openOnboardBtn = document.getElementById('open-onboard-tenant-modal');
    const onboardForm = document.getElementById('onboard-tenant-form');
    const onboardPlanSelect = document.getElementById('onboard-plan');
    const onboardSubmitBtn = document.getElementById('onboard-submit-btn');

    if (openOnboardBtn) {
      openOnboardBtn.addEventListener('click', () => {
        if (onboardForm) onboardForm.reset();
        populatePlanSelect(onboardPlanSelect);
        openModal('onboard-tenant-modal');
      });
    }

    if (onboardForm) {
      onboardForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const name = document.getElementById('onboard-name').value.trim();
        const email = document.getElementById('onboard-email').value.trim();
        const phone = document.getElementById('onboard-phone').value.trim();
        const admin_password = document.getElementById('onboard-password').value;
        const plan_id = onboardPlanSelect.value;

        if (!name || !email || !admin_password || !plan_id) {
          showToast('Please complete all required onboarding fields.', 'error');
          return;
        }

        if (window.Auth) window.Auth.setButtonLoading(onboardSubmitBtn, true, 'Onboarding organization...');

        try {
          const res = await window.API.post('/platform/tenants', {
            name,
            email,
            phone,
            admin_name: name,
            admin_email: email,
            admin_password,
            plan_id
          });

          if (window.Auth) window.Auth.setButtonLoading(onboardSubmitBtn, false);

          if (!res.ok) {
            showToast(res.error || 'Failed to onboard tenant', 'error');
            return;
          }

          showToast(`Tenant "${name}" onboarded successfully!`, 'success');
          closeModal('onboard-tenant-modal');
          onboardForm.reset();
          loadTenants();
          loadDashboardMetrics();

        } catch (err) {
          if (window.Auth) window.Auth.setButtonLoading(onboardSubmitBtn, false);
          showToast('Connection error while onboarding tenant.', 'error');
        }
      });
    }

    // 8. Edit Tenant Form Submission
    const editTenantForm = document.getElementById('edit-tenant-form');
    const editTenantSubmitBtn = document.getElementById('edit-tenant-submit-btn');

    if (editTenantForm) {
      editTenantForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const id = document.getElementById('edit-tenant-id').value;
        const name = document.getElementById('edit-tenant-name').value.trim();
        const email = document.getElementById('edit-tenant-email').value.trim();
        const phone = document.getElementById('edit-tenant-phone').value.trim();
        const status = document.getElementById('edit-tenant-status').value;

        if (!name || !email) {
          showToast('Organization name and email are required.', 'error');
          return;
        }

        if (window.Auth) window.Auth.setButtonLoading(editTenantSubmitBtn, true, 'Saving changes...');

        try {
          const res = await window.API.put(`/platform/tenants/${id}`, {
            name,
            email,
            phone,
            status
          });

          if (window.Auth) window.Auth.setButtonLoading(editTenantSubmitBtn, false);

          if (!res.ok) {
            showToast(res.error || 'Failed to update tenant details.', 'error');
            return;
          }

          showToast('Tenant organization updated successfully.', 'success');
          closeModal('edit-tenant-modal');
          closeModal('tenant-detail-modal');
          loadTenants();
          loadDashboardMetrics();

        } catch (err) {
          if (window.Auth) window.Auth.setButtonLoading(editTenantSubmitBtn, false);
          showToast('Connection error while updating tenant.', 'error');
        }
      });
    }

    // 9. Update Tenant Status Form Submission
    const statusForm = document.getElementById('tenant-status-form');
    const statusSubmitBtn = document.getElementById('status-submit-btn');

    if (statusForm) {
      statusForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const id = document.getElementById('status-tenant-id').value;
        const status = document.getElementById('status-tenant-select').value;

        if (window.Auth) window.Auth.setButtonLoading(statusSubmitBtn, true, 'Applying status...');

        try {
          const res = await window.API.patch(`/platform/tenants/${id}/status`, { status });
          if (window.Auth) window.Auth.setButtonLoading(statusSubmitBtn, false);

          if (!res.ok) {
            showToast(res.error || 'Failed to update status', 'error');
            return;
          }

          showToast(`Tenant status updated to "${status}".`, 'success');
          closeModal('tenant-status-modal');
          loadTenants();
          loadDashboardMetrics();

        } catch (err) {
          if (window.Auth) window.Auth.setButtonLoading(statusSubmitBtn, false);
          showToast('Connection error while updating tenant status.', 'error');
        }
      });
    }

    // 10. Update Tenant Plan Form Submission
    const planForm = document.getElementById('tenant-plan-form');
    const planSubmitBtn = document.getElementById('plan-change-submit-btn');

    if (planForm) {
      planForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const id = document.getElementById('plan-tenant-id').value;
        const plan_id = document.getElementById('change-plan-select').value;

        if (!plan_id) {
          showToast('Please select a target subscription plan.', 'error');
          return;
        }

        if (window.Auth) window.Auth.setButtonLoading(planSubmitBtn, true, 'Updating subscription...');

        try {
          const res = await window.API.post(`/platform/tenants/${id}/subscription`, { plan_id });
          if (window.Auth) window.Auth.setButtonLoading(planSubmitBtn, false);

          if (!res.ok) {
            showToast(res.error || 'Failed to update subscription plan', 'error');
            return;
          }

          showToast('Tenant subscription plan updated successfully.', 'success');
          closeModal('tenant-plan-modal');
          closeModal('tenant-detail-modal');
          loadTenants();
          loadSubscriptions();
          loadDashboardMetrics();

        } catch (err) {
          if (window.Auth) window.Auth.setButtonLoading(planSubmitBtn, false);
          showToast('Connection error while updating subscription.', 'error');
        }
      });
    }

    // 11. Plans Management
    const plansFilter = document.getElementById('plans-status-filter');
    if (plansFilter) {
      plansFilter.addEventListener('change', () => {
        state.plans.status = plansFilter.value === 'all' ? '' : plansFilter.value;
        loadPlans();
      });
    }

    const openCreatePlanBtn = document.getElementById('open-create-plan-modal');
    const createPlanForm = document.getElementById('create-plan-form');
    const createPlanSubmitBtn = document.getElementById('create-plan-submit-btn');

    if (openCreatePlanBtn) {
      openCreatePlanBtn.addEventListener('click', () => {
        if (createPlanForm) createPlanForm.reset();
        openModal('create-plan-modal');
      });
    }

    if (createPlanForm) {
      createPlanForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const plan_name = document.getElementById('plan-name').value.trim();
        const price = document.getElementById('plan-price').value;
        const note = document.getElementById('plan-note').value.trim();
        const status = document.getElementById('plan-status').value;

        if (!plan_name || price === '') {
          showToast('Plan name and price are required.', 'error');
          return;
        }

        if (window.Auth) window.Auth.setButtonLoading(createPlanSubmitBtn, true, 'Creating plan...');

        try {
          const res = await window.API.post('/platform/plans', {
            plan_name,
            price: price.toString(),
            note,
            status
          });

          if (window.Auth) window.Auth.setButtonLoading(createPlanSubmitBtn, false);

          if (!res.ok) {
            showToast(res.error || 'Failed to create plan.', 'error');
            return;
          }

          showToast(`Plan "${plan_name}" created successfully!`, 'success');
          closeModal('create-plan-modal');
          createPlanForm.reset();
          loadPlans();

        } catch (err) {
          if (window.Auth) window.Auth.setButtonLoading(createPlanSubmitBtn, false);
          showToast('Connection error while creating plan.', 'error');
        }
      });
    }

    // Edit Plan Form Submission
    const editPlanForm = document.getElementById('edit-plan-form');
    const editPlanSubmitBtn = document.getElementById('edit-plan-submit-btn');

    if (editPlanForm) {
      editPlanForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const id = document.getElementById('edit-plan-id').value;
        const plan_name = document.getElementById('edit-plan-name').value.trim();
        const price = document.getElementById('edit-plan-price').value;
        const note = document.getElementById('edit-plan-note').value.trim();
        const status = document.getElementById('edit-plan-status').value;

        if (!plan_name || price === '') {
          showToast('Plan name and price are required.', 'error');
          return;
        }

        if (window.Auth) window.Auth.setButtonLoading(editPlanSubmitBtn, true, 'Saving plan...');

        try {
          const res = await window.API.put(`/platform/plans/${id}`, {
            plan_name,
            price: price.toString(),
            note,
            status
          });

          if (window.Auth) window.Auth.setButtonLoading(editPlanSubmitBtn, false);

          if (!res.ok) {
            showToast(res.error || 'Failed to update plan.', 'error');
            return;
          }

          showToast(`Plan "${plan_name}" updated successfully.`, 'success');
          closeModal('edit-plan-modal');
          loadPlans();

        } catch (err) {
          if (window.Auth) window.Auth.setButtonLoading(editPlanSubmitBtn, false);
          showToast('Connection error while updating plan.', 'error');
        }
      });
    }

    // 12. Subscriptions Management
    const subsFilter = document.getElementById('subs-status-filter');
    const subsRefresh = document.getElementById('subscriptions-refresh-btn');
    const subsPrev = document.getElementById('subs-prev-btn');
    const subsNext = document.getElementById('subs-next-btn');

    if (subsFilter) {
      subsFilter.addEventListener('change', () => {
        state.subscriptions.status = subsFilter.value === 'all' ? '' : subsFilter.value;
        state.subscriptions.page = 1;
        loadSubscriptions();
      });
    }

    if (subsRefresh) subsRefresh.addEventListener('click', () => loadSubscriptions());

    if (subsPrev) {
      subsPrev.addEventListener('click', () => {
        if (state.subscriptions.page > 1) {
          state.subscriptions.page--;
          loadSubscriptions();
        }
      });
    }

    if (subsNext) {
      subsNext.addEventListener('click', () => {
        state.subscriptions.page++;
        loadSubscriptions();
      });
    }

    // 13. Transactions Management
    const txnsFilter = document.getElementById('txns-status-filter');
    const txnsRefresh = document.getElementById('transactions-refresh-btn');
    const txnsPrev = document.getElementById('txns-prev-btn');
    const txnsNext = document.getElementById('txns-next-btn');

    if (txnsFilter) {
      txnsFilter.addEventListener('change', () => {
        state.transactions.status = txnsFilter.value === 'all' ? '' : txnsFilter.value;
        state.transactions.page = 1;
        loadTransactions();
      });
    }

    if (txnsRefresh) txnsRefresh.addEventListener('click', () => loadTransactions());

    if (txnsPrev) {
      txnsPrev.addEventListener('click', () => {
        if (state.transactions.page > 1) {
          state.transactions.page--;
          loadTransactions();
        }
      });
    }

    if (txnsNext) {
      txnsNext.addEventListener('click', () => {
        state.transactions.page++;
        loadTransactions();
      });
    }

    // 14. Subscription Lifecycle Status Form Submission
    const subStatusForm = document.getElementById('sub-status-form');
    const subStatusSubmitBtn = document.getElementById('sub-status-submit-btn');

    if (subStatusForm) {
      subStatusForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const tenantId = document.getElementById('status-sub-tenant-id').value;
        const status = document.getElementById('status-sub-select').value;

        if (!tenantId || !status) {
          showToast('Please select a valid subscription status.', 'error');
          return;
        }

        if (window.Auth) window.Auth.setButtonLoading(subStatusSubmitBtn, true, 'Applying status...');

        try {
          const res = await window.API.patch(`/platform/tenants/${tenantId}/subscription/status`, { status });
          if (window.Auth) window.Auth.setButtonLoading(subStatusSubmitBtn, false);

          if (!res.ok) {
            showToast(res.error || 'Failed to update subscription status', 'error');
            return;
          }

          showToast(`Subscription status updated to "${status}".`, 'success');
          closeModal('sub-status-modal');
          loadSubscriptions();
          loadDashboardMetrics();

        } catch (err) {
          if (window.Auth) window.Auth.setButtonLoading(subStatusSubmitBtn, false);
          showToast('Connection error while updating subscription status.', 'error');
        }
      });
    }
  }

  // =========================================================================
  // Initialization Routine
  // =========================================================================
  async function initializePlatformPortal() {
    // 1. Verify Platform Authentication with ping
    try {
      const ping = await window.API.get('/platform/ping');
      if (!ping.ok) {
        if (ping.status === 401 || ping.status === 403) {
          if (window.Auth) window.Auth.clearPlatformAuth();
          window.location.replace('/platform/login');
          return;
        }
      } else {
        const adminData = ping.data || {};
        const nameEl = document.getElementById('admin-display-name');
        if (nameEl && adminData.email) {
          nameEl.textContent = adminData.email.split('@')[0];
        }
      }
    } catch (e) {
      // Offline or network error
    }

    // 2. Setup user display name from local storage if available
    try {
      const user = JSON.parse(localStorage.getItem('vyavsa_platform_user') || '{}');
      const nameEl = document.getElementById('admin-display-name');
      if (nameEl && (user.username || user.email)) {
        nameEl.textContent = user.username || user.email.split('@')[0];
      }
    } catch (e) {}

    // 3. Setup Events
    initEventHandlers();

    // 4. Navigate to initial path based on current window location
    navigateTo(window.location.pathname, false);
  }

  // Launch on DOM ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initializePlatformPortal);
  } else {
    initializePlatformPortal();
  }

})();
