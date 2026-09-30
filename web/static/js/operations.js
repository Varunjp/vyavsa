/**
 * Vyavsa Operations Management Module
 * Comprehensive Client-Side Controller for Tenant Admin & Tenant User Daily Operations
 */

const Operations = (() => {
  let currentUser = null;
  let currentRole = 'user'; // 'admin' or 'user'
  let currentTenant = null;
  let cachedEmployees = [];
  let cachedCustomers = [];
  let cachedBanks = [];

  function formatCurrency(amount) {
    if (amount === null || amount === undefined || amount === '') return '₹0.00';
    const num = parseFloat(amount);
    if (isNaN(num)) return '₹0.00';
    return '₹' + num.toLocaleString('en-IN', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  }

  function formatDate(dStr) {
    if (!dStr) return '';
    try {
      const parts = dStr.split('T')[0].split('-');
      if (parts.length === 3) {
        const d = new Date(parseInt(parts[0]), parseInt(parts[1]) - 1, parseInt(parts[2]));
        return d.toLocaleDateString('en-IN', { day: '2-digit', month: 'short', year: 'numeric' });
      }
      const d = new Date(dStr);
      if (isNaN(d.getTime())) return dStr;
      return d.toLocaleDateString('en-IN', { day: '2-digit', month: 'short', year: 'numeric' });
    } catch (e) {
      return dStr;
    }
  }

  function formatDateTime(dStr) {
    if (!dStr) return '';
    try {
      const d = new Date(dStr);
      if (isNaN(d.getTime())) return dStr;
      const today = new Date();
      const isToday = d.toDateString() === today.toDateString();
      const timeStr = d.toLocaleTimeString('en-IN', { hour: '2-digit', minute: '2-digit', hour12: true });
      if (isToday) {
        return `Today, ${timeStr}`;
      }
      return `${d.toLocaleDateString('en-IN', { day: '2-digit', month: 'short' })}, ${timeStr}`;
    } catch (e) {
      return dStr;
    }
  }

  function getTodayString() {
    const d = new Date();
    const year = d.getFullYear();
    const month = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  }

  function showToast(message, type = 'success') {
    let container = document.getElementById('toast-container');
    if (!container) {
      container = document.createElement('div');
      container.id = 'toast-container';
      container.style.cssText = 'position:fixed;bottom:24px;right:24px;z-index:9999;display:flex;flex-direction:column;gap:10px;pointer-events:none;';
      document.body.appendChild(container);
    }

    const toast = document.createElement('div');
    toast.className = `toast-msg toast-${type}`;
    const bg = type === 'error' ? '#ef4444' : type === 'warning' ? '#f59e0b' : '#10b981';
    toast.style.cssText = `background:${bg};color:#fff;padding:12px 20px;border-radius:10px;font-size:0.875rem;font-weight:600;box-shadow:0 4px 14px rgba(0,0,0,0.18);display:flex;align-items:center;gap:8px;pointer-events:auto;animation:fadeIn 0.2s ease-out;`;
    toast.innerHTML = `<span>${type === 'error' ? '⚠️' : '✓'}</span><span>${escapeHTML(message)}</span>`;
    container.appendChild(toast);

    setTimeout(() => {
      toast.style.opacity = '0';
      toast.style.transition = 'opacity 0.3s ease';
      setTimeout(() => toast.remove(), 300);
    }, 4000);
  }

  function escapeHTML(str) {
    if (!str) return '';
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
  }

  // Generic modal handlers
  function openModal(modalId) {
    const el = document.getElementById(modalId);
    if (el) {
      el.style.display = 'flex';
      document.body.style.overflow = 'hidden';
    }
  }

  function closeModal(modalId) {
    const el = document.getElementById(modalId);
    if (el) {
      el.style.display = 'none';
      document.body.style.overflow = 'auto';
      const form = el.querySelector('form');
      if (form) form.reset();
      const errEl = el.querySelector('.modal-error-banner');
      if (errEl) errEl.style.display = 'none';
    }
  }

  function setButtonLoading(btn, isLoading, originalText = 'Save') {
    if (!btn) return;
    if (isLoading) {
      btn.dataset.origText = btn.innerHTML;
      btn.disabled = true;
      btn.innerHTML = `<span class="spinner" style="display:inline-block;width:14px;height:14px;border:2px solid #fff;border-top-color:transparent;border-radius:50%;animation:spin 0.6s linear infinite;margin-right:6px;"></span>Saving...`;
    } else {
      btn.disabled = false;
      btn.innerHTML = btn.dataset.origText || originalText;
    }
  }

  function showModalError(modalId, msg) {
    const modal = document.getElementById(modalId);
    if (!modal) return;
    let errEl = modal.querySelector('.modal-error-banner');
    if (!errEl) {
      errEl = document.createElement('div');
      errEl.className = 'modal-error-banner';
      errEl.style.cssText = 'background:#fef2f2;border:1px solid #fecaca;color:#991b1b;padding:10px 14px;border-radius:8px;font-size:0.8125rem;margin-bottom:14px;font-weight:600;display:none;';
      const form = modal.querySelector('form') || modal.querySelector('.modal-body');
      if (form) form.insertBefore(errEl, form.firstChild);
    }
    errEl.textContent = msg;
    errEl.style.display = 'block';
  }

  // Preload dropdown dependencies
  async function loadDependencies() {
    try {
      const [empRes, custRes, bankRes, profRes] = await Promise.all([
        window.API.get('/tenant/employees?page_size=100'),
        window.API.get('/tenant/customers?page_size=100'),
        window.API.get('/tenant/banks?page_size=100'),
        window.API.get('/tenant/profile')
      ]);

      if (empRes.ok && Array.isArray(empRes.data)) cachedEmployees = empRes.data;
      if (custRes.ok && Array.isArray(custRes.data)) cachedCustomers = custRes.data;
      if (bankRes.ok && Array.isArray(bankRes.data)) cachedBanks = bankRes.data;

      if (profRes.ok && profRes.data) {
        currentTenant = profRes.data;
        const bizNameEl = document.getElementById('topbar-business-name');
        if (bizNameEl) bizNameEl.textContent = currentTenant.name || 'Vyavsa Store';
      }
    } catch (e) {
      console.warn('Failed loading dependencies:', e);
    }

    populateSelectDropdowns();
  }

  function populateSelectDropdowns() {
    document.querySelectorAll('.select-employee').forEach(sel => {
      const cur = sel.value;
      sel.innerHTML = '<option value="">Select Employee...</option>' +
        cachedEmployees.map(e => `<option value="${e.id}">${escapeHTML(e.name || 'Staff')}</option>`).join('');
      if (cur) sel.value = cur;
    });

    document.querySelectorAll('.select-customer').forEach(sel => {
      const cur = sel.value;
      sel.innerHTML = '<option value="">Select Customer...</option>' +
        cachedCustomers.map(c => `<option value="${c.id}">${escapeHTML(c.customer_name || 'Customer')}</option>`).join('');
      if (cur) sel.value = cur;
    });

    document.querySelectorAll('.select-bank').forEach(sel => {
      const cur = sel.value;
      sel.innerHTML = '<option value="">None / Cash Only</option>' +
        cachedBanks.map(b => `<option value="${b.id}">${escapeHTML(b.bank_name)} (${escapeHTML(b.account_number || 'Main')})</option>`).join('');
      if (cur) sel.value = cur;
    });
  }

  // ==========================================
  // Section Loaders
  // ==========================================

  // 1. Dashboard Financial Metrics & Stale Data Indicator
  async function loadDashboardMetrics() {
    const res = await window.API.get('/tenant/metrics');
    if (!res.ok) {
      console.warn('Failed to load metrics:', res.error);
      const banner = document.getElementById('data-freshness-banner');
      if (banner) {
        banner.className = 'freshness-banner stale';
        banner.innerHTML = `
          <div class="freshness-left">
            <span class="warning-pulse-icon">⚠️</span>
            <span>Unable to load live dashboard statistics. <button class="btn btn-sm btn-secondary" onclick="Operations.loadDashboardMetrics()" style="padding:0.2rem 0.5rem;font-size:0.75rem;margin-left:0.5rem;">Retry</button></span>
          </div>
        `;
      }
      const attLoaded = document.getElementById('overview-att-loaded');
      const attEmpty = document.getElementById('overview-att-empty');
      if (attLoaded) attLoaded.style.display = 'none';
      if (attEmpty) {
        attEmpty.style.display = 'flex';
        const emptyText = attEmpty.querySelector('.overview-empty-text span:last-child');
        if (emptyText) emptyText.textContent = 'Unable to load attendance';
      }
      const itemLoaded = document.getElementById('overview-item-loaded');
      const itemEmpty = document.getElementById('overview-item-empty');
      if (itemLoaded) itemLoaded.style.display = 'none';
      if (itemEmpty) itemEmpty.style.display = 'flex';
      return;
    }

    const m = res.data;
    if (!m) return;

    // Render Enhanced Today's Overview
    if (m.today_overview) {
      renderTodayOverview(m.today_overview);
    } else {
      window.API.get('/tenant/dashboard/today').then(todayRes => {
        if (todayRes.ok && todayRes.data) {
          renderTodayOverview(todayRes.data);
        }
      }).catch(e => {
        console.warn('Failed to load today overview:', e);
      });
    }

    // Financial Position & Cash/Bank Metrics
    const elNetCash = document.getElementById('kpi-net-cash');
    const elNetBank = document.getElementById('kpi-net-bank');
    const elReceivables = document.getElementById('kpi-receivables');
    const elPayables = document.getElementById('kpi-payables');
    const elPendingSalary = document.getElementById('kpi-pending-salary');

    if (elNetCash) elNetCash.textContent = formatCurrency(m.cash_balance);
    if (elNetBank) elNetBank.textContent = formatCurrency(m.bank_balance);
    if (elReceivables) elReceivables.textContent = formatCurrency(m.total_receivable);
    if (elPayables) elPayables.textContent = formatCurrency(m.net_dues || m.total_payable);
    if (elPendingSalary) elPendingSalary.textContent = formatCurrency(m.pending_salary);

    // Update Cash & Bank and Receivables/Dues tab summaries
    const cbCash = document.getElementById('cb-cash-balance');
    const cbBank = document.getElementById('cb-bank-balance');
    const cbTotal = document.getElementById('cb-total-liquid');
    if (cbCash) cbCash.textContent = formatCurrency(m.cash_balance);
    if (cbBank) cbBank.textContent = formatCurrency(m.bank_balance);
    if (cbTotal) {
      const cNum = parseFloat(m.cash_balance) || 0;
      const bNum = parseFloat(m.bank_balance) || 0;
      cbTotal.textContent = formatCurrency(cNum + bNum);
    }

    const rdRec = document.getElementById('rd-receivables');
    const rdPay = document.getElementById('rd-payables');
    const rdSal = document.getElementById('rd-salary');
    if (rdRec) rdRec.textContent = formatCurrency(m.total_receivable);
    if (rdPay) rdPay.textContent = formatCurrency(m.total_payable);
    if (rdSal) rdSal.textContent = formatCurrency(m.pending_salary);

    // Dynamic Business Date Display in Greeting
    const businessDateBadge = document.getElementById('active-business-date-text');
    const displayDate = m.requested_date || getTodayString();
    if (businessDateBadge) {
      businessDateBadge.textContent = formatDate(displayDate);
    }

    // Dynamic Data Freshness Indicator
    const banner = document.getElementById('data-freshness-banner');
    const todaySub = document.getElementById('today-overview-sub');
    if (banner) {
      if (m.is_current) {
        banner.className = 'freshness-banner fresh';
        banner.innerHTML = `
          <div class="freshness-left">
            <span class="pulse-dot"></span>
            <span><strong>Today's Live Overview:</strong> Updated dynamically for ${formatDate(m.data_date || displayDate)}</span>
          </div>
          <button type="button" class="btn btn-secondary btn-sm" onclick="Operations.loadDashboardMetrics()" style="padding: 0.25rem 0.625rem; font-size: 0.75rem;">
            ↻ Refresh
          </button>
        `;
        if (todaySub) todaySub.textContent = 'Updated today';
      } else {
        const daysText = m.days_old === 1 ? "yesterday's" : formatDate(m.data_date);
        banner.className = 'freshness-banner stale';
        banner.innerHTML = `
          <div class="freshness-left">
            <span class="warning-pulse-icon">⚠️</span>
            <div>
              <strong>Showing ${daysText} data</strong>
              <div style="font-size:0.75rem;opacity:0.9;">Data is ${m.days_old} day(s) old. Today's statistics are not recorded yet.</div>
            </div>
          </div>
          <button type="button" class="btn btn-secondary btn-sm" onclick="Operations.loadDashboardMetrics()" style="padding: 0.25rem 0.625rem; font-size: 0.75rem;">
            ↻ Refresh
          </button>
        `;
        if (todaySub) todaySub.textContent = `Showing previous available (${daysText})`;
      }
    }

    // Today / Active stats
    if (m.today_stats) {
      const ts = m.today_stats;
      const elTodaySales = document.getElementById('kpi-today-sales');
      const elTodayPurchases = document.getElementById('kpi-today-purchases');
      const elTodayExpenses = document.getElementById('kpi-today-expenses');
      const elTodayAttendance = document.getElementById('kpi-today-attendance');
      const elTodayLineSales = document.getElementById('kpi-today-line-sales');
      const elTodayCounterSales = document.getElementById('kpi-today-counter-sales');

      if (elTodaySales) elTodaySales.textContent = formatCurrency(ts.total_sales);
      if (elTodayPurchases) elTodayPurchases.textContent = formatCurrency(ts.purchase_amount);
      if (elTodayExpenses) elTodayExpenses.textContent = formatCurrency(ts.expense_amount);
      if (elTodayAttendance) elTodayAttendance.textContent = `${ts.attendance_present} Present / ${ts.attendance_absent} Absent`;
      if (elTodayLineSales) elTodayLineSales.textContent = formatCurrency(ts.line_sale_amount);
      if (elTodayCounterSales) elTodayCounterSales.textContent = formatCurrency(ts.counter_sale_amount);
    }

    // Today's Employee Salary KPI
    const elTodaySalary = document.getElementById('kpi-today-salary');
    const salaryVal = m.today_employee_salary !== undefined ? m.today_employee_salary : (m.today_stats && m.today_stats.today_employee_salary ? m.today_stats.today_employee_salary : 0);
    if (elTodaySalary) elTodaySalary.textContent = formatCurrency(salaryVal);

    // Render individual bank account balances on Overview
    renderDashboardBankBalances(m.bank_balances);

    // Load dynamic recent activity
    loadRecentActivity();
  }

  // 1A. Render Individual Bank Accounts Widget
  function renderDashboardBankBalances(banks) {
    const container = document.getElementById('overview-bank-balances-container');
    if (!container) return;
    if (!Array.isArray(banks) || banks.length === 0) {
      container.innerHTML = `
        <div style="grid-column: 1/-1; padding: 1.25rem; text-align: center; color: var(--color-text-muted); background: var(--color-surface); border: 1px dashed var(--color-border); border-radius: var(--radius-md); font-size: 0.875rem;">
          No bank accounts registered yet. Click "Add Bank Account" in Cash & Bank tab.
        </div>
      `;
      return;
    }
    container.innerHTML = banks.map(b => `
      <div class="bank-balance-card">
        <div class="bank-card-title">
          <span>${escapeHTML(b.bank_name)}</span>
          <span class="status-badge ${b.status === 'active' ? 'paid' : 'pending'}">${escapeHTML(b.status)}</span>
        </div>
        <div class="bank-card-acc">A/C: ${escapeHTML(b.account_number || '—')}</div>
        <div class="bank-card-balance">${formatCurrency(b.current_balance || 0)}</div>
      </div>
    `).join('');
  }

  // 1A-1. Render Enhanced Today's Overview
  function renderTodayOverview(overview) {
    if (!overview) return;

    // 1. Staff Attendance
    const att = overview.attendance || {};
    const attLoaded = document.getElementById('overview-att-loaded');
    const attEmpty = document.getElementById('overview-att-empty');
    const attPresent = document.getElementById('overview-att-present');
    const attTotal = document.getElementById('overview-att-total');
    const attTrend = document.getElementById('overview-att-trend');
    const attBadge = document.getElementById('overview-att-badge');

    if (att.has_records) {
      if (attLoaded) attLoaded.style.display = 'block';
      if (attEmpty) attEmpty.style.display = 'none';
      if (attPresent) attPresent.textContent = String(att.present !== undefined ? att.present : 0);
      if (attTotal) attTotal.textContent = String(att.total !== undefined ? att.total : (att.present || 0));
      if (attTrend) attTrend.textContent = `${att.present || 0} on duty · ${att.absent || 0} absent`;
      if (attBadge) {
        attBadge.className = 'status-badge cleared';
        attBadge.textContent = 'Active';
      }
    } else {
      if (attLoaded) attLoaded.style.display = 'none';
      if (attEmpty) attEmpty.style.display = 'flex';
      const emptyText = attEmpty ? attEmpty.querySelector('.overview-empty-text span:last-child') : null;
      if (emptyText) {
        if (att.total > 0) {
          emptyText.textContent = `No attendance recorded today (${att.total} staff)`;
        } else {
          emptyText.textContent = 'No staff members registered';
        }
      }
      if (attBadge) {
        attBadge.className = 'status-badge pending';
        attBadge.textContent = 'Not Marked';
      }
    }

    // 2. Line Sale
    const elLineSale = document.getElementById('overview-line-sale-val');
    if (elLineSale) {
      elLineSale.textContent = formatCurrency(overview.line_sale || 0);
    }

    // 3. Counter Sale
    const elCounterSale = document.getElementById('overview-counter-sale-val');
    if (elCounterSale) {
      elCounterSale.textContent = formatCurrency(overview.counter_sale || 0);
    }

    // 4. Employee Total Advance
    const elEmpAdvance = document.getElementById('overview-emp-advance-val');
    const elEmpAdvanceTrend = document.getElementById('overview-emp-advance-trend');
    if (elEmpAdvance) {
      const advAmount = overview.employee_advance || 0;
      elEmpAdvance.textContent = formatCurrency(advAmount);
      if (elEmpAdvanceTrend) {
        const advNum = parseFloat(advAmount) || 0;
        elEmpAdvanceTrend.textContent = advNum > 0 ? 'Aggregated staff advances today' : 'No advances given today';
      }
    }

    // 5. Current Item
    const item = overview.current_item;
    const itemLoaded = document.getElementById('overview-item-loaded');
    const itemEmpty = document.getElementById('overview-item-empty');
    const itemName = document.getElementById('overview-item-name');
    const itemQty = document.getElementById('overview-item-qty');
    const itemTotal = document.getElementById('overview-item-total');
    const itemStatus = document.getElementById('overview-item-status');
    const itemBadge = document.getElementById('overview-item-badge');
    const itemTrend = document.getElementById('overview-item-trend');

    if (item && item.name) {
      if (itemLoaded) itemLoaded.style.display = 'block';
      if (itemEmpty) itemEmpty.style.display = 'none';
      if (itemName) itemName.textContent = item.name;
      if (itemQty) {
        if (item.quantity) {
          itemQty.style.display = 'inline-flex';
          itemQty.textContent = `Qty: ${item.quantity}`;
        } else {
          itemQty.style.display = 'none';
        }
      }
      if (itemTotal) {
        itemTotal.textContent = formatCurrency(item.total_amount || 0);
      }
      if (itemStatus) {
        if (item.is_today) {
          itemStatus.className = 'overview-pill highlight';
          itemStatus.textContent = item.source === 'counter_sale' ? 'Counter Sale · Today' : 'Stock Intake · Today';
        } else {
          itemStatus.className = 'overview-pill';
          itemStatus.textContent = item.date ? formatDate(item.date) : 'Recent Record';
        }
      }
      if (itemBadge) {
        itemBadge.className = 'status-badge cleared';
        itemBadge.textContent = item.source === 'counter_sale' ? 'Counter Sale' : 'Inventory';
      }
      if (itemTrend) {
        itemTrend.textContent = item.is_today ? 'Latest procurement recorded today' : 'Latest available inventory record';
      }
    } else {
      if (itemLoaded) itemLoaded.style.display = 'none';
      if (itemEmpty) itemEmpty.style.display = 'flex';
      if (itemBadge) {
        itemBadge.className = 'status-badge pending';
        itemBadge.textContent = 'No Items';
      }
    }
  }

  // 1B. Dynamic Recent Activity Stream (fetches real backend records)
  async function loadRecentActivity() {
    const tbody = document.getElementById('recent-activity-table-body');
    if (!tbody) return;

    try {
      const [lineRes, countRes, purchRes, expRes] = await Promise.all([
        window.API.get('/tenant/line-sales?page=1&page_size=4'),
        window.API.get('/tenant/counter-sales?page=1&page_size=4'),
        window.API.get('/tenant/purchases?page=1&page_size=4'),
        window.API.get('/tenant/expenses?page=1&page_size=4')
      ]);

      const items = [];

      if (lineRes.ok && Array.isArray(lineRes.data)) {
        lineRes.data.forEach(s => {
          items.push({
            desc: `Line Sale - ${s.customer_name || 'Customer'}${s.route ? ' (' + s.route + ')' : ''}`,
            category: 'Line Sale',
            date: s.created_at,
            amount: '+' + formatCurrency(s.total_amount),
            amountColor: 'var(--color-success)',
            badgeText: parseFloat(s.balance) > 0 ? 'Credit Due' : 'Paid',
            badgeClass: parseFloat(s.balance) > 0 ? 'pending' : 'paid'
          });
        });
      }

      if (countRes.ok && Array.isArray(countRes.data)) {
        countRes.data.forEach(s => {
          items.push({
            desc: `Counter Sale - ${s.item || 'Retail Goods'}`,
            category: 'Counter Sale',
            date: s.created_at,
            amount: '+' + formatCurrency(s.total_amount),
            amountColor: 'var(--color-success)',
            badgeText: 'Completed',
            badgeClass: 'paid'
          });
        });
      }

      if (purchRes.ok && Array.isArray(purchRes.data)) {
        purchRes.data.forEach(p => {
          items.push({
            desc: `Stock Intake - ${p.item || 'Inventory'}`,
            category: 'Purchase',
            date: p.created_at,
            amount: '-' + formatCurrency(p.total_amount),
            amountColor: 'var(--color-error)',
            badgeText: parseFloat(p.total_pending) > 0 ? 'Due Pending' : 'Cleared',
            badgeClass: parseFloat(p.total_pending) > 0 ? 'danger' : 'cleared'
          });
        });
      }

      if (expRes.ok && Array.isArray(expRes.data)) {
        expRes.data.forEach(e => {
          items.push({
            desc: `Expense - ${e.item || 'Operating Cost'}`,
            category: 'Expense',
            date: e.created_at,
            amount: '-' + formatCurrency(e.total_amount),
            amountColor: 'var(--color-error)',
            badgeText: 'Recorded',
            badgeClass: 'cleared'
          });
        });
      }

      // Sort by date descending
      items.sort((a, b) => new Date(b.date) - new Date(a.date));

      if (items.length === 0) {
        tbody.innerHTML = `
          <tr>
            <td colspan="5">
              <div class="empty-state-box">
                <div class="empty-state-icon">📋</div>
                <div class="empty-state-title">No transactions recorded yet today</div>
                <div class="empty-state-desc">Record route line sales, walk-in counter sales, or operating expenses to start tracking.</div>
                <div style="display:flex;gap:0.5rem;justify-content:center;flex-wrap:wrap;">
                  <button type="button" class="btn btn-primary btn-sm" onclick="Operations.openModal('modal-add-line-sale')">+ Add Line Sale</button>
                  <button type="button" class="btn btn-secondary btn-sm" onclick="Operations.openModal('modal-add-counter-sale')">+ Add Counter Sale</button>
                </div>
              </div>
            </td>
          </tr>
        `;
        return;
      }

      const displayList = items.slice(0, 6);
      tbody.innerHTML = displayList.map(item => `
        <tr>
          <td><strong>${escapeHTML(item.desc)}</strong></td>
          <td><span class="status-badge cleared">${escapeHTML(item.category)}</span></td>
          <td style="color:var(--color-text-muted);font-size:0.8125rem;">${formatDateTime(item.date)}</td>
          <td style="color:${item.amountColor};font-weight:700;font-family:var(--font-sans);">${item.amount}</td>
          <td><span class="status-badge ${item.badgeClass}">${item.badgeText}</span></td>
        </tr>
      `).join('');
    } catch (e) {
      console.warn('Failed loading recent activity:', e);
    }
  }

  // 2. Line Sales
  async function loadLineSales(page = 1) {
    const dateInput = document.getElementById('filter-line-sales-date');
    const date = dateInput ? dateInput.value : '';
    const search = document.getElementById('search-line-sales')?.value || '';
    const res = await window.API.get(`/tenant/line-sales?page=${page}&page_size=20&date=${date}&search=${encodeURIComponent(search)}`);
    const tbody = document.getElementById('line-sales-table-body');
    if (!tbody) return;

    if (!res.ok || !Array.isArray(res.data) || res.data.length === 0) {
      tbody.innerHTML = `
        <tr>
          <td colspan="8">
            <div class="empty-state-box">
              <div class="empty-state-icon">🚚</div>
              <div class="empty-state-title">No line sales found for this date</div>
              <div class="empty-state-desc">Record route distribution deliveries, cash-in collections, and customer credit balance updates.</div>
              <button type="button" class="btn btn-primary btn-sm" onclick="Operations.openModal('modal-add-line-sale')">+ Record Line Sale</button>
            </div>
          </td>
        </tr>
      `;
      return;
    }

    tbody.innerHTML = res.data.map(s => {
      const balance = parseFloat(s.balance) || 0;
      const isAdmin = currentRole === 'admin';
      return `
        <tr>
          <td><strong>${escapeHTML(s.customer_name || 'Customer')}</strong></td>
          <td>${escapeHTML(s.route || '—')}</td>
          <td>${escapeHTML(s.salesman || '—')}</td>
          <td>${formatDate(s.created_at)}</td>
          <td style="font-weight:700;">${formatCurrency(s.total_amount)}</td>
          <td style="color:var(--color-success);font-weight:600;">${formatCurrency(s.total_cash_in)}</td>
          <td style="color:${balance > 0 ? 'var(--color-warning)' : 'var(--color-text-muted)'};font-weight:600;">${formatCurrency(s.balance)}</td>
          <td style="text-align:right;">
            <div style="display:inline-flex;gap:0.375rem;justify-content:flex-end;">
              <button class="btn btn-sm btn-secondary" onclick="Operations.viewLineSale('${s.id}')">View</button>
              ${isAdmin ? `
                <button class="btn btn-sm btn-secondary" onclick="Operations.openEditLineSale('${s.id}')">Edit</button>
                <button class="btn btn-sm btn-secondary" onclick="Operations.deleteLineSale('${s.id}')" style="color:var(--color-error);">Delete</button>
              ` : ''}
            </div>
          </td>
        </tr>
      `;
    }).join('');
  }

  // 3. Counter Sales
  async function loadCounterSales(page = 1) {
    const dateInput = document.getElementById('filter-counter-sales-date');
    const date = dateInput ? dateInput.value : '';
    const res = await window.API.get(`/tenant/counter-sales?page=${page}&page_size=20&date=${date}`);
    const tbody = document.getElementById('counter-sales-table-body');
    if (!tbody) return;

    if (!res.ok || !Array.isArray(res.data) || res.data.length === 0) {
      tbody.innerHTML = `
        <tr>
          <td colspan="7">
            <div class="empty-state-box">
              <div class="empty-state-icon">🛒</div>
              <div class="empty-state-title">No counter sales found for this date</div>
              <div class="empty-state-desc">Record over-the-counter shop billing, instant cash payments, and QR/UPI settlements.</div>
              <button type="button" class="btn btn-primary btn-sm" onclick="Operations.openModal('modal-add-counter-sale')">+ Record Counter Sale</button>
            </div>
          </td>
        </tr>
      `;
      return;
    }

    tbody.innerHTML = res.data.map(s => {
      const isAdmin = currentRole === 'admin';
      return `
        <tr>
          <td><strong>${escapeHTML(s.item || 'Item')}</strong></td>
          <td>${formatDate(s.created_at)}</td>
          <td><span class="status-badge cleared">${escapeHTML(s.payment_method || 'Cash')}</span></td>
          <td style="font-weight:700;">${formatCurrency(s.total_amount)}</td>
          <td style="color:var(--color-success);font-weight:600;">${formatCurrency(s.cash)}</td>
          <td style="color:var(--color-primary);font-weight:600;">${formatCurrency(s.account)}</td>
          <td style="text-align:right;">
            <div style="display:inline-flex;gap:0.375rem;justify-content:flex-end;">
              <button class="btn btn-sm btn-secondary" onclick="Operations.viewCounterSale('${s.id}')">View</button>
              ${isAdmin ? `
                <button class="btn btn-sm btn-secondary" onclick="Operations.openEditCounterSale('${s.id}')">Edit</button>
                <button class="btn btn-sm btn-secondary" onclick="Operations.deleteCounterSale('${s.id}')" style="color:var(--color-error);">Delete</button>
              ` : ''}
            </div>
          </td>
        </tr>
      `;
    }).join('');
  }

  // 4. Staff Attendance
  async function loadAttendance(page = 1) {
    const dateInput = document.getElementById('filter-attendance-date');
    const date = dateInput ? dateInput.value : '';
    const res = await window.API.get(`/tenant/attendance?page=${page}&page_size=20&date=${date}`);
    const tbody = document.getElementById('attendance-table-body');
    if (!tbody) return;

    if (!res.ok || !Array.isArray(res.data) || res.data.length === 0) {
      tbody.innerHTML = `
        <tr>
          <td colspan="6">
            <div class="empty-state-box">
              <div class="empty-state-icon">📅</div>
              <div class="empty-state-title">No attendance marked for this date</div>
              <div class="empty-state-desc">Log employee daily attendance status, overtime hours, and cash advances.</div>
              <button type="button" class="btn btn-primary btn-sm" onclick="Operations.openModal('modal-mark-attendance')">+ Mark Attendance</button>
            </div>
          </td>
        </tr>
      `;
      return;
    }

    tbody.innerHTML = res.data.map(a => {
      const isPresent = a.status === 'present';
      const isAbsent = a.status === 'absent';
      const badgeCls = isPresent ? 'paid' : isAbsent ? 'danger' : 'pending';
      return `
        <tr>
          <td><strong>${escapeHTML(a.employee_name || 'Staff')}</strong></td>
          <td>${formatDate(a.date)}</td>
          <td><span class="status-badge ${badgeCls}">${escapeHTML(a.status.toUpperCase())}</span></td>
          <td>${parseFloat(a.ot) > 0 ? a.ot + ' hrs' : '—'}</td>
          <td style="color:${parseFloat(a.advance) > 0 ? 'var(--color-error)' : 'inherit'};font-weight:600;">${parseFloat(a.advance) > 0 ? formatCurrency(a.advance) : '—'}</td>
          <td style="text-align:right;">
            <button class="btn btn-sm btn-secondary" onclick="Operations.openEditAttendance('${a.id}')">Edit</button>
          </td>
        </tr>
      `;
    }).join('');
  }

  // 5. Purchases
  async function loadPurchases(page = 1) {
    const dateInput = document.getElementById('filter-purchases-date');
    const date = dateInput ? dateInput.value : '';
    const res = await window.API.get(`/tenant/purchases?page=${page}&page_size=20&date=${date}`);
    const tbody = document.getElementById('purchases-table-body');
    if (!tbody) return;

    if (!res.ok || !Array.isArray(res.data) || res.data.length === 0) {
      tbody.innerHTML = `
        <tr>
          <td colspan="7">
            <div class="empty-state-box">
              <div class="empty-state-icon">📦</div>
              <div class="empty-state-title">No purchase records found</div>
              <div class="empty-state-desc">Record raw materials, vendor invoices, stock intake, and supplier dues.</div>
              <button type="button" class="btn btn-primary btn-sm" onclick="Operations.openModal('modal-add-purchase')">+ Record Purchase</button>
            </div>
          </td>
        </tr>
      `;
      return;
    }

    tbody.innerHTML = res.data.map(p => {
      const pending = parseFloat(p.total_pending) || 0;
      const isAdmin = currentRole === 'admin';
      return `
        <tr>
          <td><strong>${escapeHTML(p.item || 'Item')}</strong></td>
          <td>${p.quantity || 1}</td>
          <td>${formatDate(p.created_at)}</td>
          <td style="font-weight:700;">${formatCurrency(p.total_amount)}</td>
          <td style="color:var(--color-success);font-weight:600;">${formatCurrency(p.total_paid)}</td>
          <td style="color:${pending > 0 ? 'var(--color-error)' : 'var(--color-text-muted)'};font-weight:600;">${formatCurrency(p.total_pending)}</td>
          <td style="text-align:right;">
            <div style="display:inline-flex;gap:0.375rem;justify-content:flex-end;">
              ${isAdmin ? `
                <button class="btn btn-sm btn-secondary" onclick="Operations.openEditPurchase('${p.id}')">Edit</button>
                <button class="btn btn-sm btn-secondary" onclick="Operations.deletePurchase('${p.id}')" style="color:var(--color-error);">Delete</button>
              ` : '<span style="color:var(--color-text-subtle);font-size:0.75rem;">Recorded</span>'}
            </div>
          </td>
        </tr>
      `;
    }).join('');
  }

  // 6. Expenses
  async function loadExpenses(page = 1) {
    const dateInput = document.getElementById('filter-expenses-date');
    const date = dateInput ? dateInput.value : '';
    const res = await window.API.get(`/tenant/expenses?page=${page}&page_size=20&date=${date}`);
    const tbody = document.getElementById('expenses-table-body');
    if (!tbody) return;

    if (!res.ok || !Array.isArray(res.data) || res.data.length === 0) {
      tbody.innerHTML = `
        <tr>
          <td colspan="4">
            <div class="empty-state-box">
              <div class="empty-state-icon">⚡</div>
              <div class="empty-state-title">No expenses recorded for this date</div>
              <div class="empty-state-desc">Record utility bills, shop rent, fuel, transport, and miscellaneous cash disbursements.</div>
              <button type="button" class="btn btn-primary btn-sm" onclick="Operations.openModal('modal-add-expense')">+ Record Expense</button>
            </div>
          </td>
        </tr>
      `;
      return;
    }

    tbody.innerHTML = res.data.map(e => {
      const isAdmin = currentRole === 'admin';
      return `
        <tr>
          <td><strong>${escapeHTML(e.item || 'Expense')}</strong></td>
          <td>${formatDate(e.created_at)}</td>
          <td style="font-weight:700;color:var(--color-error);">${formatCurrency(e.total_amount)}</td>
          <td style="text-align:right;">
            <div style="display:inline-flex;gap:0.375rem;justify-content:flex-end;">
              ${isAdmin ? `
                <button class="btn btn-sm btn-secondary" onclick="Operations.openEditExpense('${e.id}')">Edit</button>
                <button class="btn btn-sm btn-secondary" onclick="Operations.deleteExpense('${e.id}')" style="color:var(--color-error);">Delete</button>
              ` : '<span style="color:var(--color-text-subtle);font-size:0.75rem;">Recorded</span>'}
            </div>
          </td>
        </tr>
      `;
    }).join('');
  }

  // 7. Cash & Bank
  async function loadCashBank() {
    loadBanks();
  }

  // 8. Receivables & Dues
  async function loadReceivablesDues() {
    const res = await window.API.get('/tenant/customers?page_size=50');
    const tbody = document.getElementById('rd-customers-table-body');
    if (!tbody) return;

    if (!res.ok || !Array.isArray(res.data) || res.data.length === 0) {
      tbody.innerHTML = `<tr><td colspan="5" style="text-align:center;color:var(--color-text-muted);padding:2rem;">No customer credit ledgers found.</td></tr>`;
      return;
    }

    const creditCustomers = res.data.filter(c => parseFloat(c.current_balance) > 0);
    if (creditCustomers.length === 0) {
      tbody.innerHTML = `<tr><td colspan="5" style="text-align:center;color:var(--color-text-muted);padding:2rem;">All customer balances are currently cleared (₹0.00 dues).</td></tr>`;
      return;
    }

    tbody.innerHTML = creditCustomers.map(c => `
      <tr>
        <td><strong>${escapeHTML(c.customer_name)}</strong></td>
        <td>${escapeHTML(c.phone || '—')}</td>
        <td>${formatCurrency(c.opening_balance)}</td>
        <td style="color:var(--color-warning);font-weight:700;">${formatCurrency(c.current_balance)}</td>
        <td><span class="status-badge pending">Credit Due</span></td>
      </tr>
    `).join('');
  }

  // 9. Tenant Users (Admin Only)
  async function loadTenantUsers(page = 1) {
    if (currentRole !== 'admin') return;
    const search = document.getElementById('search-users')?.value || '';
    const res = await window.API.get(`/tenant/users?page=${page}&page_size=20&search=${encodeURIComponent(search)}`);
    const tbody = document.getElementById('users-table-body');
    if (!tbody) return;

    if (!res.ok || !Array.isArray(res.data) || res.data.length === 0) {
      tbody.innerHTML = `<tr><td colspan="5" style="text-align:center;color:var(--color-text-muted);padding:2rem;">No tenant users found.</td></tr>`;
      return;
    }

    tbody.innerHTML = res.data.map(u => `
      <tr>
        <td><strong>${escapeHTML(u.name)}</strong></td>
        <td>${escapeHTML(u.email)}</td>
        <td><span class="status-badge ${u.role === 'admin' ? 'paid' : 'cleared'}">${escapeHTML(u.role.toUpperCase())}</span></td>
        <td><span class="status-badge ${u.status === 'active' ? 'paid' : 'pending'}">${escapeHTML(u.status)}</span></td>
        <td style="text-align:right;">
          <div style="display:inline-flex;gap:0.375rem;justify-content:flex-end;">
            <button class="btn btn-sm btn-secondary" onclick="Operations.openEditUser('${u.id}')">Edit</button>
            <button class="btn btn-sm btn-secondary" onclick="Operations.deleteUser('${u.id}')" style="color:var(--color-error);">Delete</button>
          </div>
        </td>
      </tr>
    `).join('');
  }

  // 10. Employees (Admin Only)
  async function loadEmployees(page = 1) {
    if (currentRole !== 'admin') return;
    const search = document.getElementById('search-employees')?.value || '';
    const res = await window.API.get(`/tenant/employees?page=${page}&page_size=20&search=${encodeURIComponent(search)}`);
    const tbody = document.getElementById('employees-table-body');
    if (!tbody) return;

    if (!res.ok || !Array.isArray(res.data) || res.data.length === 0) {
      tbody.innerHTML = `<tr><td colspan="6" style="text-align:center;color:var(--color-text-muted);padding:2rem;">No employees registered. Click "Onboard Employee" to add staff.</td></tr>`;
      return;
    }

    cachedEmployees = res.data;
    populateSelectDropdowns();

    tbody.innerHTML = res.data.map(e => `
      <tr>
        <td><strong>${escapeHTML(e.name)}</strong></td>
        <td>${escapeHTML(e.phone || '—')}</td>
        <td style="font-weight:600;">${formatCurrency(e.salary)}</td>
        <td>${formatCurrency(e.ot_rate)}/hr</td>
        <td><span class="status-badge ${e.status === 'active' ? 'paid' : 'pending'}">${escapeHTML(e.status)}</span></td>
        <td style="text-align:right;">
          <div style="display:inline-flex;gap:0.375rem;justify-content:flex-end;">
            <button class="btn btn-sm btn-secondary" onclick="Operations.viewEmployee('${e.id}')">View</button>
            <button class="btn btn-sm btn-secondary" onclick="Operations.openEditEmployee('${e.id}')">Edit</button>
            <button class="btn btn-sm btn-secondary" onclick="Operations.deleteEmployee('${e.id}')" style="color:var(--color-error);">Delete</button>
          </div>
        </td>
      </tr>
    `).join('');
  }

  // 11. Customers (Admin Only)
  async function loadCustomers(page = 1) {
    if (currentRole !== 'admin') return;
    const search = document.getElementById('search-customers')?.value || '';
    const res = await window.API.get(`/tenant/customers?page=${page}&page_size=20&search=${encodeURIComponent(search)}`);
    const tbody = document.getElementById('customers-table-body');
    if (!tbody) return;

    if (!res.ok || !Array.isArray(res.data) || res.data.length === 0) {
      tbody.innerHTML = `<tr><td colspan="6" style="text-align:center;color:var(--color-text-muted);padding:2rem;">No customers found. Click "Add Customer" to create an account.</td></tr>`;
      return;
    }

    cachedCustomers = res.data;
    populateSelectDropdowns();

    tbody.innerHTML = res.data.map(c => `
      <tr>
        <td><strong>${escapeHTML(c.customer_name)}</strong></td>
        <td>${escapeHTML(c.phone || '—')}</td>
        <td>${formatCurrency(c.opening_balance)}</td>
        <td style="font-weight:700;color:${parseFloat(c.current_balance) > 0 ? 'var(--color-warning)' : 'inherit'};">${formatCurrency(c.current_balance)}</td>
        <td><span class="status-badge ${c.status === 'active' ? 'paid' : 'pending'}">${escapeHTML(c.status)}</span></td>
        <td style="text-align:right;">
          <div style="display:inline-flex;gap:0.375rem;justify-content:flex-end;">
            <button class="btn btn-sm btn-secondary" onclick="Operations.viewCustomer('${c.id}')">View</button>
            <button class="btn btn-sm btn-primary" onclick="Operations.openAdjustCustomerBalance('${c.id}')" title="Adjust Balance">Adjust</button>
            <button class="btn btn-sm btn-secondary" onclick="Operations.openEditCustomer('${c.id}')">Edit</button>
            <button class="btn btn-sm btn-secondary" onclick="Operations.deleteCustomer('${c.id}')" style="color:var(--color-error);">Delete</button>
          </div>
        </td>
      </tr>
    `).join('');
  }

  // 12. Banks
  async function loadBanks() {
    const res = await window.API.get('/tenant/banks?page_size=50');
    const tbodies = [
      document.getElementById('banks-table-body'),
      document.getElementById('cb-banks-table-body')
    ];

    if (!res.ok || !Array.isArray(res.data) || res.data.length === 0) {
      tbodies.forEach(tbody => {
        if (tbody) tbody.innerHTML = `<tr><td colspan="6" style="text-align:center;color:var(--color-text-muted);padding:2rem;">No bank accounts registered.</td></tr>`;
      });
      return;
    }

    cachedBanks = res.data;
    populateSelectDropdowns();

    const isAdmin = currentRole === 'admin';
    const rows = res.data.map(b => `
      <tr>
        <td><strong>${escapeHTML(b.bank_name)}</strong></td>
        <td style="font-family:var(--font-mono);">${escapeHTML(b.account_number || '—')}</td>
        <td style="font-family:var(--font-mono);">${escapeHTML(b.ifsc || '—')}</td>
        <td style="font-weight:700;color:var(--color-primary);">${formatCurrency(b.current_balance || 0)}</td>
        <td><span class="status-badge ${b.status === 'active' ? 'paid' : 'pending'}">${escapeHTML(b.status)}</span></td>
        ${isAdmin ? `
          <td style="text-align:right;">
            <div style="display:inline-flex;gap:0.375rem;justify-content:flex-end;">
              <button class="btn btn-sm btn-secondary" onclick="Operations.openEditBank('${b.id}')">Edit</button>
              <button class="btn btn-sm btn-secondary" onclick="Operations.deleteBank('${b.id}')" style="color:var(--color-error);">Delete</button>
            </div>
          </td>
        ` : ''}
      </tr>
    `).join('');

    tbodies.forEach(tbody => {
      if (tbody) tbody.innerHTML = rows;
    });
  }

  // 13. Salaries (Admin Only)
  async function loadSalaries() {
    if (currentRole !== 'admin') return;

    // Load Pending Salaries
    const pendingRes = await window.API.get('/tenant/salaries/pending');
    const pendingTbody = document.getElementById('pending-salaries-table-body');
    if (pendingTbody) {
      if (!pendingRes.ok || !Array.isArray(pendingRes.data) || pendingRes.data.length === 0) {
        pendingTbody.innerHTML = `<tr><td colspan="4" style="text-align:center;color:var(--color-text-muted);padding:1.5rem;">No staff members have pending accrued salary balances.</td></tr>`;
      } else {
        pendingTbody.innerHTML = pendingRes.data.map(ps => `
          <tr>
            <td><strong>${escapeHTML(ps.employee_name || 'Staff')}</strong></td>
            <td>${formatCurrency(ps.salary_rate)}</td>
            <td style="font-weight:700;color:var(--color-warning);">${formatCurrency(ps.balance)}</td>
            <td style="text-align:right;">
              <button class="btn btn-sm btn-primary" onclick="Operations.openPaySalary('${ps.employee_id}', '${escapeHTML(ps.employee_name)}', '${ps.balance}')">
                Disburse Payout →
              </button>
            </td>
          </tr>
        `).join('');
      }
    }

    // Load All Salaries
    const allRes = await window.API.get('/tenant/salaries?page_size=50');
    const allTbody = document.getElementById('salaries-table-body');
    if (allTbody) {
      if (!allRes.ok || !Array.isArray(allRes.data) || allRes.data.length === 0) {
        allTbody.innerHTML = `<tr><td colspan="5" style="text-align:center;color:var(--color-text-muted);padding:1.5rem;">No staff payroll ledgers found.</td></tr>`;
      } else {
        allTbody.innerHTML = allRes.data.map(s => `
          <tr>
            <td><strong>${escapeHTML(s.employee_name || 'Staff')}</strong></td>
            <td>${formatCurrency(s.salary_rate)}</td>
            <td>${formatCurrency(s.ot_rate)}/hr</td>
            <td style="font-weight:700;">${formatCurrency(s.balance)}</td>
            <td style="text-align:right;">
              <div style="display:inline-flex;gap:0.375rem;justify-content:flex-end;">
                <button class="btn btn-sm btn-secondary" onclick="Operations.openPaySalary('${s.employee_id}', '${escapeHTML(s.employee_name)}', '${s.balance}')">Pay</button>
                <button class="btn btn-sm btn-secondary" onclick="Operations.openAdjustSalary('${s.employee_id}', '${escapeHTML(s.employee_name)}', '${s.balance}')">Adjust Rate</button>
              </div>
            </td>
          </tr>
        `).join('');
      }
    }
  }

  // 14. Profile & Organization
  async function loadProfile() {
    try {
      const res = await window.API.get('/tenant/profile');
      if (res.ok && res.data) {
        const t = res.data;
        document.getElementById('profile-biz-name').textContent = t.name || '—';
        document.getElementById('profile-biz-email').textContent = t.email || '—';
        document.getElementById('profile-biz-phone').textContent = t.phone || '—';
        document.getElementById('profile-biz-status').textContent = t.status || 'active';
        document.getElementById('profile-biz-id').textContent = t.id || '—';
      }

      if (currentUser) {
        document.getElementById('profile-user-name').textContent = currentUser.name || 'Business User';
        document.getElementById('profile-user-email').textContent = currentUser.email || '—';
        const roleEl = document.getElementById('profile-user-role');
        roleEl.textContent = currentUser.role === 'admin' ? 'Tenant Administrator' : 'Tenant Operator';
        roleEl.className = 'status-badge ' + (currentUser.role === 'admin' ? 'paid' : 'cleared');
      }

      const subRes = await window.API.get('/tenant/subscription');
      if (subRes.ok && subRes.data) {
        document.getElementById('profile-plan-name').textContent = subRes.data.plan_name || 'Active Plan';
        document.getElementById('profile-plan-status').textContent = subRes.data.status || 'active';
      }
    } catch (e) {
      console.warn('Failed loading profile:', e);
    }
  }

  // ==========================================
  // Form Submit Handlers (Create & Edit)
  // ==========================================

  // Tenant User: Create
  async function submitAddUser(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-user');
    setButtonLoading(btn, true);

    const payload = {
      name: document.getElementById('user-form-name').value.trim(),
      email: document.getElementById('user-form-email').value.trim(),
      password: document.getElementById('user-form-password').value,
      role: document.getElementById('user-form-role').value
    };

    const res = await window.API.post('/tenant/users', payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-add-user', res.error);
      return;
    }

    closeModal('modal-add-user');
    showToast('Tenant user created successfully');
    loadTenantUsers();
  }

  // Tenant User: Edit
  async function openEditUser(id) {
    const res = await window.API.get(`/tenant/users/${id}`);
    if (!res.ok || !res.data) {
      showToast(res.error || 'Failed to fetch user', 'error');
      return;
    }
    const u = res.data;
    document.getElementById('edit-user-id').value = u.id;
    document.getElementById('edit-user-name').value = u.name || '';
    document.getElementById('edit-user-role').value = u.role || 'user';
    document.getElementById('edit-user-status').value = u.status || 'active';
    document.getElementById('edit-user-password').value = '';
    openModal('modal-edit-user');
  }

  async function submitEditUser(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-edit-user');
    const id = document.getElementById('edit-user-id').value;
    setButtonLoading(btn, true);

    const payload = {
      name: document.getElementById('edit-user-name').value.trim(),
      role: document.getElementById('edit-user-role').value,
      status: document.getElementById('edit-user-status').value
    };
    const pw = document.getElementById('edit-user-password').value;
    if (pw) payload.password = pw;

    const res = await window.API.put(`/tenant/users/${id}`, payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-edit-user', res.error);
      return;
    }

    closeModal('modal-edit-user');
    showToast('User updated successfully');
    loadTenantUsers();
  }

  // Employee: Create
  async function submitAddEmployee(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-employee');
    setButtonLoading(btn, true);

    const payload = {
      name: document.getElementById('emp-form-name').value.trim(),
      phone: document.getElementById('emp-form-phone').value.trim(),
      salary: parseFloat(document.getElementById('emp-form-salary').value) || 0,
      ot_rate: parseFloat(document.getElementById('emp-form-ot').value) || 0
    };

    const res = await window.API.post('/tenant/employees', payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-add-employee', res.error);
      return;
    }

    closeModal('modal-add-employee');
    showToast('Employee onboarded successfully');
    loadEmployees();
    loadDependencies();
  }

  // Employee: Edit
  async function openEditEmployee(id) {
    const res = await window.API.get(`/tenant/employees/${id}`);
    if (!res.ok || !res.data) {
      showToast(res.error || 'Failed to fetch employee', 'error');
      return;
    }
    const e = res.data;
    document.getElementById('edit-emp-id').value = e.id;
    document.getElementById('edit-emp-name').value = e.name || '';
    document.getElementById('edit-emp-phone').value = e.phone || '';
    document.getElementById('edit-emp-salary').value = e.salary || '0';
    document.getElementById('edit-emp-ot').value = e.ot_rate || '0';
    document.getElementById('edit-emp-status').value = e.status || 'active';
    openModal('modal-edit-employee');
  }

  async function submitEditEmployee(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-edit-employee');
    const id = document.getElementById('edit-emp-id').value;
    setButtonLoading(btn, true);

    const salary = parseFloat(document.getElementById('edit-emp-salary').value) || 0;
    const otRate = parseFloat(document.getElementById('edit-emp-ot').value) || 0;

    const payload = {
      name: document.getElementById('edit-emp-name').value.trim(),
      phone: document.getElementById('edit-emp-phone').value.trim(),
      salary: salary,
      ot_rate: otRate,
      status: document.getElementById('edit-emp-status').value
    };

    const res = await window.API.put(`/tenant/employees/${id}`, payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-edit-employee', res.error);
      return;
    }

    closeModal('modal-edit-employee');
    showToast('Employee updated successfully');
    loadEmployees();
  }

  // Customer: Create
  async function submitAddCustomer(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-customer');
    setButtonLoading(btn, true);

    const payload = {
      customer_name: document.getElementById('cust-form-name').value.trim(),
      phone: document.getElementById('cust-form-phone').value.trim(),
      opening_balance: parseFloat(document.getElementById('cust-form-opening').value) || 0
    };

    const res = await window.API.post('/tenant/customers', payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-add-customer', res.error);
      return;
    }

    closeModal('modal-add-customer');
    showToast('Customer added successfully');
    loadCustomers();
    loadDependencies();
  }

  // Customer: Edit
  async function openEditCustomer(id) {
    const res = await window.API.get(`/tenant/customers/${id}`);
    if (!res.ok || !res.data) {
      showToast(res.error || 'Failed to fetch customer', 'error');
      return;
    }
    const c = res.data;
    document.getElementById('edit-cust-id').value = c.id;
    document.getElementById('edit-cust-name').value = c.customer_name || '';
    document.getElementById('edit-cust-phone').value = c.phone || '';
    document.getElementById('edit-cust-status').value = c.status || 'active';
    openModal('modal-edit-customer');
  }

  async function submitEditCustomer(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-edit-customer');
    const id = document.getElementById('edit-cust-id').value;
    setButtonLoading(btn, true);

    const payload = {
      customer_name: document.getElementById('edit-cust-name').value.trim(),
      phone: document.getElementById('edit-cust-phone').value.trim(),
      status: document.getElementById('edit-cust-status').value
    };

    const res = await window.API.put(`/tenant/customers/${id}`, payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-edit-customer', res.error);
      return;
    }

    closeModal('modal-edit-customer');
    showToast('Customer updated successfully');
    loadCustomers();
  }

  // Customer: Balance Adjustment
  let activeAdjustCustomer = null;

  async function openAdjustCustomerBalance(id) {
    const res = await window.API.get(`/tenant/customers/${id}`);
    if (!res.ok || !res.data) {
      showToast(res.error || 'Failed to load customer', 'error');
      return;
    }
    const c = res.data;
    activeAdjustCustomer = c;

    document.getElementById('adj-cust-id').value = c.id;
    document.getElementById('adj-cust-name').textContent = c.customer_name;
    document.getElementById('adj-cust-current-bal').textContent = formatCurrency(c.current_balance || 0);

    // Default to new_balance
    document.getElementById('adj-cust-type').value = 'new_balance';
    document.getElementById('adj-cust-amount').value = parseFloat(c.current_balance || 0).toFixed(2);
    document.getElementById('adj-cust-reason').value = '';
    handleCustomerAdjustmentTypeChange();
    calcCustomerAdjustmentPreview();

    // Load recent adjustment history
    const histRes = await window.API.get(`/tenant/customers/${id}/adjustments?page=1&page_size=5`);
    const histSec = document.getElementById('adj-cust-history-section');
    const histList = document.getElementById('adj-cust-history-list');
    if (histSec && histList) {
      if (histRes.ok && Array.isArray(histRes.data) && histRes.data.length > 0) {
        histSec.style.display = 'block';
        histList.innerHTML = histRes.data.map(a => `
          <div style="display:flex;justify-content:space-between;padding:0.35rem 0;border-bottom:1px solid var(--color-border);">
            <div>
              <span style="font-weight:600;">${escapeHTML(a.reason || 'Manual Adjustment')}</span>
              <div style="font-size:0.7rem;color:var(--color-text-muted);">${formatDate(a.created_at)}</div>
            </div>
            <div style="text-align:right;">
              <span style="font-weight:700;color:${parseFloat(a.adjustment_amount) >= 0 ? 'var(--color-warning)' : 'var(--color-success)'};">
                ${parseFloat(a.adjustment_amount) >= 0 ? '+' : ''}${formatCurrency(a.adjustment_amount)}
              </span>
              <div style="font-size:0.7rem;color:var(--color-text-muted);">Bal: ${formatCurrency(a.new_balance)}</div>
            </div>
          </div>
        `).join('');
      } else {
        histSec.style.display = 'none';
        histList.innerHTML = '';
      }
    }

    openModal('modal-adjust-customer-balance');
  }

  function handleCustomerAdjustmentTypeChange() {
    const type = document.getElementById('adj-cust-type')?.value;
    const label = document.getElementById('adj-cust-amount-label');
    const hint = document.getElementById('adj-cust-hint');
    if (type === 'new_balance') {
      if (label) label.textContent = 'New Balance (₹) *';
      if (hint) hint.textContent = 'Enter the target total balance for this customer.';
    } else {
      if (label) label.textContent = 'Adjustment Amount (₹) *';
      if (hint) hint.textContent = 'Positive (+) to increase customer dues, negative (-) to decrease/waive.';
    }
    calcCustomerAdjustmentPreview();
  }

  function calcCustomerAdjustmentPreview() {
    if (!activeAdjustCustomer) return;
    const current = parseFloat(activeAdjustCustomer.current_balance) || 0;
    const type = document.getElementById('adj-cust-type')?.value;
    const inputVal = parseFloat(document.getElementById('adj-cust-amount')?.value);

    const previewValEl = document.getElementById('adj-cust-preview-val');
    const previewDiffEl = document.getElementById('adj-cust-preview-diff');

    if (isNaN(inputVal)) {
      if (previewValEl) previewValEl.textContent = formatCurrency(current);
      if (previewDiffEl) previewDiffEl.textContent = 'Net change: ₹0.00';
      return;
    }

    let targetBal = 0;
    let diff = 0;
    if (type === 'new_balance') {
      targetBal = inputVal;
      diff = targetBal - current;
    } else {
      diff = inputVal;
      targetBal = current + diff;
    }

    if (previewValEl) previewValEl.textContent = formatCurrency(targetBal);
    if (previewDiffEl) {
      const sign = diff > 0 ? '+' : '';
      previewDiffEl.textContent = `Net change: ${sign}${formatCurrency(diff)} (Previous: ${formatCurrency(current)})`;
      previewDiffEl.style.color = diff > 0 ? 'var(--color-warning)' : (diff < 0 ? 'var(--color-success)' : 'var(--color-text-muted)');
    }
  }

  async function submitAdjustCustomerBalance(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-adjust-customer');
    const id = document.getElementById('adj-cust-id').value;
    const type = document.getElementById('adj-cust-type').value;
    const amountVal = parseFloat(document.getElementById('adj-cust-amount').value);
    const reason = document.getElementById('adj-cust-reason').value.trim();

    if (isNaN(amountVal)) {
      showModalError('modal-adjust-customer-balance', 'Please enter a valid amount.');
      return;
    }
    if (!reason) {
      showModalError('modal-adjust-customer-balance', 'Please provide a reason for the adjustment.');
      return;
    }

    setButtonLoading(btn, true);

    const payload = {
      reason: reason
    };
    if (type === 'new_balance') {
      payload.new_balance = amountVal;
    } else {
      payload.adjustment_amount = amountVal;
    }

    const res = await window.API.post(`/tenant/customers/${id}/adjust-balance`, payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-adjust-customer-balance', res.error);
      return;
    }

    closeModal('modal-adjust-customer-balance');
    showToast('Customer balance adjusted successfully');
    loadCustomers();
    loadDashboardMetrics();
    loadReceivablesDues();
  }

  // Bank: Create
  async function submitAddBank(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-bank');
    setButtonLoading(btn, true);

    const payload = {
      bank_name: document.getElementById('bank-form-name').value.trim(),
      account_number: document.getElementById('bank-form-account').value.trim(),
      ifsc: document.getElementById('bank-form-ifsc').value.trim(),
      opening_balance: parseFloat(document.getElementById('bank-form-opening')?.value) || 0
    };

    const res = await window.API.post('/tenant/banks', payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-add-bank', res.error);
      return;
    }

    closeModal('modal-add-bank');
    showToast('Bank account registered successfully');
    loadBanks();
    loadDashboardMetrics();
    loadDependencies();
  }

  // Bank: Edit
  async function openEditBank(id) {
    const res = await window.API.get(`/tenant/banks/${id}`);
    if (!res.ok || !res.data) {
      showToast(res.error || 'Failed to fetch bank', 'error');
      return;
    }
    const b = res.data;
    document.getElementById('edit-bank-id').value = b.id;
    document.getElementById('edit-bank-name').value = b.bank_name || '';
    document.getElementById('edit-bank-account').value = b.account_number || '';
    document.getElementById('edit-bank-ifsc').value = b.ifsc || '';
    document.getElementById('edit-bank-status').value = b.status || 'active';
    openModal('modal-edit-bank');
  }

  async function submitEditBank(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-edit-bank');
    const id = document.getElementById('edit-bank-id').value;
    setButtonLoading(btn, true);

    const payload = {
      bank_name: document.getElementById('edit-bank-name').value.trim(),
      account_number: document.getElementById('edit-bank-account').value.trim(),
      ifsc: document.getElementById('edit-bank-ifsc').value.trim(),
      status: document.getElementById('edit-bank-status').value
    };

    const res = await window.API.put(`/tenant/banks/${id}`, payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-edit-bank', res.error);
      return;
    }

    closeModal('modal-edit-bank');
    showToast('Bank account updated successfully');
    loadBanks();
  }

  // Line Sale: Calculations & Creation
  function calcLineSaleDue() {
    const total = parseFloat(document.getElementById('ls-form-amount')?.value) || 0;
    const cash = parseFloat(document.getElementById('ls-form-cash')?.value) || 0;
    const bank = parseFloat(document.getElementById('ls-form-bank-amount')?.value) || 0;
    const collected = cash + bank;
    const due = Math.max(0, total - collected);

    const elCollected = document.getElementById('ls-calc-collected');
    const elDue = document.getElementById('ls-calc-due');
    const elWarning = document.getElementById('ls-calc-warning');
    const btn = document.getElementById('btn-submit-line-sale');

    if (elCollected) elCollected.textContent = formatCurrency(collected);
    if (elDue) elDue.textContent = formatCurrency(due);

    if (total > 0 && collected > total) {
      if (elWarning) elWarning.style.display = 'block';
      if (elCollected) elCollected.style.color = 'var(--color-error)';
      if (btn) btn.disabled = true;
    } else {
      if (elWarning) elWarning.style.display = 'none';
      if (elCollected) elCollected.style.color = 'var(--color-success)';
      if (btn) btn.disabled = false;
    }
  }

  async function submitAddLineSale(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-line-sale');

    const customerID = document.getElementById('ls-form-customer').value;
    const amount = parseFloat(document.getElementById('ls-form-amount').value) || 0;
    const cashIn = parseFloat(document.getElementById('ls-form-cash').value) || 0;
    const bankID = document.getElementById('ls-form-bank').value || null;
    const bankAmount = parseFloat(document.getElementById('ls-form-bank-amount').value) || 0;
    const collected = cashIn + bankAmount;

    if (bankAmount > 0 && (!bankID || bankID === '')) {
      showModalError('modal-add-line-sale', 'Please select a specific bank account for the bank payment amount.');
      return;
    }

    if (amount > 0 && collected > amount) {
      showModalError('modal-add-line-sale', 'Collected amount cannot exceed the total invoice amount.');
      return;
    }

    setButtonLoading(btn, true);

    const payments = [];
    if (cashIn > 0) {
      payments.push({
        payment_method: 'cash',
        amount: cashIn,
        note: 'Cash collection'
      });
    }
    if (bankAmount > 0 && bankID) {
      payments.push({
        payment_method: 'bank',
        bank_id: bankID,
        amount: bankAmount,
        note: 'Bank / UPI collection'
      });
    }

    const payload = {
      customer_id: customerID,
      route: document.getElementById('ls-form-route').value.trim(),
      salesman: document.getElementById('ls-form-salesman').value.trim(),
      total_amount: amount,
      total_cash_in: cashIn,
      bank_amount: bankAmount,
      bank_id: bankID,
      note: document.getElementById('ls-form-note').value.trim(),
      payments: payments
    };

    const res = await window.API.post('/tenant/line-sales', payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-add-line-sale', res.error);
      return;
    }

    closeModal('modal-add-line-sale');
    showToast('Line sale recorded successfully');
    loadLineSales();
    loadDashboardMetrics();
    loadCashBank();
    loadCustomers();
  }

  // Line Sale: Edit
  async function openEditLineSale(id) {
    const res = await window.API.get(`/tenant/line-sales/${id}`);
    if (!res.ok || !res.data) {
      showToast(res.error || 'Failed to fetch line sale', 'error');
      return;
    }
    const s = res.data;
    document.getElementById('edit-ls-id').value = s.id;
    document.getElementById('edit-ls-route').value = s.route || '';
    document.getElementById('edit-ls-salesman').value = s.salesman || '';
    document.getElementById('edit-ls-amount').value = s.total_amount || '0';
    document.getElementById('edit-ls-cash').value = s.total_cash_in || '0';
    document.getElementById('edit-ls-note').value = s.note || '';
    openModal('modal-edit-line-sale');
  }

  async function submitEditLineSale(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-edit-ls');
    const id = document.getElementById('edit-ls-id').value;
    setButtonLoading(btn, true);

    const amount = parseFloat(document.getElementById('edit-ls-amount').value) || 0;
    const cash = parseFloat(document.getElementById('edit-ls-cash').value) || 0;

    const payload = {
      route: document.getElementById('edit-ls-route').value.trim(),
      salesman: document.getElementById('edit-ls-salesman').value.trim(),
      total_amount: amount,
      total_cash_in: cash,
      note: document.getElementById('edit-ls-note').value.trim()
    };

    const res = await window.API.put(`/tenant/line-sales/${id}`, payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-edit-line-sale', res.error);
      return;
    }

    closeModal('modal-edit-line-sale');
    showToast('Line sale updated successfully');
    loadLineSales();
    loadDashboardMetrics();
  }

  // Counter Sale: Mode Toggle, Calculation & Submission
  function handleCounterSaleMethodChange() {
    const method = document.getElementById('cs-form-method')?.value || 'cash';
    const total = parseFloat(document.getElementById('cs-form-total')?.value) || 0;
    const bankGroup = document.getElementById('cs-bank-select-group');
    const cashInput = document.getElementById('cs-form-cash');
    const bankInput = document.getElementById('cs-form-account');

    if (method === 'cash') {
      if (bankGroup) bankGroup.style.display = 'none';
      if (cashInput) cashInput.value = total.toFixed(2);
      if (bankInput) bankInput.value = '0.00';
    } else if (method === 'bank') {
      if (bankGroup) bankGroup.style.display = 'block';
      if (cashInput) cashInput.value = '0.00';
      if (bankInput) bankInput.value = total.toFixed(2);
    } else if (method === 'split') {
      if (bankGroup) bankGroup.style.display = 'block';
    }

    calcCounterSaleDue();
  }

  function calcCounterSaleDue() {
    const total = parseFloat(document.getElementById('cs-form-total')?.value) || 0;
    const method = document.getElementById('cs-form-method')?.value || 'cash';
    let cash = parseFloat(document.getElementById('cs-form-cash')?.value) || 0;
    let bank = parseFloat(document.getElementById('cs-form-account')?.value) || 0;

    if (method === 'cash') {
      cash = total;
      bank = 0;
    } else if (method === 'bank') {
      cash = 0;
      bank = total;
    }

    const collected = cash + bank;
    const due = Math.max(0, total - collected);

    const elCollected = document.getElementById('cs-calc-collected');
    const elDue = document.getElementById('cs-calc-due');
    const elWarning = document.getElementById('cs-calc-warning');
    const btn = document.getElementById('btn-submit-counter-sale');

    if (elCollected) elCollected.textContent = formatCurrency(collected);
    if (elDue) elDue.textContent = formatCurrency(due);

    if (total > 0 && collected > total) {
      if (elWarning) elWarning.style.display = 'block';
      if (elCollected) elCollected.style.color = 'var(--color-error)';
      if (btn) btn.disabled = true;
    } else {
      if (elWarning) elWarning.style.display = 'none';
      if (elCollected) elCollected.style.color = 'var(--color-success)';
      if (btn) btn.disabled = false;
    }
  }

  async function submitAddCounterSale(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-counter-sale');

    const price = parseFloat(document.getElementById('cs-form-price').value) || 0;
    const total = parseFloat(document.getElementById('cs-form-total').value) || 0;
    const method = document.getElementById('cs-form-method').value;
    let cash = parseFloat(document.getElementById('cs-form-cash').value) || 0;
    let bank = parseFloat(document.getElementById('cs-form-account').value) || 0;
    const bankID = document.getElementById('cs-form-bank')?.value || null;

    if (method === 'cash') {
      cash = total;
      bank = 0;
    } else if (method === 'bank') {
      cash = 0;
      bank = total;
    }

    const collected = cash + bank;

    if (bank > 0 && (!bankID || bankID === '')) {
      showModalError('modal-add-counter-sale', 'Please select a specific bank account for the bank payment.');
      return;
    }

    if (total > 0 && collected > total) {
      showModalError('modal-add-counter-sale', 'Collected amount cannot exceed the total sale amount.');
      return;
    }

    setButtonLoading(btn, true);

    const payload = {
      item: document.getElementById('cs-form-item').value.trim(),
      price: price > 0 ? price : total,
      total_amount: total,
      payment_method: method,
      cash: cash,
      bank_amount: bank,
      bank_id: bankID,
      account: bank
    };

    const res = await window.API.post('/tenant/counter-sales', payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-add-counter-sale', res.error);
      return;
    }

    closeModal('modal-add-counter-sale');
    showToast('Counter sale recorded successfully');
    loadCounterSales();
    loadDashboardMetrics();
    loadCashBank();
  }

  // Counter Sale: Edit
  async function openEditCounterSale(id) {
    const res = await window.API.get(`/tenant/counter-sales/${id}`);
    if (!res.ok || !res.data) {
      showToast(res.error || 'Failed to fetch counter sale', 'error');
      return;
    }
    const s = res.data;
    document.getElementById('edit-cs-id').value = s.id;
    document.getElementById('edit-cs-item').value = s.item || '';
    document.getElementById('edit-cs-price').value = s.price || '0';
    document.getElementById('edit-cs-total').value = s.total_amount || '0';
    document.getElementById('edit-cs-cash').value = s.cash || '0';
    document.getElementById('edit-cs-account').value = s.account || '0';
    openModal('modal-edit-counter-sale');
  }

  async function submitEditCounterSale(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-edit-cs');
    const id = document.getElementById('edit-cs-id').value;
    setButtonLoading(btn, true);

    const price = parseFloat(document.getElementById('edit-cs-price').value) || 0;
    const total = parseFloat(document.getElementById('edit-cs-total').value) || 0;
    const cash = parseFloat(document.getElementById('edit-cs-cash').value) || 0;
    const account = parseFloat(document.getElementById('edit-cs-account').value) || 0;

    const payload = {
      item: document.getElementById('edit-cs-item').value.trim(),
      price: price,
      total_amount: total,
      cash: cash,
      account: account
    };

    const res = await window.API.put(`/tenant/counter-sales/${id}`, payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-edit-counter-sale', res.error);
      return;
    }

    closeModal('modal-edit-counter-sale');
    showToast('Counter sale updated successfully');
    loadCounterSales();
    loadDashboardMetrics();
  }

  // Purchase: Create
  async function submitAddPurchase(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-purchase');
    setButtonLoading(btn, true);

    const total = parseFloat(document.getElementById('purch-form-total').value) || 0;
    const paid = parseFloat(document.getElementById('purch-form-paid').value) || 0;
    const method = document.getElementById('purch-form-method').value;
    const bankID = document.getElementById('purch-form-bank').value || null;

    const payments = [];
    if (paid > 0) {
      payments.push({
        payment_method: method,
        bank_id: bankID,
        amount: paid,
        note: 'Initial procurement payment'
      });
    }

    const payload = {
      item: document.getElementById('purch-form-item').value.trim(),
      quantity: parseInt(document.getElementById('purch-form-quantity').value) || 1,
      total_amount: total,
      total_paid: paid,
      payments: payments
    };

    const res = await window.API.post('/tenant/purchases', payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-add-purchase', res.error);
      return;
    }

    closeModal('modal-add-purchase');
    showToast('Purchase recorded successfully');
    loadPurchases();
    loadDashboardMetrics();
  }

  // Purchase: Edit
  async function openEditPurchase(id) {
    const res = await window.API.get(`/tenant/purchases/${id}`);
    if (!res.ok || !res.data) {
      showToast(res.error || 'Failed to fetch purchase', 'error');
      return;
    }
    const p = res.data;
    document.getElementById('edit-purch-id').value = p.id;
    document.getElementById('edit-purch-item').value = p.item || '';
    document.getElementById('edit-purch-quantity').value = p.quantity || 1;
    document.getElementById('edit-purch-total').value = p.total_amount || '0';
    document.getElementById('edit-purch-paid').value = p.total_paid || '0';
    openModal('modal-edit-purchase');
  }

  async function submitEditPurchase(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-edit-purch');
    const id = document.getElementById('edit-purch-id').value;
    setButtonLoading(btn, true);

    const qty = parseInt(document.getElementById('edit-purch-quantity').value) || 1;
    const total = parseFloat(document.getElementById('edit-purch-total').value) || 0;
    const paid = parseFloat(document.getElementById('edit-purch-paid').value) || 0;

    const payload = {
      item: document.getElementById('edit-purch-item').value.trim(),
      quantity: qty,
      total_amount: total,
      total_paid: paid
    };

    const res = await window.API.put(`/tenant/purchases/${id}`, payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-edit-purchase', res.error);
      return;
    }

    closeModal('modal-edit-purchase');
    showToast('Purchase updated successfully');
    loadPurchases();
    loadDashboardMetrics();
  }

  // Expense: Create
  async function submitAddExpense(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-expense');
    setButtonLoading(btn, true);

    const amount = parseFloat(document.getElementById('exp-form-amount').value) || 0;
    const method = document.getElementById('exp-form-method').value;
    const bankID = document.getElementById('exp-form-bank').value || null;

    const payload = {
      item: document.getElementById('exp-form-item').value.trim(),
      total_amount: amount,
      payment_method: method,
      bank_id: bankID
    };

    const res = await window.API.post('/tenant/expenses', payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-add-expense', res.error);
      return;
    }

    closeModal('modal-add-expense');
    showToast('Expense recorded successfully');
    loadExpenses();
    loadDashboardMetrics();
  }

  // Expense: Edit
  async function openEditExpense(id) {
    const res = await window.API.get(`/tenant/expenses/${id}`);
    if (!res.ok || !res.data) {
      showToast(res.error || 'Failed to fetch expense', 'error');
      return;
    }
    const e = res.data;
    document.getElementById('edit-exp-id').value = e.id;
    document.getElementById('edit-exp-item').value = e.item || '';
    document.getElementById('edit-exp-amount').value = e.total_amount || '0';
    openModal('modal-edit-expense');
  }

  async function submitEditExpense(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-edit-exp');
    const id = document.getElementById('edit-exp-id').value;
    setButtonLoading(btn, true);

    const amount = parseFloat(document.getElementById('edit-exp-amount').value) || 0;

    const payload = {
      item: document.getElementById('edit-exp-item').value.trim(),
      total_amount: amount
    };

    const res = await window.API.put(`/tenant/expenses/${id}`, payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-edit-expense', res.error);
      return;
    }

    closeModal('modal-edit-expense');
    showToast('Expense updated successfully');
    loadExpenses();
    loadDashboardMetrics();
  }

  // Attendance: Mark
  async function submitMarkAttendance(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-attendance');
    setButtonLoading(btn, true);

    const payload = {
      employee_id: document.getElementById('att-form-emp').value,
      date: document.getElementById('att-form-date').value || getTodayString(),
      status: document.getElementById('att-form-status').value,
      ot: parseFloat(document.getElementById('att-form-ot').value) || 0,
      advance: parseFloat(document.getElementById('att-form-advance').value) || 0
    };

    const res = await window.API.post('/tenant/attendance', payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-mark-attendance', res.error);
      return;
    }

    closeModal('modal-mark-attendance');
    showToast('Attendance recorded successfully');
    loadAttendance();
    loadDashboardMetrics();
  }

  // Attendance: Edit
  async function openEditAttendance(id) {
    const res = await window.API.get(`/tenant/attendance/${id}`);
    if (!res.ok || !res.data) {
      showToast(res.error || 'Failed to fetch attendance', 'error');
      return;
    }
    const a = res.data;
    document.getElementById('edit-att-id').value = a.id;
    document.getElementById('edit-att-empname').textContent = a.employee_name || 'Staff Member';
    document.getElementById('edit-att-status').value = a.status || 'present';
    document.getElementById('edit-att-ot').value = a.ot || '0';
    document.getElementById('edit-att-advance').value = a.advance || '0';
    openModal('modal-edit-attendance');
  }

  async function submitEditAttendance(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-edit-att');
    const id = document.getElementById('edit-att-id').value;
    setButtonLoading(btn, true);

    const ot = parseFloat(document.getElementById('edit-att-ot').value) || 0;
    const advance = parseFloat(document.getElementById('edit-att-advance').value) || 0;

    const payload = {
      status: document.getElementById('edit-att-status').value,
      ot: ot,
      advance: advance
    };

    const res = await window.API.put(`/tenant/attendance/${id}`, payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-edit-attendance', res.error);
      return;
    }

    closeModal('modal-edit-attendance');
    showToast('Attendance updated successfully');
    loadAttendance();
    loadDashboardMetrics();
  }

  // Overtime: Record
  async function submitRecordOvertime(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-ot');
    setButtonLoading(btn, true);

    const payload = {
      employee_id: document.getElementById('ot-form-emp').value,
      date: document.getElementById('ot-form-date').value || getTodayString(),
      ot: parseFloat(document.getElementById('ot-form-hours').value) || 0
    };

    const res = await window.API.post('/tenant/overtime', payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-record-ot', res.error);
      return;
    }

    closeModal('modal-record-ot');
    showToast('Overtime hours recorded successfully');
    loadAttendance();
  }

  // Advance: Record
  async function submitRecordAdvance(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-advance');
    setButtonLoading(btn, true);

    const payload = {
      employee_id: document.getElementById('adv-form-emp').value,
      date: document.getElementById('adv-form-date').value || getTodayString(),
      amount: parseFloat(document.getElementById('adv-form-amount').value) || 0,
      payment_method: document.getElementById('adv-form-method').value
    };

    const res = await window.API.post('/tenant/advances', payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-record-advance', res.error);
      return;
    }

    closeModal('modal-record-advance');
    showToast('Staff advance payment recorded');
    loadAttendance();
    loadDashboardMetrics();
  }

  // Salary: Pay
  function openPaySalary(empId, empName, balance) {
    document.getElementById('pay-salary-empid').value = empId;
    document.getElementById('pay-salary-empname').textContent = empName;
    document.getElementById('pay-salary-pending').textContent = formatCurrency(balance);
    document.getElementById('pay-salary-amount').value = Math.max(0, parseFloat(balance) || 0);
    openModal('modal-pay-salary');
  }

  async function submitPaySalary(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-pay-salary');
    const empId = document.getElementById('pay-salary-empid').value;
    setButtonLoading(btn, true);

    const payload = {
      amount: parseFloat(document.getElementById('pay-salary-amount').value) || 0,
      payment_method: document.getElementById('pay-salary-method').value,
      bank_id: document.getElementById('pay-salary-bank').value || null,
      note: document.getElementById('pay-salary-note').value.trim()
    };

    const res = await window.API.post(`/tenant/salaries/${empId}/pay`, payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-pay-salary', res.error);
      return;
    }

    closeModal('modal-pay-salary');
    showToast('Salary disbursed successfully');
    loadSalaries();
    loadDashboardMetrics();
  }

  // Salary: Adjust Balance
  function openAdjustSalary(empId, empName, currentBal) {
    document.getElementById('adj-salary-empid').value = empId;
    document.getElementById('adj-salary-empname').textContent = empName;
    document.getElementById('adj-salary-balance').value = currentBal || '0';
    openModal('modal-adjust-salary');
  }

  async function submitAdjustSalary(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-adjust-salary');
    const empId = document.getElementById('adj-salary-empid').value;
    setButtonLoading(btn, true);

    const payload = {
      balance: parseFloat(document.getElementById('adj-salary-balance').value) || 0
    };

    const res = await window.API.put(`/tenant/salaries/${empId}`, payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-adjust-salary', res.error);
      return;
    }

    closeModal('modal-adjust-salary');
    showToast('Salary balance adjusted successfully');
    loadSalaries();
    loadDashboardMetrics();
  }

  // ==========================================
  // View Details Modals (Inspection)
  // ==========================================

  async function viewLineSale(id) {
    const res = await window.API.get(`/tenant/line-sales/${id}`);
    if (!res.ok || !res.data) {
      showToast(res.error || 'Failed to load details', 'error');
      return;
    }
    const s = res.data;
    document.getElementById('view-details-title').textContent = `Line Sale Details - ${s.customer_name || 'Customer'}`;
    const paymentsHtml = Array.isArray(s.payments) && s.payments.length > 0
      ? s.payments.map(p => `<li>${formatCurrency(p.amount)} via ${p.payment_method.toUpperCase()} ${p.note ? '— ' + escapeHTML(p.note) : ''}</li>`).join('')
      : '<li>No separate payment records attached.</li>';

    document.getElementById('view-details-content').innerHTML = `
      <div style="display:flex;flex-direction:column;gap:0.75rem;font-size:0.875rem;">
        <div><strong>Customer:</strong> ${escapeHTML(s.customer_name)}</div>
        <div><strong>Route / Driver:</strong> ${escapeHTML(s.route || '—')} / ${escapeHTML(s.salesman || '—')}</div>
        <div><strong>Date:</strong> ${formatDateTime(s.created_at)}</div>
        <div style="font-size:1.125rem;font-weight:700;color:var(--color-text-main);margin-top:0.5rem;">
          Total Amount: ${formatCurrency(s.total_amount)}
        </div>
        <div><strong>Cash Collected:</strong> <span style="color:var(--color-success);font-weight:600;">${formatCurrency(s.total_cash_in)}</span></div>
        <div><strong>Balance Due:</strong> <span style="color:${parseFloat(s.balance) > 0 ? 'var(--color-warning)' : 'inherit'};font-weight:600;">${formatCurrency(s.balance)}</span></div>
        ${s.note ? `<div><strong>Remarks:</strong> ${escapeHTML(s.note)}</div>` : ''}
        <div style="margin-top:0.5rem;border-top:1px solid var(--color-border);padding-top:0.5rem;">
          <strong>Payments Breakdown:</strong>
          <ul style="margin:0.375rem 0 0 1.25rem;padding:0;color:var(--color-text-muted);">
            ${paymentsHtml}
          </ul>
        </div>
      </div>
    `;
    openModal('modal-view-details');
  }

  async function viewCounterSale(id) {
    const res = await window.API.get(`/tenant/counter-sales/${id}`);
    if (!res.ok || !res.data) {
      showToast(res.error || 'Failed to load details', 'error');
      return;
    }
    const s = res.data;
    document.getElementById('view-details-title').textContent = `Counter Sale - ${s.item || 'Item'}`;
    document.getElementById('view-details-content').innerHTML = `
      <div style="display:flex;flex-direction:column;gap:0.75rem;font-size:0.875rem;">
        <div><strong>Item Description:</strong> ${escapeHTML(s.item)}</div>
        <div><strong>Date:</strong> ${formatDateTime(s.created_at)}</div>
        <div><strong>Payment Method:</strong> <span class="status-badge cleared">${escapeHTML(s.payment_method.toUpperCase())}</span></div>
        <div style="font-size:1.125rem;font-weight:700;color:var(--color-text-main);margin-top:0.5rem;">
          Total Bill Amount: ${formatCurrency(s.total_amount)}
        </div>
        <div><strong>Cash Portion:</strong> ${formatCurrency(s.cash)}</div>
        <div><strong>Online / UPI Portion:</strong> ${formatCurrency(s.account)}</div>
      </div>
    `;
    openModal('modal-view-details');
  }

  async function viewEmployee(id) {
    const res = await window.API.get(`/tenant/employees/${id}`);
    if (!res.ok || !res.data) {
      showToast(res.error || 'Failed to load details', 'error');
      return;
    }
    const e = res.data;
    document.getElementById('view-details-title').textContent = `Employee Profile - ${e.name}`;
    document.getElementById('view-details-content').innerHTML = `
      <div style="display:flex;flex-direction:column;gap:0.75rem;font-size:0.875rem;">
        <div><strong>Name:</strong> ${escapeHTML(e.name)}</div>
        <div><strong>Phone:</strong> ${escapeHTML(e.phone || '—')}</div>
        <div><strong>Base Salary:</strong> ${formatCurrency(e.salary)}</div>
        <div><strong>Overtime Policy:</strong> ${formatCurrency(e.ot_rate)} per hour</div>
        <div><strong>Status:</strong> <span class="status-badge ${e.status === 'active' ? 'paid' : 'pending'}">${escapeHTML(e.status)}</span></div>
        <div><strong>Joined:</strong> ${formatDate(e.created_at)}</div>
      </div>
    `;
    openModal('modal-view-details');
  }

  async function viewCustomer(id) {
    const res = await window.API.get(`/tenant/customers/${id}`);
    if (!res.ok || !res.data) {
      showToast(res.error || 'Failed to load details', 'error');
      return;
    }
    const c = res.data;
    document.getElementById('view-details-title').textContent = `Customer Ledger - ${c.customer_name}`;
    document.getElementById('view-details-content').innerHTML = `
      <div style="display:flex;flex-direction:column;gap:0.75rem;font-size:0.875rem;">
        <div><strong>Customer / Business:</strong> ${escapeHTML(c.customer_name)}</div>
        <div><strong>Contact Phone:</strong> ${escapeHTML(c.phone || '—')}</div>
        <div><strong>Opening Credit Balance:</strong> ${formatCurrency(c.opening_balance)}</div>
        <div style="font-size:1.125rem;font-weight:700;margin-top:0.5rem;">
          Current Outstanding Dues: <span style="color:var(--color-warning);">${formatCurrency(c.current_balance)}</span>
        </div>
        <div><strong>Account Status:</strong> <span class="status-badge ${c.status === 'active' ? 'paid' : 'pending'}">${escapeHTML(c.status)}</span></div>
      </div>
    `;
    openModal('modal-view-details');
  }

  // ==========================================
  // Deletions
  // ==========================================

  async function deleteUser(id) {
    if (!confirm('Are you sure you want to delete this tenant user?')) return;
    const res = await window.API.delete(`/tenant/users/${id}`);
    if (!res.ok) {
      showToast(res.error, 'error');
      return;
    }
    showToast('User removed');
    loadTenantUsers();
  }

  async function deleteEmployee(id) {
    if (!confirm('Are you sure you want to delete this employee?')) return;
    const res = await window.API.delete(`/tenant/employees/${id}`);
    if (!res.ok) {
      showToast(res.error, 'error');
      return;
    }
    showToast('Employee removed');
    loadEmployees();
    loadDependencies();
  }

  async function deleteCustomer(id) {
    if (!confirm('Are you sure you want to delete this customer?')) return;
    const res = await window.API.delete(`/tenant/customers/${id}`);
    if (!res.ok) {
      showToast(res.error, 'error');
      return;
    }
    showToast('Customer removed');
    loadCustomers();
    loadDependencies();
  }

  async function deleteBank(id) {
    if (!confirm('Are you sure you want to delete this bank account?')) return;
    const res = await window.API.delete(`/tenant/banks/${id}`);
    if (!res.ok) {
      showToast(res.error, 'error');
      return;
    }
    showToast('Bank account removed');
    loadBanks();
    loadDependencies();
  }

  async function deleteLineSale(id) {
    if (!confirm('Are you sure you want to delete this line sale?')) return;
    const res = await window.API.delete(`/tenant/line-sales/${id}`);
    if (!res.ok) {
      showToast(res.error, 'error');
      return;
    }
    showToast('Line sale removed');
    loadLineSales();
    loadDashboardMetrics();
  }

  async function deleteCounterSale(id) {
    if (!confirm('Are you sure you want to delete this counter sale?')) return;
    const res = await window.API.delete(`/tenant/counter-sales/${id}`);
    if (!res.ok) {
      showToast(res.error, 'error');
      return;
    }
    showToast('Counter sale removed');
    loadCounterSales();
    loadDashboardMetrics();
  }

  async function deletePurchase(id) {
    if (!confirm('Are you sure you want to delete this purchase entry?')) return;
    const res = await window.API.delete(`/tenant/purchases/${id}`);
    if (!res.ok) {
      showToast(res.error, 'error');
      return;
    }
    showToast('Purchase record removed');
    loadPurchases();
    loadDashboardMetrics();
  }

  async function deleteExpense(id) {
    if (!confirm('Are you sure you want to delete this expense entry?')) return;
    const res = await window.API.delete(`/tenant/expenses/${id}`);
    if (!res.ok) {
      showToast(res.error, 'error');
      return;
    }
    showToast('Expense record removed');
    loadExpenses();
    loadDashboardMetrics();
  }

  // ==========================================
  // Navigation & Tab Switching
  // ==========================================

  function switchTab(tabName) {
    if (!tabName) tabName = 'overview';

    // Hide all tab panes
    document.querySelectorAll('.tab-pane').forEach(el => {
      el.style.display = 'none';
    });

    // Remove active from all sidebar links
    document.querySelectorAll('.sidebar-link').forEach(btn => {
      btn.classList.remove('active');
    });

    // Show target pane
    const target = document.getElementById(`tab-${tabName}`);
    if (target) target.style.display = 'block';

    // Highlight target link
    const btn = document.querySelector(`.sidebar-link[data-tab="${tabName}"]`);
    if (btn) btn.classList.add('active');

    // Close mobile drawer if open
    closeMobileSidebar();

    // Trigger tab-specific loaders
    switch (tabName) {
      case 'overview':
        loadDashboardMetrics();
        break;
      case 'users':
        loadTenantUsers();
        break;
      case 'employees':
        loadEmployees();
        break;
      case 'customers':
        loadCustomers();
        break;
      case 'banks':
        loadBanks();
        break;
      case 'line-sales':
        loadLineSales();
        break;
      case 'counter-sales':
        loadCounterSales();
        break;
      case 'purchases':
        loadPurchases();
        break;
      case 'expenses':
        loadExpenses();
        break;
      case 'attendance':
        loadAttendance();
        break;
      case 'salaries':
        loadSalaries();
        break;
      case 'cash-bank':
        loadCashBank();
        break;
      case 'receivables-dues':
        loadReceivablesDues();
        break;
      case 'profile':
        loadProfile();
        break;
    }
  }

  function openMobileSidebar() {
    const sidebar = document.getElementById('app-sidebar');
    const backdrop = document.getElementById('sidebar-backdrop');
    if (sidebar) sidebar.classList.add('open');
    if (backdrop) backdrop.classList.add('active');
    document.body.style.overflow = 'hidden';
  }

  function closeMobileSidebar() {
    const sidebar = document.getElementById('app-sidebar');
    const backdrop = document.getElementById('sidebar-backdrop');
    if (sidebar) sidebar.classList.remove('open');
    if (backdrop) backdrop.classList.remove('active');
    document.body.style.overflow = 'auto';
  }

  function toggleMobileSidebar() {
    const sidebar = document.getElementById('app-sidebar');
    if (sidebar && sidebar.classList.contains('open')) {
      closeMobileSidebar();
    } else {
      openMobileSidebar();
    }
  }

  // ==========================================
  // Initialization
  // ==========================================

  async function init() {
    currentUser = window.Auth ? window.Auth.getUser() : null;
    currentRole = (currentUser && currentUser.role) ? currentUser.role : 'user';

    // Enforce Frontend RBAC Visibility
    const isAdmin = currentRole === 'admin';
    document.querySelectorAll('.admin-only').forEach(el => {
      el.style.display = isAdmin ? '' : 'none';
    });
    document.querySelectorAll('.user-only').forEach(el => {
      el.style.display = !isAdmin ? '' : 'none';
    });

    const roleBadge = document.getElementById('user-role-badge');
    if (roleBadge) {
      roleBadge.textContent = isAdmin ? 'Tenant Admin' : 'Tenant Operator';
      roleBadge.className = 'status-badge ' + (isAdmin ? 'paid' : 'cleared');
    }

    // Wire Mobile Sidebar Toggle & Backdrop
    const mobileToggle = document.getElementById('mobile-sidebar-toggle');
    if (mobileToggle) {
      mobileToggle.addEventListener('click', toggleMobileSidebar);
    }
    const backdrop = document.getElementById('sidebar-backdrop');
    if (backdrop) {
      backdrop.addEventListener('click', closeMobileSidebar);
    }

    // Wire Sidebar Tab Navigation
    document.querySelectorAll('.sidebar-link[data-tab]').forEach(link => {
      link.addEventListener('click', (e) => {
        e.preventDefault();
        const tab = link.dataset.tab;
        switchTab(tab);
      });
    });

    // Default dates on inputs to today
    const today = getTodayString();
    document.querySelectorAll('input[type="date"]').forEach(inp => {
      if (!inp.value) inp.value = today;
    });

    // Global Search listener (live filter on active view)
    const globalSearch = document.getElementById('global-search-input');
    if (globalSearch) {
      globalSearch.addEventListener('input', (e) => {
        const query = e.target.value.toLowerCase().trim();
        const activePane = document.querySelector('.tab-pane[style*="display: block"]');
        if (!activePane) return;
        const rows = activePane.querySelectorAll('tbody tr');
        rows.forEach(row => {
          const text = row.textContent.toLowerCase();
          row.style.display = (!query || text.includes(query)) ? '' : 'none';
        });
      });
    }

    // Wire Real-time Filter Listeners
    const dateLineSale = document.getElementById('filter-line-sales-date');
    if (dateLineSale) dateLineSale.addEventListener('change', () => loadLineSales());
    const searchLineSale = document.getElementById('search-line-sales');
    if (searchLineSale) searchLineSale.addEventListener('input', () => loadLineSales());

    const dateCounterSale = document.getElementById('filter-counter-sales-date');
    if (dateCounterSale) dateCounterSale.addEventListener('change', () => loadCounterSales());

    const datePurch = document.getElementById('filter-purchases-date');
    if (datePurch) datePurch.addEventListener('change', () => loadPurchases());

    const dateExp = document.getElementById('filter-expenses-date');
    if (dateExp) dateExp.addEventListener('change', () => loadExpenses());

    const dateAtt = document.getElementById('filter-attendance-date');
    if (dateAtt) dateAtt.addEventListener('change', () => loadAttendance());

    const searchUsers = document.getElementById('search-users');
    if (searchUsers) searchUsers.addEventListener('input', () => loadTenantUsers());

    const searchEmp = document.getElementById('search-employees');
    if (searchEmp) searchEmp.addEventListener('input', () => loadEmployees());

    const searchCust = document.getElementById('search-customers');
    if (searchCust) searchCust.addEventListener('input', () => loadCustomers());

    // Wire Modal Backdrop Click
    document.querySelectorAll('.modal-overlay').forEach(modal => {
      modal.addEventListener('click', (e) => {
        if (e.target === modal) {
          closeModal(modal.id);
        }
      });
    });

    // Wire Form Submissions (Create)
    const formUser = document.getElementById('form-add-user');
    if (formUser) formUser.addEventListener('submit', submitAddUser);

    const formEmp = document.getElementById('form-add-employee');
    if (formEmp) formEmp.addEventListener('submit', submitAddEmployee);

    const formCust = document.getElementById('form-add-customer');
    if (formCust) formCust.addEventListener('submit', submitAddCustomer);

    const formAdjCust = document.getElementById('form-adjust-customer-balance');
    if (formAdjCust) formAdjCust.addEventListener('submit', submitAdjustCustomerBalance);

    const formBank = document.getElementById('form-add-bank');
    if (formBank) formBank.addEventListener('submit', submitAddBank);

    const formLineSale = document.getElementById('form-add-line-sale');
    if (formLineSale) formLineSale.addEventListener('submit', submitAddLineSale);

    const formCounterSale = document.getElementById('form-add-counter-sale');
    if (formCounterSale) formCounterSale.addEventListener('submit', submitAddCounterSale);

    const formPurch = document.getElementById('form-add-purchase');
    if (formPurch) formPurch.addEventListener('submit', submitAddPurchase);

    const formExp = document.getElementById('form-add-expense');
    if (formExp) formExp.addEventListener('submit', submitAddExpense);

    const formAtt = document.getElementById('form-mark-attendance');
    if (formAtt) formAtt.addEventListener('submit', submitMarkAttendance);

    const formOT = document.getElementById('form-record-ot');
    if (formOT) formOT.addEventListener('submit', submitRecordOvertime);

    const formAdv = document.getElementById('form-record-advance');
    if (formAdv) formAdv.addEventListener('submit', submitRecordAdvance);

    const formPaySalary = document.getElementById('form-pay-salary');
    if (formPaySalary) formPaySalary.addEventListener('submit', submitPaySalary);

    const formAdjSalary = document.getElementById('form-adjust-salary');
    if (formAdjSalary) formAdjSalary.addEventListener('submit', submitAdjustSalary);

    // Wire Form Submissions (Edit)
    const formEditUser = document.getElementById('form-edit-user');
    if (formEditUser) formEditUser.addEventListener('submit', submitEditUser);

    const formEditEmp = document.getElementById('form-edit-employee');
    if (formEditEmp) formEditEmp.addEventListener('submit', submitEditEmployee);

    const formEditCust = document.getElementById('form-edit-customer');
    if (formEditCust) formEditCust.addEventListener('submit', submitEditCustomer);

    const formEditBank = document.getElementById('form-edit-bank');
    if (formEditBank) formEditBank.addEventListener('submit', submitEditBank);

    const formEditLS = document.getElementById('form-edit-line-sale');
    if (formEditLS) formEditLS.addEventListener('submit', submitEditLineSale);

    const formEditCS = document.getElementById('form-edit-counter-sale');
    if (formEditCS) formEditCS.addEventListener('submit', submitEditCounterSale);

    const formEditPurch = document.getElementById('form-edit-purchase');
    if (formEditPurch) formEditPurch.addEventListener('submit', submitEditPurchase);

    const formEditExp = document.getElementById('form-edit-expense');
    if (formEditExp) formEditExp.addEventListener('submit', submitEditExpense);

    const formEditAtt = document.getElementById('form-edit-attendance');
    if (formEditAtt) formEditAtt.addEventListener('submit', submitEditAttendance);

    // Initial Data Preload & Overview Load
    await loadDependencies();
    switchTab('overview');
  }

  return {
    init,
    openModal,
    closeModal,
    switchTab,
    deleteUser,
    deleteEmployee,
    deleteCustomer,
    deleteBank,
    deleteLineSale,
    deleteCounterSale,
    deletePurchase,
    deleteExpense,
    openPaySalary,
    openAdjustSalary,
    openEditUser,
    openEditEmployee,
    openEditCustomer,
    openEditBank,
    openEditLineSale,
    openEditCounterSale,
    openEditPurchase,
    openEditExpense,
    openEditAttendance,
    viewLineSale,
    viewCounterSale,
    viewEmployee,
    viewCustomer,
    calcLineSaleDue,
    handleCounterSaleMethodChange,
    calcCounterSaleDue,
    openAdjustCustomerBalance,
    handleCustomerAdjustmentTypeChange,
    calcCustomerAdjustmentPreview,
    submitAdjustCustomerBalance,
    loadLineSales,
    loadCounterSales,
    loadPurchases,
    loadExpenses,
    loadAttendance,
    loadEmployees,
    loadCustomers,
    loadTenantUsers,
    loadBanks,
    loadSalaries,
    loadCashBank,
    loadReceivablesDues,
    loadDashboardMetrics
  };
})();

window.Operations = Operations;
