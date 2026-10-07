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
  let purchaseCustomerBalance = null;
  let lineSaleCustomerBalance = null;
  let lineSaleBankRowCount = 0;
  let counterSaleBankRowCount = 0;
  let purchaseBankRowCount = 0;
  let expenseBankRowCount = 0;
  let editExpenseBankRowCount = 0;
  let activeStatementCustomerID = null;
  let supplierPaymentTarget = { customerId: null, purchaseId: null, pending: 0, customerName: '', item: '' };

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
      if (modalId === 'modal-add-expense') {
        handleExpensePaymentMethodChange('cash');
      }
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

      // Specific modal cleanup
      if (modalId === 'modal-add-line-sale') {
        const balBox = document.getElementById('ls-customer-balance-box');
        if (balBox) balBox.style.display = 'none';
        const bankContainer = document.getElementById('ls-bank-payments-container');
        if (bankContainer) bankContainer.innerHTML = '';
        const warn = document.getElementById('ls-bank-duplicate-warning');
        if (warn) warn.style.display = 'none';
        lineSaleCustomerBalance = null;
        calcLineSaleDue();
      } else if (modalId === 'modal-add-counter-sale') {
        const bankContainer = document.getElementById('cs-bank-payments-container');
        if (bankContainer) bankContainer.innerHTML = '';
        const warn = document.getElementById('cs-bank-duplicate-warning');
        if (warn) warn.style.display = 'none';
        calcCounterSaleDue();
      } else if (modalId === 'modal-add-purchase') {
        const balBox = document.getElementById('purch-customer-balance-box');
        if (balBox) balBox.style.display = 'none';
        const bankContainer = document.getElementById('purch-bank-payments-container');
        if (bankContainer) bankContainer.innerHTML = '';
        const warn = document.getElementById('purch-bank-duplicate-warning');
        if (warn) warn.style.display = 'none';
        purchaseCustomerBalance = null;
        calcPurchaseBalance();
      } else if (modalId === 'modal-make-supplier-payment') {
        supplierPaymentTarget = { customerId: null, purchaseId: null, pending: 0, customerName: '', item: '' };
        const warn = document.getElementById('supp-pay-warning');
        if (warn) warn.style.display = 'none';
      } else if (modalId === 'modal-add-expense') {
        const bankContainer = document.getElementById('exp-bank-payments-container');
        if (bankContainer) bankContainer.innerHTML = '';
        const warn = document.getElementById('exp-bank-duplicate-warning');
        if (warn) warn.style.display = 'none';
        handleExpensePaymentMethodChange('cash');
      } else if (modalId === 'modal-edit-expense') {
        const bankContainer = document.getElementById('edit-exp-bank-payments-container');
        if (bankContainer) bankContainer.innerHTML = '';
        const warn = document.getElementById('edit-exp-bank-duplicate-warning');
        if (warn) warn.style.display = 'none';
        handleEditExpensePaymentMethodChange('cash');
      }
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
      const isPurchase = sel.id && sel.id.includes('purch');
      const defaultText = isPurchase ? 'None / Walk-in Vendor' : 'Select Customer...';
      sel.innerHTML = `<option value="">${defaultText}</option>` +
        cachedCustomers.map(c => `<option value="${c.id}">${escapeHTML(c.customer_name || 'Customer')}${c.phone ? ' (' + escapeHTML(c.phone) + ')' : ''}</option>`).join('');
      if (cur) sel.value = cur;
    });

    document.querySelectorAll('.select-bank').forEach(sel => {
      if (sel.classList.contains('ls-bank-select') || sel.classList.contains('cs-bank-select')) return;
      const cur = sel.value;
      const activeBanks = cachedBanks.filter(b => b.status === 'active' || !b.status);
      sel.innerHTML = '<option value="">None / Cash Only</option>' +
        activeBanks.map(b => `<option value="${b.id}">${escapeHTML(b.bank_name)} (${escapeHTML(b.account_number || 'Main')})</option>`).join('');
      if (cur) sel.value = cur;
    });
  }

  function filterCustomerDropdown(inputEl, selectId) {
    const sel = document.getElementById(selectId);
    if (!sel) return;
    const query = (inputEl.value || '').toLowerCase().trim();
    const curVal = sel.value;

    const matched = cachedCustomers.filter(c => {
      if (!query) return true;
      const name = (c.customer_name || '').toLowerCase();
      const phone = (c.phone || '').toLowerCase();
      return name.includes(query) || phone.includes(query);
    });

    const isPurchase = selectId.includes('purch');
    const defaultLabel = isPurchase ? 'None / Walk-in Vendor' : 'Select Customer...';

    sel.innerHTML = `<option value="">${defaultLabel}</option>` +
      matched.map(c => `<option value="${c.id}">${escapeHTML(c.customer_name || 'Customer')}${c.phone ? ' (' + escapeHTML(c.phone) + ')' : ''}</option>`).join('');

    if (curVal && matched.some(c => c.id === curVal)) {
      sel.value = curVal;
    } else {
      sel.value = '';
    }
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

    const elTodayPayable = document.getElementById('overview-outstanding-payable-val');
    if (elTodayPayable && (!m.today_overview || m.today_overview.outstanding_payable === undefined)) {
      elTodayPayable.textContent = formatCurrency(m.total_payable || 0);
    }

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

    // 6. Outstanding Payable
    const elPayableVal = document.getElementById('overview-outstanding-payable-val');
    const elPayableTrend = document.getElementById('overview-outstanding-payable-trend');
    const elPayableBadge = document.getElementById('overview-payable-badge');
    if (elPayableVal) {
      const payableAmount = overview.outstanding_payable !== undefined ? overview.outstanding_payable : 0;
      elPayableVal.textContent = formatCurrency(payableAmount);
      const payableNum = parseFloat(payableAmount) || 0;
      if (elPayableTrend) {
        elPayableTrend.textContent = payableNum > 0 ? 'Total pending payables across suppliers' : 'All supplier dues settled';
      }
      if (elPayableBadge) {
        if (payableNum > 0) {
          elPayableBadge.className = 'status-badge pending';
          elPayableBadge.textContent = 'Supplier Dues';
        } else {
          elPayableBadge.className = 'status-badge cleared';
          elPayableBadge.textContent = 'All Settled';
        }
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
              <button class="btn btn-sm btn-secondary" onclick="Operations.showLineSalePaymentHistory('${s.id}', '${escapeHTML(s.customer_name || 'Customer')}', ${s.total_amount || 0}, ${s.total_cash_in || 0}, ${s.balance || 0})" title="View Payment History">History</button>
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
              <button class="btn btn-sm btn-secondary" onclick="Operations.showCounterSalePaymentHistory('${s.id}', '${escapeHTML(s.item || 'Counter Sale')}', ${s.total_amount || 0})" title="View Payment History">History</button>
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
          <td style="color:${parseFloat(a.ot_amount) > 0 ? 'var(--color-success)' : 'inherit'};font-weight:600;">${parseFloat(a.ot_amount) > 0 ? formatCurrency(a.ot_amount) : (parseFloat(a.ot) > 0 ? a.ot + ' hrs' : '—')}</td>
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
          <td colspan="9">
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

      let statusBadge = '<span class="status-badge danger">UNPAID</span>';
      if (p.payment_status === 'PAID' || pending <= 0) {
        statusBadge = '<span class="status-badge paid">PAID</span>';
      } else if (p.payment_status === 'PARTIALLY_PAID' || (parseFloat(p.total_paid) > 0 && pending > 0)) {
        statusBadge = '<span class="status-badge pending">PARTIALLY PAID</span>';
      }

      return `
        <tr>
          <td><strong>${escapeHTML(p.item || 'Item')}</strong></td>
          <td>${p.customer_name ? `<span class="badge" style="background:rgba(59,130,246,0.1);color:#3b82f6;font-weight:600;">${escapeHTML(p.customer_name)}</span>` : '<span style="color:var(--color-text-muted);font-size:0.8125rem;">Direct / Vendor</span>'}</td>
          <td>${p.quantity || 1}</td>
          <td>${formatDate(p.created_at)}</td>
          <td style="font-weight:700;">${formatCurrency(p.total_amount)}</td>
          <td style="color:var(--color-success);font-weight:600;">${formatCurrency(p.total_paid)}</td>
          <td style="color:${pending > 0 ? 'var(--color-error)' : 'var(--color-text-muted)'};font-weight:600;">${formatCurrency(p.total_pending)}</td>
          <td>${statusBadge}</td>
          <td style="text-align:right;">
            <div style="display:inline-flex;gap:0.375rem;justify-content:flex-end;align-items:center;">
              <button class="btn btn-sm btn-secondary" onclick="Operations.showPurchasePaymentHistory('${p.id}', '${escapeHTML(p.item || 'Purchase')}', ${p.total_amount || 0})" title="View Payment History" style="padding: 0.2rem 0.6rem; font-size: 0.75rem;">History</button>
              ${pending > 0 ? `
                <button class="btn btn-sm btn-primary" onclick="Operations.openMakePaymentModal({purchaseId: '${p.id}', customerId: '${p.customer_id || ''}', customerName: '${escapeHTML(p.customer_name || 'Direct Vendor')}', pending: ${pending}, item: '${escapeHTML(p.item || '')}'})" title="Pay outstanding for this purchase" style="padding: 0.2rem 0.6rem; font-size: 0.75rem;">Pay</button>
              ` : ''}
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
          <td colspan="5">
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
      const isAdvance = e.category === 'employee_advance';
      let pmBadge = '<span class="status-badge paid" style="font-size:0.75rem;">Cash</span>';
      if (e.payment_method === 'bank') {
        const banksCount = e.payment_breakdown?.banks?.length || 1;
        const bankLabel = banksCount > 1 ? `Bank (${banksCount} accounts)` : (e.payment_breakdown?.banks?.[0]?.bank_name ? `Bank: ${escapeHTML(e.payment_breakdown.banks[0].bank_name)}` : 'Bank');
        pmBadge = `<span class="status-badge cleared" style="font-size:0.75rem;">${bankLabel}</span>`;
      } else if (e.payment_method === 'cash_bank') {
        const banksCount = e.payment_breakdown?.banks?.length || 1;
        const bankSuffix = banksCount > 1 ? ` (${banksCount} banks)` : '';
        pmBadge = `<span class="status-badge pending" style="font-size:0.75rem; background: rgba(147, 51, 234, 0.1); color: #7e22ce; border: 1px solid rgba(147, 51, 234, 0.2);">Cash + Bank${bankSuffix}</span>`;
      }

      return `
        <tr>
          <td>
            <strong>${escapeHTML(e.item || 'Expense')}</strong>
            ${isAdvance ? `<span class="badge" style="margin-left: 0.375rem; font-size: 0.7rem; background: rgba(239, 68, 68, 0.1); color: var(--color-error); font-weight: 700;">${escapeHTML(e.employee_name ? 'STAFF ADVANCE: ' + e.employee_name : 'STAFF ADVANCE')}</span>` : (e.category && e.category !== 'general' ? `<span class="badge" style="margin-left: 0.375rem; font-size: 0.7rem; background: rgba(99, 102, 241, 0.1); color: #6366f1;">${escapeHTML(e.category.replace('_', ' ').toUpperCase())}</span>` : '')}
          </td>
          <td>${formatDate(e.created_at)}</td>
          <td>${pmBadge}</td>
          <td style="font-weight:700;color:var(--color-error);">${formatCurrency(e.total_amount)}</td>
          <td style="text-align:right;">
            <div style="display:inline-flex;gap:0.375rem;justify-content:flex-end;">
              <button class="btn btn-sm btn-secondary" onclick="Operations.viewExpense('${e.id}')">View</button>
              ${isAdmin ? `
                <button class="btn btn-sm btn-secondary" onclick="Operations.openEditExpense('${e.id}')">Edit</button>
                <button class="btn btn-sm btn-secondary" onclick="Operations.deleteExpense('${e.id}')" style="color:var(--color-error);">Delete</button>
              ` : ''}
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

    tbody.innerHTML = res.data.map(c => {
      const recv = parseFloat(c.current_balance) || 0;
      const payable = parseFloat(c.outstanding_payable) || 0;

      return `
        <tr>
          <td><strong>${escapeHTML(c.customer_name)}</strong></td>
          <td>${escapeHTML(c.phone || '—')}</td>
          <td style="font-weight:700;color:${recv > 0 ? 'var(--color-warning)' : 'inherit'};">${formatCurrency(c.current_balance)}</td>
          <td style="font-weight:700;color:${payable > 0 ? 'var(--color-error)' : 'var(--color-text-muted)'};">${formatCurrency(c.outstanding_payable || 0)}</td>
          <td><span class="status-badge ${c.status === 'active' ? 'paid' : 'pending'}">${escapeHTML(c.status)}</span></td>
          <td style="text-align:right;">
            <div style="display:inline-flex;gap:0.375rem;justify-content:flex-end;align-items:center;">
              <button class="btn btn-sm btn-secondary" onclick="Operations.openCustomerStatement('${c.id}')" title="View Customer Account Statement">Statement</button>
              ${payable > 0 ? `
                <button class="btn btn-sm btn-primary" onclick="Operations.openMakePaymentModal({customerId: '${c.id}', customerName: '${escapeHTML(c.customer_name)}', pending: ${payable}})" title="Pay outstanding payable" style="padding: 0.2rem 0.6rem; font-size: 0.75rem;">Pay</button>
              ` : ''}
              <button class="btn btn-sm btn-secondary" onclick="Operations.viewCustomer('${c.id}')">View</button>
              <button class="btn btn-sm btn-secondary" onclick="Operations.openAdjustCustomerBalance('${c.id}')" title="Adjust Balance">Adjust</button>
              <button class="btn btn-sm btn-secondary" onclick="Operations.openEditCustomer('${c.id}')">Edit</button>
              <button class="btn btn-sm btn-secondary" onclick="Operations.deleteCustomer('${c.id}')" style="color:var(--color-error);">Delete</button>
            </div>
          </td>
        </tr>
      `;
    }).join('');
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
        pendingTbody.innerHTML = `<tr><td colspan="8" style="text-align:center;color:var(--color-text-muted);padding:1.5rem;">No staff members have pending accrued salary balances.</td></tr>`;
      } else {
        pendingTbody.innerHTML = pendingRes.data.map(ps => `
          <tr>
            <td><strong>${escapeHTML(ps.employee_name || 'Staff')}</strong></td>
            <td>${formatCurrency(ps.salary_rate)}</td>
            <td style="color:var(--color-success);font-weight:600;">+${formatCurrency(ps.total_overtime || 0)}</td>
            <td style="font-weight:600;">${formatCurrency(ps.gross_salary || ps.salary_rate)}</td>
            <td style="color:var(--color-error);font-weight:600;">-${formatCurrency(ps.total_advances || 0)}</td>
            <td style="color:var(--color-text-muted);font-weight:600;">-${formatCurrency(ps.total_paid || 0)}</td>
            <td style="font-weight:700;color:var(--color-warning);">${formatCurrency(ps.balance)}</td>
            <td style="text-align:right;">
              <div style="display:inline-flex;gap:0.375rem;justify-content:flex-end;">
                <button class="btn btn-sm btn-secondary" onclick="Operations.openSalaryStatement('${ps.employee_id}', '${escapeHTML(ps.employee_name)}')">Statement</button>
                <button class="btn btn-sm btn-primary" onclick="Operations.openPaySalary('${ps.employee_id}', '${escapeHTML(ps.employee_name)}', '${ps.balance}')">
                  Disburse Payout →
                </button>
              </div>
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
        allTbody.innerHTML = `<tr><td colspan="8" style="text-align:center;color:var(--color-text-muted);padding:1.5rem;">No staff payroll ledgers found.</td></tr>`;
      } else {
        allTbody.innerHTML = allRes.data.map(s => `
          <tr>
            <td><strong>${escapeHTML(s.employee_name || 'Staff')}</strong></td>
            <td>${formatCurrency(s.salary_rate)}</td>
            <td style="color:var(--color-success);font-weight:600;">+${formatCurrency(s.total_overtime || 0)}</td>
            <td style="font-weight:600;">${formatCurrency(s.gross_salary || s.salary_rate)}</td>
            <td style="color:var(--color-error);font-weight:600;">-${formatCurrency(s.total_advances || 0)}</td>
            <td style="color:var(--color-text-muted);font-weight:600;">-${formatCurrency(s.total_paid || 0)}</td>
            <td style="font-weight:700;color:${parseFloat(s.balance) > 0 ? 'var(--color-warning)' : 'inherit'};">${formatCurrency(s.balance)}</td>
            <td style="text-align:right;">
              <div style="display:inline-flex;gap:0.375rem;justify-content:flex-end;">
                <button class="btn btn-sm btn-secondary" onclick="Operations.openSalaryStatement('${s.employee_id}', '${escapeHTML(s.employee_name)}')">Statement</button>
                <button class="btn btn-sm btn-secondary" onclick="Operations.openPaySalary('${s.employee_id}', '${escapeHTML(s.employee_name)}', '${s.balance}')">Pay</button>
                <button class="btn btn-sm btn-secondary" onclick="Operations.openAdjustSalary('${s.employee_id}', '${escapeHTML(s.employee_name)}', '${s.balance}')">Adjust Rate</button>
              </div>
            </td>
          </tr>
        `).join('');
      }
    }
  }

  // 14. Tenant Settings & Profile
  async function loadTenantSettings() {
    try {
      const res = await window.API.get('/tenant/profile');
      if (res.ok && res.data) {
        const t = res.data;
        const nameEl = document.getElementById('settings-tenant-name');
        const emailEl = document.getElementById('settings-tenant-email');
        const phoneEl = document.getElementById('settings-tenant-phone');
        const idEl = document.getElementById('settings-display-id');
        const statusEl = document.getElementById('settings-display-status');
        const createdEl = document.getElementById('settings-display-created');

        if (nameEl) nameEl.value = t.name || '';
        if (emailEl) emailEl.value = t.email || '';
        if (phoneEl) phoneEl.value = t.phone || '';
        if (idEl) idEl.textContent = t.id || '—';
        if (statusEl) {
          statusEl.textContent = t.status || 'active';
          statusEl.className = 'status-badge ' + (t.status === 'active' ? 'paid' : 'danger');
        }
        if (createdEl) createdEl.textContent = formatDateTime(t.created_at);
      }
    } catch (e) {
      console.warn('Failed loading tenant settings:', e);
    }
  }

  async function submitTenantSettings(e) {
    if (e && e.preventDefault) e.preventDefault();
    if (currentRole !== 'admin') {
      showToast('Admin privilege required to update settings', 'error');
      return;
    }

    const btn = document.getElementById('btn-save-settings');
    setButtonLoading(btn, true);

    const name = document.getElementById('settings-tenant-name').value.trim();
    const email = document.getElementById('settings-tenant-email').value.trim();
    const phone = document.getElementById('settings-tenant-phone').value.trim();

    if (!name || !email) {
      showToast('Business name and email are required', 'error');
      setButtonLoading(btn, false);
      return;
    }

    try {
      const res = await window.API.put('/tenant/settings', { name, email, phone });
      setButtonLoading(btn, false);

      if (!res.ok) {
        showToast(res.error || 'Failed to update business settings', 'error');
        return;
      }

      showToast('Business settings updated successfully', 'success');
      const greetingHeading = document.getElementById('greeting-heading');
      if (greetingHeading && res.data && res.data.name) {
        greetingHeading.textContent = `Welcome, ${res.data.name}`;
      }
      loadTenantSettings();
    } catch (err) {
      setButtonLoading(btn, false);
      showToast('Network error while updating settings', 'error');
    }
  }

  // 15. Subscription & Plans Management
  let cachedCurrentSub = null;

  async function loadTenantSubscription() {
    try {
      const res = await window.API.get('/tenant/subscription');
      if (res.ok && res.data) {
        const sub = res.data;
        cachedCurrentSub = sub;

        const planNameEl = document.getElementById('sub-card-plan-name');
        const priceEl = document.getElementById('sub-card-price');
        const noteEl = document.getElementById('sub-card-note');
        const startEl = document.getElementById('sub-card-start-date');
        const endEl = document.getElementById('sub-card-end-date');
        const daysEl = document.getElementById('sub-card-days-remaining');
        const statusPill = document.getElementById('sub-status-pill');
        const expiryAlert = document.getElementById('sub-expired-alert');
        const expiryHint = document.getElementById('sub-card-expiry-hint');

        if (planNameEl) planNameEl.textContent = sub.plan_name || 'Free Starter';
        if (priceEl) priceEl.textContent = formatCurrency(sub.price || 0);
        if (noteEl) noteEl.textContent = sub.note || 'Active Tier Subscription';
        if (startEl) startEl.textContent = sub.start_date ? formatDate(sub.start_date) : 'Registration Date';
        if (endEl) endEl.textContent = sub.end_date ? formatDate(sub.end_date) : 'No Expiry (Lifetime)';
        
        if (daysEl) {
          if (sub.is_expired) {
            daysEl.textContent = '0 Days';
            daysEl.style.color = 'var(--color-error)';
          } else {
            daysEl.textContent = sub.end_date ? `${sub.days_remaining} Days` : 'Unlimited';
            daysEl.style.color = 'var(--color-primary)';
          }
        }

        if (sub.is_expired) {
          if (expiryAlert) expiryAlert.style.display = 'flex';
          if (statusPill) {
            statusPill.textContent = 'Expired';
            statusPill.className = 'status-badge danger';
          }
          if (expiryHint) {
            expiryHint.textContent = 'Subscription Expired';
            expiryHint.style.color = 'var(--color-error)';
          }
        } else {
          if (expiryAlert) expiryAlert.style.display = 'none';
          if (statusPill) {
            statusPill.textContent = 'Active';
            statusPill.className = 'status-badge paid';
          }
          if (expiryHint) {
            expiryHint.textContent = `${sub.days_remaining} days left in billing cycle`;
            expiryHint.style.color = 'var(--color-text-muted)';
          }
        }
      }

      await loadAvailablePlans();
    } catch (e) {
      console.warn('Failed loading subscription:', e);
    }
  }

  async function loadAvailablePlans() {
    const container = document.getElementById('plans-container');
    if (!container) return;

    try {
      const res = await window.API.get('/tenant/subscription/plans');
      if (!res.ok || !res.data) {
        container.innerHTML = '<div style="color:var(--color-text-muted);padding:2rem;text-align:center;grid-column:1/-1;">No subscription plans available at this time.</div>';
        return;
      }

      const plans = res.data;
      if (plans.length === 0) {
        container.innerHTML = '<div style="color:var(--color-text-muted);padding:2rem;text-align:center;grid-column:1/-1;">No subscription plans configured.</div>';
        return;
      }

      const currentPlanId = cachedCurrentSub ? cachedCurrentSub.plan_id : null;
      const isSubExpired = cachedCurrentSub ? cachedCurrentSub.is_expired : false;

      container.innerHTML = plans.map(p => {
        const isCurrent = currentPlanId === p.id && !isSubExpired;
        const priceNum = parseFloat(p.price) || 0;
        const priceDisplay = priceNum === 0 ? 'Free' : formatCurrency(priceNum);

        return `
          <div class="plan-card ${isCurrent ? 'current-active' : ''}">
            ${isCurrent ? '<span class="plan-badge-active">Current Plan</span>' : ''}
            <div style="font-size: 1.125rem; font-weight: 800; color: var(--color-text-main);">${escapeHTML(p.name)}</div>
            <div style="font-size: 0.8125rem; color: var(--color-text-muted); min-height: 2.2rem; margin-top: 0.25rem;">${escapeHTML(p.description || 'Full-featured small business billing & operations.')}</div>
            
            <div class="plan-price-tag">
              ${priceDisplay}
              ${priceNum > 0 ? '<span class="plan-price-period">/ 30 days</span>' : '<span class="plan-price-period">/ starter</span>'}
            </div>

            <ul class="plan-features-list">
              <li>
                <svg fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
                <span>Full Counter & Line Sale Billing</span>
              </li>
              <li>
                <svg fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
                <span>Staff Attendance & Salary Ledger</span>
              </li>
              <li>
                <svg fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
                <span>Live Cash & Multi-Bank Balances</span>
              </li>
              <li>
                <svg fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
                <span>Receivables & Dues Settlement</span>
              </li>
            </ul>

            <div style="margin-top: 1rem;">
              ${isCurrent ? `
                <button type="button" class="btn btn-secondary btn-sm" disabled style="width: 100%; opacity: 0.85; cursor: default;">
                  ✓ Currently Active
                </button>
              ` : `
                <button type="button" class="btn btn-primary btn-sm" style="width: 100%;" onclick="Operations.openPurchaseModal('${p.id}', '${escapeHTML(p.name)}', ${priceNum})">
                  ${isSubExpired && currentPlanId === p.id ? 'Renew Plan' : 'Select / Upgrade'}
                </button>
              `}
            </div>
          </div>
        `;
      }).join('');
    } catch (e) {
      container.innerHTML = '<div style="color:var(--color-error);padding:2rem;text-align:center;grid-column:1/-1;">Failed to load available plans.</div>';
    }
  }

  function scrollToPlans() {
    const el = document.getElementById('available-plans-section');
    if (el) el.scrollIntoView({ behavior: 'smooth' });
  }

  function openPurchaseModal(planId, planName, planPrice) {
    if (currentRole !== 'admin') {
      showToast('Admin privilege required to purchase or upgrade plans', 'error');
      return;
    }

    const idInput = document.getElementById('purchase-plan-id');
    const nameEl = document.getElementById('purchase-modal-plan-name');
    const priceEl = document.getElementById('purchase-modal-plan-price');

    if (idInput) idInput.value = planId;
    if (nameEl) nameEl.textContent = planName || 'Pro Tier';
    if (priceEl) priceEl.textContent = planPrice === 0 ? 'Free' : formatCurrency(planPrice);

    openModal('modal-purchase-plan');
  }

  async function submitPurchasePlan(e) {
    if (e && e.preventDefault) e.preventDefault();
    if (currentRole !== 'admin') {
      showToast('Admin privilege required to purchase plans', 'error');
      return;
    }

    const planId = document.getElementById('purchase-plan-id').value;
    const paymentMethod = document.getElementById('purchase-payment-method').value;
    const btn = document.getElementById('btn-submit-purchase');

    setButtonLoading(btn, true);

    try {
      const res = await window.API.post('/tenant/subscription/purchase', {
        plan_id: planId,
        payment_method: paymentMethod
      });
      setButtonLoading(btn, false);

      if (!res.ok) {
        showToast(res.error || 'Failed to process subscription purchase', 'error');
        return;
      }

      closeModal('modal-purchase-plan');
      showToast('Subscription plan activated successfully!', 'success');
      await loadTenantSubscription();
      loadTenantTransactions();
    } catch (err) {
      setButtonLoading(btn, false);
      showToast('Network error during plan purchase', 'error');
    }
  }

  // 16. Tenant Transactions History
  async function loadTenantTransactions() {
    const tbody = document.getElementById('table-transactions-body');
    const countBadge = document.getElementById('txn-count-badge');
    if (!tbody) return;

    try {
      const res = await window.API.get('/tenant/subscription/transactions?page=1&page_size=50');
      if (!res.ok || !res.data) {
        tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;color:var(--color-text-muted);padding:2rem;">No billing transactions recorded yet.</td></tr>';
        if (countBadge) countBadge.textContent = '0';
        return;
      }

      const txns = res.data;
      if (countBadge) countBadge.textContent = String(txns.length);

      if (txns.length === 0) {
        tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;color:var(--color-text-muted);padding:2rem;">No billing transactions found for this business.</td></tr>';
        return;
      }

      tbody.innerHTML = txns.map(t => {
        let statusBadge = '<span class="status-badge paid">Completed</span>';
        if (t.status === 'pending') {
          statusBadge = '<span class="status-badge pending">Pending</span>';
        } else if (t.status === 'failed') {
          statusBadge = '<span class="status-badge danger">Failed</span>';
        }

        return `
          <tr>
            <td><code style="font-family: var(--font-mono); font-size: 0.75rem;">${escapeHTML(t.transaction_id)}</code></td>
            <td><strong>${escapeHTML(t.plan_name || 'Standard Plan')}</strong></td>
            <td><strong style="color: var(--color-primary);">${formatCurrency(t.amount)}</strong></td>
            <td><span class="status-badge cleared" style="text-transform: uppercase;">${escapeHTML(t.payment_method)}</span></td>
            <td>${statusBadge}</td>
            <td style="font-size: 0.8125rem; color: var(--color-text-muted);">${formatDateTime(t.created_at)}</td>
            <td style="font-size: 0.8125rem; color: var(--color-text-muted);">${escapeHTML(t.failure_reason || '—')}</td>
          </tr>
        `;
      }).join('');
    } catch (e) {
      tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;color:var(--color-error);padding:2rem;">Failed to load transactions.</td></tr>';
    }
  }

  // Legacy profile compatibility loader
  async function loadProfile() {
    loadTenantSettings();
    loadTenantSubscription();
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

  // Bank & Payment Row Helpers
  function renderBankSelectOptions(selectedId = '') {
    const activeBanks = cachedBanks.filter(b => b.status === 'active' || !b.status);
    return '<option value="">Select Bank Account...</option>' +
      activeBanks.map(b => `<option value="${b.id}" ${b.id === selectedId ? 'selected' : ''}>${escapeHTML(b.bank_name)} (${escapeHTML(b.account_number || 'Main')})</option>`).join('');
  }

  // Line Sale: Customer Balance & Bank Rows
  async function handleLineSaleCustomerChange() {
    const sel = document.getElementById('ls-form-customer');
    const custId = sel ? sel.value : null;
    const box = document.getElementById('ls-customer-balance-box');

    if (!custId) {
      lineSaleCustomerBalance = null;
      if (box) box.style.display = 'none';
      calcLineSaleDue();
      return;
    }

    const badge = document.getElementById('ls-cust-balance-badge');
    if (badge) {
      badge.textContent = 'Fetching...';
      badge.style.cssText = 'background: rgba(148, 163, 184, 0.2); color: #475569;';
    }
    if (box) box.style.display = 'block';

    const res = await window.API.get(`/tenant/customers/${custId}/balance`);
    if (!res.ok || !res.data) {
      const cached = cachedCustomers.find(c => c.id === custId);
      lineSaleCustomerBalance = cached ? parseFloat(cached.current_balance) || 0 : 0;
    } else {
      lineSaleCustomerBalance = parseFloat(res.data.current_balance) || 0;
    }

    calcLineSaleDue();
  }

  function addLineSaleBankRow(bankId = '', amount = '') {
    const container = document.getElementById('ls-bank-payments-container');
    if (!container) return;
    lineSaleBankRowCount++;
    const rowId = `ls-bank-row-${lineSaleBankRowCount}`;

    const row = document.createElement('div');
    row.id = rowId;
    row.className = 'bank-payment-row';
    row.innerHTML = `
      <select class="form-control select-bank ls-bank-select" required onchange="Operations.validateLineSaleBanks(); Operations.calcLineSaleDue();">
        ${renderBankSelectOptions(bankId)}
      </select>
      <input type="number" step="0.01" class="form-control bank-amount-input ls-bank-amount" min="0.01" required placeholder="Amount (₹)" value="${amount}" oninput="Operations.calcLineSaleDue();">
      <button type="button" class="btn-remove-bank" title="Remove Bank Payment" onclick="Operations.removeLineSaleBankRow('${rowId}')">✕</button>
    `;
    container.appendChild(row);
    validateLineSaleBanks();
    calcLineSaleDue();
  }

  function removeLineSaleBankRow(rowId) {
    const row = document.getElementById(rowId);
    if (row) row.remove();
    validateLineSaleBanks();
    calcLineSaleDue();
  }

  function validateLineSaleBanks() {
    const selects = document.querySelectorAll('#ls-bank-payments-container .ls-bank-select');
    const warning = document.getElementById('ls-bank-duplicate-warning');
    const selected = [];
    let hasDuplicate = false;

    selects.forEach(s => {
      const val = s.value;
      if (val) {
        if (selected.includes(val)) {
          hasDuplicate = true;
          s.style.borderColor = 'var(--color-error)';
        } else {
          selected.push(val);
          s.style.borderColor = '';
        }
      } else {
        s.style.borderColor = '';
      }
    });

    if (warning) warning.style.display = hasDuplicate ? 'block' : 'none';
    return !hasDuplicate;
  }

  // Line Sale: Calculations & Creation
  function calcLineSaleDue() {
    const total = parseFloat(document.getElementById('ls-form-amount')?.value) || 0;
    const cash = parseFloat(document.getElementById('ls-form-cash')?.value) || 0;

    let bankTotal = 0;
    document.querySelectorAll('#ls-bank-payments-container .ls-bank-amount').forEach(inp => {
      const v = parseFloat(inp.value) || 0;
      if (v > 0) bankTotal += v;
    });

    const collected = cash + bankTotal;
    const due = Math.max(0, total - collected);

    const elCash = document.getElementById('ls-calc-cash');
    const elBank = document.getElementById('ls-calc-bank');
    const elCollected = document.getElementById('ls-calc-collected');
    const elDue = document.getElementById('ls-calc-due');
    const elWarning = document.getElementById('ls-calc-warning');
    const btn = document.getElementById('btn-submit-line-sale');

    if (elCash) elCash.textContent = formatCurrency(cash);
    if (elBank) elBank.textContent = formatCurrency(bankTotal);
    if (elCollected) elCollected.textContent = formatCurrency(collected);
    if (elDue) elDue.textContent = formatCurrency(due);

    // Update customer live balance preview if selected
    if (lineSaleCustomerBalance !== null) {
      const badge = document.getElementById('ls-cust-balance-badge');
      const elSaleDue = document.getElementById('ls-cust-sale-due');
      const elResulting = document.getElementById('ls-cust-resulting-balance');
      const bal = lineSaleCustomerBalance;

      if (badge) {
        if (bal > 0) {
          badge.textContent = `${formatCurrency(bal)} (Outstanding Due)`;
          badge.style.cssText = 'background: rgba(245, 158, 11, 0.15); color: #d97706; font-weight: 700;';
        } else if (bal < 0) {
          badge.textContent = `${formatCurrency(Math.abs(bal))} (Credit / Advance)`;
          badge.style.cssText = 'background: rgba(59, 130, 246, 0.15); color: #2563eb; font-weight: 700;';
        } else {
          badge.textContent = '₹0.00 (No Outstanding)';
          badge.style.cssText = 'background: rgba(16, 185, 129, 0.15); color: #059669; font-weight: 700;';
        }
      }
      if (elSaleDue) elSaleDue.textContent = formatCurrency(due);
      if (elResulting) {
        const resulting = bal + due;
        elResulting.textContent = formatCurrency(resulting);
        elResulting.style.color = resulting > 0 ? 'var(--color-warning)' : resulting < 0 ? '#2563eb' : 'var(--color-success)';
      }
    }

    const hasDuplicateBanks = !validateLineSaleBanks();
    if (total > 0 && collected > total) {
      if (elWarning) elWarning.style.display = 'block';
      if (elCollected) elCollected.style.color = 'var(--color-error)';
      if (btn) btn.disabled = true;
    } else if (hasDuplicateBanks) {
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

    // Collect bank payments from dynamic rows
    const bankPayments = [];
    const seenBanks = new Set();
    let hasBankError = false;

    document.querySelectorAll('#ls-bank-payments-container .bank-payment-row').forEach(row => {
      const select = row.querySelector('.ls-bank-select');
      const input = row.querySelector('.ls-bank-amount');
      const bankId = select?.value;
      const bAmt = parseFloat(input?.value) || 0;

      if (!bankId) {
        showModalError('modal-add-line-sale', 'Please select a bank account for all bank payment rows.');
        hasBankError = true;
        return;
      }
      if (bAmt <= 0) {
        showModalError('modal-add-line-sale', 'Bank payment amount must be greater than zero.');
        hasBankError = true;
        return;
      }
      if (seenBanks.has(bankId)) {
        showModalError('modal-add-line-sale', 'Duplicate bank selected in payment rows.');
        hasBankError = true;
        return;
      }
      seenBanks.add(bankId);
      const bObj = cachedBanks.find(b => b.id === bankId);
      bankPayments.push({
        bank_id: bankId,
        bank_name: bObj ? bObj.bank_name : '',
        amount: bAmt,
        note: 'Bank payment'
      });
    });

    if (hasBankError) return;

    let totalBank = 0;
    bankPayments.forEach(bp => { totalBank += bp.amount; });
    const collected = cashIn + totalBank;

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
    bankPayments.forEach(bp => {
      payments.push({
        payment_method: 'bank',
        bank_id: bp.bank_id,
        amount: bp.amount,
        note: bp.note || 'Bank payment'
      });
    });

    const payload = {
      customer_id: customerID,
      route: document.getElementById('ls-form-route').value.trim(),
      salesman: document.getElementById('ls-form-salesman').value.trim(),
      total_amount: amount,
      total_cash_in: cashIn,
      cash_amount: cashIn,
      bank_amount: totalBank,
      bank_id: bankPayments.length > 0 ? bankPayments[0].bank_id : null,
      bank_payments: bankPayments,
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

  // Counter Sale: Multi-Bank Rows & Calculations
  function addCounterSaleBankRow(bankId = '', amount = '') {
    const container = document.getElementById('cs-bank-payments-container');
    if (!container) return;
    counterSaleBankRowCount++;
    const rowId = `cs-bank-row-${counterSaleBankRowCount}`;

    const row = document.createElement('div');
    row.id = rowId;
    row.className = 'bank-payment-row';
    row.innerHTML = `
      <select class="form-control select-bank cs-bank-select" required onchange="Operations.validateCounterSaleBanks(); Operations.calcCounterSaleDue();">
        ${renderBankSelectOptions(bankId)}
      </select>
      <input type="number" step="0.01" class="form-control bank-amount-input cs-bank-amount" min="0.01" required placeholder="Amount (₹)" value="${amount}" oninput="Operations.calcCounterSaleDue();">
      <button type="button" class="btn-remove-bank" title="Remove Bank Payment" onclick="Operations.removeCounterSaleBankRow('${rowId}')">✕</button>
    `;
    container.appendChild(row);
    validateCounterSaleBanks();
    calcCounterSaleDue();
  }

  function removeCounterSaleBankRow(rowId) {
    const row = document.getElementById(rowId);
    if (row) row.remove();
    validateCounterSaleBanks();
    calcCounterSaleDue();
  }

  function validateCounterSaleBanks() {
    const selects = document.querySelectorAll('#cs-bank-payments-container .cs-bank-select');
    const warning = document.getElementById('cs-bank-duplicate-warning');
    const selected = [];
    let hasDuplicate = false;

    selects.forEach(s => {
      const val = s.value;
      if (val) {
        if (selected.includes(val)) {
          hasDuplicate = true;
          s.style.borderColor = 'var(--color-error)';
        } else {
          selected.push(val);
          s.style.borderColor = '';
        }
      } else {
        s.style.borderColor = '';
      }
    });

    if (warning) warning.style.display = hasDuplicate ? 'block' : 'none';
    return !hasDuplicate;
  }

  function calcCounterSaleDue() {
    const total = parseFloat(document.getElementById('cs-form-total')?.value) || 0;
    const cash = parseFloat(document.getElementById('cs-form-cash')?.value) || 0;

    let bankTotal = 0;
    document.querySelectorAll('#cs-bank-payments-container .cs-bank-amount').forEach(inp => {
      const v = parseFloat(inp.value) || 0;
      if (v > 0) bankTotal += v;
    });

    const collected = cash + bankTotal;
    const due = Math.max(0, total - collected);

    const elCash = document.getElementById('cs-calc-cash');
    const elBank = document.getElementById('cs-calc-bank');
    const elCollected = document.getElementById('cs-calc-collected');
    const elDue = document.getElementById('cs-calc-due');
    const elWarning = document.getElementById('cs-calc-warning');
    const btn = document.getElementById('btn-submit-counter-sale');

    if (elCash) elCash.textContent = formatCurrency(cash);
    if (elBank) elBank.textContent = formatCurrency(bankTotal);
    if (elCollected) elCollected.textContent = formatCurrency(collected);
    if (elDue) elDue.textContent = formatCurrency(due);

    const hasDuplicateBanks = !validateCounterSaleBanks();
    if (total > 0 && collected > total) {
      if (elWarning) elWarning.style.display = 'block';
      if (elCollected) elCollected.style.color = 'var(--color-error)';
      if (btn) btn.disabled = true;
    } else if (hasDuplicateBanks) {
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
    const cash = parseFloat(document.getElementById('cs-form-cash').value) || 0;

    // Collect bank payments from dynamic rows
    const bankPayments = [];
    const seenBanks = new Set();
    let hasBankError = false;

    document.querySelectorAll('#cs-bank-payments-container .bank-payment-row').forEach(row => {
      const select = row.querySelector('.cs-bank-select');
      const input = row.querySelector('.cs-bank-amount');
      const bankId = select?.value;
      const bAmt = parseFloat(input?.value) || 0;

      if (!bankId) {
        showModalError('modal-add-counter-sale', 'Please select a bank account for all bank payment rows.');
        hasBankError = true;
        return;
      }
      if (bAmt <= 0) {
        showModalError('modal-add-counter-sale', 'Bank payment amount must be greater than zero.');
        hasBankError = true;
        return;
      }
      if (seenBanks.has(bankId)) {
        showModalError('modal-add-counter-sale', 'Duplicate bank selected in payment rows.');
        hasBankError = true;
        return;
      }
      seenBanks.add(bankId);
      const bObj = cachedBanks.find(b => b.id === bankId);
      bankPayments.push({
        bank_id: bankId,
        bank_name: bObj ? bObj.bank_name : '',
        amount: bAmt,
        note: 'Counter sale bank payment'
      });
    });

    if (hasBankError) return;

    let totalBank = 0;
    bankPayments.forEach(bp => { totalBank += bp.amount; });
    const collected = cash + totalBank;

    if (total > 0 && collected > total) {
      showModalError('modal-add-counter-sale', 'Collected amount cannot exceed the total sale amount.');
      return;
    }

    setButtonLoading(btn, true);

    let paymentMethod = 'cash';
    if (cash > 0 && totalBank > 0) {
      paymentMethod = 'split';
    } else if (totalBank > 0 && cash === 0) {
      paymentMethod = 'bank';
    }

    const payload = {
      item: document.getElementById('cs-form-item').value.trim(),
      price: price > 0 ? price : total,
      total_amount: total,
      payment_method: paymentMethod,
      cash: cash,
      cash_amount: cash,
      account: totalBank,
      bank_amount: totalBank,
      bank_id: bankPayments.length > 0 ? bankPayments[0].bank_id : null,
      bank_payments: bankPayments
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

  // Purchase: Multi-Bank Helpers & Live Balance Calculation
  function addPurchaseBankRow(bankId = '', amount = '') {
    const container = document.getElementById('purch-bank-payments-container');
    if (!container) return;
    purchaseBankRowCount++;
    const rowId = `purch-bank-row-${purchaseBankRowCount}`;

    const row = document.createElement('div');
    row.id = rowId;
    row.className = 'bank-payment-row';
    row.innerHTML = `
      <select class="form-control select-bank purch-bank-select" required onchange="Operations.validatePurchaseBanks(); Operations.calcPurchaseBalance();">
        ${renderBankSelectOptions(bankId)}
      </select>
      <input type="number" step="0.01" class="form-control bank-amount-input purch-bank-amount" min="0.01" required placeholder="Amount (₹)" value="${amount}" oninput="Operations.calcPurchaseBalance();">
      <button type="button" class="btn-remove-bank" title="Remove Bank Payment" onclick="Operations.removePurchaseBankRow('${rowId}')">✕</button>
    `;
    container.appendChild(row);
    validatePurchaseBanks();
    calcPurchaseBalance();
  }

  function removePurchaseBankRow(rowId) {
    const row = document.getElementById(rowId);
    if (row) row.remove();
    validatePurchaseBanks();
    calcPurchaseBalance();
  }

  function validatePurchaseBanks() {
    const selects = document.querySelectorAll('#purch-bank-payments-container .purch-bank-select');
    const warning = document.getElementById('purch-bank-duplicate-warning');
    const selected = [];
    let hasDuplicate = false;

    selects.forEach(s => {
      const val = s.value;
      if (val) {
        if (selected.includes(val)) {
          hasDuplicate = true;
          s.style.borderColor = 'var(--color-error)';
        } else {
          selected.push(val);
          s.style.borderColor = '';
        }
      } else {
        s.style.borderColor = '';
      }
    });

    if (warning) warning.style.display = hasDuplicate ? 'block' : 'none';
    return !hasDuplicate;
  }

  async function handlePurchaseCustomerChange() {
    const sel = document.getElementById('purch-form-customer');
    const custId = sel ? sel.value : null;
    const box = document.getElementById('purch-customer-balance-box');

    if (!custId) {
      purchaseCustomerBalance = null;
      if (box) box.style.display = 'none';
      calcPurchaseBalance();
      return;
    }

    const badge = document.getElementById('purch-cust-balance-badge');
    if (badge) {
      badge.textContent = 'Fetching...';
      badge.style.cssText = 'background: rgba(148, 163, 184, 0.2); color: #475569;';
    }
    if (box) box.style.display = 'block';

    const res = await window.API.get(`/tenant/customers/${custId}/balance`);
    if (!res.ok || !res.data) {
      const cached = cachedCustomers.find(c => c.id === custId);
      purchaseCustomerBalance = cached ? parseFloat(cached.outstanding_payable || cached.current_balance) || 0 : 0;
    } else {
      purchaseCustomerBalance = parseFloat(res.data.outstanding_payable) || 0;
    }

    calcPurchaseBalance();
  }

  function calcPurchaseBalance() {
    const total = parseFloat(document.getElementById('purch-form-total')?.value) || 0;
    const cash = parseFloat(document.getElementById('purch-form-cash')?.value) || 0;

    let bankTotal = 0;
    document.querySelectorAll('#purch-bank-payments-container .purch-bank-amount').forEach(inp => {
      const v = parseFloat(inp.value) || 0;
      if (v > 0) bankTotal += v;
    });

    const totalPaid = cash + bankTotal;
    const pendingDue = Math.max(0, total - totalPaid);

    const elCash = document.getElementById('purch-calc-cash');
    const elBank = document.getElementById('purch-calc-bank');
    const elPaid = document.getElementById('purch-calc-paid');
    const elDue = document.getElementById('purch-calc-due');
    const statusBadge = document.getElementById('purch-calc-status-badge');
    const warn = document.getElementById('purch-calc-warning');

    if (elCash) elCash.textContent = formatCurrency(cash);
    if (elBank) elBank.textContent = formatCurrency(bankTotal);
    if (elPaid) elPaid.textContent = formatCurrency(totalPaid);
    if (elDue) elDue.textContent = formatCurrency(pendingDue);

    if (warn) {
      warn.style.display = totalPaid > total && total > 0 ? 'block' : 'none';
    }

    if (statusBadge) {
      if (totalPaid >= total && total > 0) {
        statusBadge.className = 'status-badge paid';
        statusBadge.textContent = 'PAID';
      } else if (totalPaid > 0 && totalPaid < total) {
        statusBadge.className = 'status-badge pending';
        statusBadge.textContent = 'PARTIALLY PAID';
      } else {
        statusBadge.className = 'status-badge danger';
        statusBadge.textContent = 'UNPAID';
      }
    }

    // Update customer live balance preview if selected
    if (purchaseCustomerBalance !== null) {
      const badge = document.getElementById('purch-cust-balance-badge');
      const elBillDue = document.getElementById('purch-cust-bill-due');
      const elResulting = document.getElementById('purch-cust-resulting-balance');

      const bal = purchaseCustomerBalance;
      if (badge) {
        if (bal > 0) {
          badge.textContent = `${formatCurrency(bal)} (Outstanding Payable)`;
          badge.style.cssText = 'background: rgba(239, 68, 68, 0.15); color: #dc2626; font-weight: 700;';
        } else {
          badge.textContent = '₹0.00 (No Outstanding)';
          badge.style.cssText = 'background: rgba(16, 185, 129, 0.15); color: #059669; font-weight: 700;';
        }
      }

      if (elBillDue) {
        elBillDue.textContent = formatCurrency(pendingDue);
      }

      if (elResulting) {
        const resulting = bal + pendingDue;
        elResulting.textContent = formatCurrency(resulting);
        elResulting.style.color = resulting > 0 ? 'var(--color-error)' : 'var(--color-success)';
      }
    }
  }

  // Purchase: Create
  async function submitAddPurchase(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-purchase');

    if (!validatePurchaseBanks()) {
      showToast('Each bank account can only be selected once per purchase', 'error');
      return;
    }

    const customerID = document.getElementById('purch-form-customer')?.value || null;
    const item = document.getElementById('purch-form-item').value.trim();
    const quantity = parseInt(document.getElementById('purch-form-quantity').value) || 1;
    const total = parseFloat(document.getElementById('purch-form-total').value) || 0;
    const cash = parseFloat(document.getElementById('purch-form-cash')?.value) || 0;

    const bankPayments = [];
    let bankTotal = 0;
    document.querySelectorAll('#purch-bank-payments-container .bank-payment-row').forEach(row => {
      const select = row.querySelector('.purch-bank-select');
      const amountInp = row.querySelector('.purch-bank-amount');
      const bankId = select ? select.value : '';
      const amt = amountInp ? parseFloat(amountInp.value) || 0 : 0;
      if (bankId && amt > 0) {
        bankPayments.push({
          bank_id: bankId,
          amount: amt
        });
        bankTotal += amt;
      }
    });

    const totalPaid = cash + bankTotal;
    if (totalPaid > total && total > 0) {
      showToast('Total paid cannot exceed purchase bill amount', 'error');
      return;
    }

    setButtonLoading(btn, true);

    const payload = {
      customer_id: customerID && customerID.trim() !== '' ? customerID : null,
      item: item,
      quantity: quantity,
      total_amount: total,
      total_paid: totalPaid,
      cash_amount: cash > 0 ? cash : null,
      bank_payments: bankPayments
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
    loadCustomers();
    loadCashBank();
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
    const custSel = document.getElementById('edit-purch-customer');
    if (custSel) custSel.value = p.customer_id || '';
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

    const customerID = document.getElementById('edit-purch-customer')?.value || null;
    const qty = parseInt(document.getElementById('edit-purch-quantity').value) || 1;
    const total = parseFloat(document.getElementById('edit-purch-total').value) || 0;
    const paid = parseFloat(document.getElementById('edit-purch-paid').value) || 0;

    const payload = {
      customer_id: customerID && customerID.trim() !== '' ? customerID : null,
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
    loadCustomers();
  }

  // Expense: Category Change Handler
  function handleExpenseCategoryChange(val) {
    const empGroup = document.getElementById('exp-form-emp-group');
    const empSelect = document.getElementById('exp-form-emp');
    const itemInput = document.getElementById('exp-form-item');
    if (val === 'employee_advance') {
      if (empGroup) empGroup.style.display = 'block';
      if (empSelect) empSelect.required = true;
      if (itemInput && (!itemInput.value || itemInput.value === 'Shop rent' || itemInput.value === 'Electricity bill')) {
        itemInput.value = 'Staff Advance Payment';
      }
    } else {
      if (empGroup) empGroup.style.display = 'none';
      if (empSelect) empSelect.required = false;
    }
  }

  // Expense: Payment Breakdown & Bank Row Helpers
  function handleExpensePaymentMethodChange(method) {
    const cashSection = document.getElementById('exp-cash-section');
    const bankSection = document.getElementById('exp-bank-section');
    const cashInput = document.getElementById('exp-form-cash');
    const totalAmount = parseFloat(document.getElementById('exp-form-amount')?.value) || 0;
    const cashHint = document.getElementById('exp-cash-hint');

    document.querySelectorAll('#modal-add-expense .payment-method-option').forEach(el => {
      const radio = el.querySelector('input[type="radio"]');
      if (radio && radio.value === method) {
        radio.checked = true;
        el.classList.add('active');
      } else {
        el.classList.remove('active');
      }
    });

    if (method === 'cash') {
      if (cashSection) cashSection.style.display = 'block';
      if (bankSection) bankSection.style.display = 'none';
      if (cashHint) cashHint.textContent = 'Full expense amount settled via cash in drawer.';
      if (cashInput) cashInput.value = totalAmount > 0 ? totalAmount.toFixed(2) : '0.00';
      const container = document.getElementById('exp-bank-payments-container');
      if (container) container.innerHTML = '';
      const warn = document.getElementById('exp-bank-duplicate-warning');
      if (warn) warn.style.display = 'none';
    } else if (method === 'bank') {
      if (cashSection) cashSection.style.display = 'none';
      if (bankSection) bankSection.style.display = 'block';
      if (cashInput) cashInput.value = '0.00';
      const container = document.getElementById('exp-bank-payments-container');
      if (container && container.children.length === 0) {
        addExpenseBankRow('', totalAmount > 0 ? totalAmount.toFixed(2) : '');
      }
    } else if (method === 'cash_bank') {
      if (cashSection) cashSection.style.display = 'block';
      if (bankSection) bankSection.style.display = 'block';
      if (cashHint) cashHint.textContent = 'Portion settled via cash in drawer.';
      const container = document.getElementById('exp-bank-payments-container');
      if (container && container.children.length === 0) {
        addExpenseBankRow('', '');
      }
    }
    calcExpenseBreakdown();
  }

  function addExpenseBankRow(bankId = '', amount = '') {
    const container = document.getElementById('exp-bank-payments-container');
    if (!container) return;
    expenseBankRowCount++;
    const rowId = `exp-bank-row-${expenseBankRowCount}`;

    const row = document.createElement('div');
    row.id = rowId;
    row.className = 'bank-payment-row';
    row.innerHTML = `
      <select class="form-control select-bank exp-bank-select" required onchange="Operations.validateExpenseBanks(); Operations.calcExpenseBreakdown();">
        ${renderBankSelectOptions(bankId)}
      </select>
      <input type="number" step="0.01" class="form-control bank-amount-input exp-bank-amount" min="0.01" required placeholder="Amount (₹)" value="${amount}" oninput="Operations.calcExpenseBreakdown();">
      <button type="button" class="btn-remove-bank" title="Remove Bank Payment" onclick="Operations.removeExpenseBankRow('${rowId}')">✕</button>
    `;
    container.appendChild(row);
    validateExpenseBanks();
    calcExpenseBreakdown();
  }

  function removeExpenseBankRow(rowId) {
    const row = document.getElementById(rowId);
    if (row) row.remove();
    validateExpenseBanks();
    calcExpenseBreakdown();
  }

  function validateExpenseBanks() {
    const selects = document.querySelectorAll('#exp-bank-payments-container .exp-bank-select');
    const warning = document.getElementById('exp-bank-duplicate-warning');
    const selected = [];
    let hasDuplicate = false;

    selects.forEach(s => {
      const val = s.value;
      if (val) {
        if (selected.includes(val)) {
          hasDuplicate = true;
          s.style.borderColor = 'var(--color-error)';
        } else {
          selected.push(val);
          s.style.borderColor = '';
        }
      } else {
        s.style.borderColor = '';
      }
    });

    if (warning) warning.style.display = hasDuplicate ? 'block' : 'none';
    return !hasDuplicate;
  }

  function calcExpenseBreakdown() {
    const totalAmount = parseFloat(document.getElementById('exp-form-amount')?.value) || 0;
    const method = document.querySelector('input[name="exp-payment-method"]:checked')?.value || 'cash';
    const cashInput = document.getElementById('exp-form-cash');
    const warnEl = document.getElementById('exp-calc-warning');
    const submitBtn = document.getElementById('btn-submit-expense');

    let cashPaid = 0;
    let bankPaid = 0;

    if (method === 'cash') {
      cashPaid = totalAmount;
      if (cashInput && document.activeElement !== cashInput) {
        cashInput.value = totalAmount > 0 ? totalAmount.toFixed(2) : '0.00';
      }
    } else if (method === 'bank') {
      cashPaid = 0;
      document.querySelectorAll('#exp-bank-payments-container .exp-bank-amount').forEach(inp => {
        bankPaid += parseFloat(inp.value) || 0;
      });
    } else if (method === 'cash_bank') {
      cashPaid = parseFloat(cashInput?.value) || 0;
      document.querySelectorAll('#exp-bank-payments-container .exp-bank-amount').forEach(inp => {
        bankPaid += parseFloat(inp.value) || 0;
      });
    }

    const totalPaid = Math.round((cashPaid + bankPaid) * 100) / 100;
    const diff = Math.round((totalAmount - totalPaid) * 100) / 100;

    const totalEl = document.getElementById('exp-calc-total');
    const cashEl = document.getElementById('exp-calc-cash');
    const bankEl = document.getElementById('exp-calc-bank');
    const paidEl = document.getElementById('exp-calc-paid');
    const remEl = document.getElementById('exp-calc-remaining');
    const cashRow = document.getElementById('exp-calc-cash-row');
    const bankRow = document.getElementById('exp-calc-bank-row');

    if (totalEl) totalEl.textContent = formatCurrency(totalAmount);
    if (cashEl) cashEl.textContent = formatCurrency(cashPaid);
    if (bankEl) bankEl.textContent = formatCurrency(bankPaid);
    if (paidEl) paidEl.textContent = formatCurrency(totalPaid);
    if (remEl) remEl.textContent = formatCurrency(diff);

    if (cashRow) cashRow.style.display = (method === 'bank') ? 'none' : 'flex';
    if (bankRow) bankRow.style.display = (method === 'cash') ? 'none' : 'flex';

    const hasDuplicate = !validateExpenseBanks();
    const isMismatch = Math.abs(diff) > 0.005;
    const hasZeroTotal = totalAmount <= 0;

    let hasEmptyBank = false;
    if (method === 'bank' || method === 'cash_bank') {
      const bankRows = document.querySelectorAll('#exp-bank-payments-container .bank-payment-row');
      if (bankRows.length === 0) {
        hasEmptyBank = true;
      } else {
        bankRows.forEach(r => {
          const sel = r.querySelector('.exp-bank-select')?.value;
          const amt = parseFloat(r.querySelector('.exp-bank-amount')?.value) || 0;
          if (!sel || amt <= 0) hasEmptyBank = true;
        });
      }
    }

    if (warnEl) {
      if (hasZeroTotal) {
        warnEl.textContent = '⚠️ Please enter an expense amount.';
        warnEl.style.display = 'block';
      } else if (hasDuplicate) {
        warnEl.textContent = '⚠️ Each bank account can only be selected once per expense.';
        warnEl.style.display = 'block';
      } else if (hasEmptyBank) {
        warnEl.textContent = '⚠️ Please select a bank account and enter a valid amount for all bank rows.';
        warnEl.style.display = 'block';
      } else if (isMismatch) {
        warnEl.textContent = `⚠️ Payment breakdown must equal the expense amount (${formatCurrency(totalAmount)}).`;
        warnEl.style.display = 'block';
      } else {
        warnEl.style.display = 'none';
      }
    }

    if (remEl) {
      remEl.style.color = isMismatch ? 'var(--color-error)' : 'var(--color-success)';
    }

    if (submitBtn) {
      submitBtn.disabled = hasZeroTotal || hasDuplicate || hasEmptyBank || isMismatch;
    }
  }

  // Edit Expense: Helpers
  function handleEditExpensePaymentMethodChange(method) {
    const cashSection = document.getElementById('edit-exp-cash-section');
    const bankSection = document.getElementById('edit-exp-bank-section');
    const cashInput = document.getElementById('edit-exp-form-cash');
    const totalAmount = parseFloat(document.getElementById('edit-exp-amount')?.value) || 0;
    const cashHint = document.getElementById('edit-exp-cash-hint');

    document.querySelectorAll('#modal-edit-expense .payment-method-option').forEach(el => {
      const radio = el.querySelector('input[type="radio"]');
      if (radio && radio.value === method) {
        radio.checked = true;
        el.classList.add('active');
      } else {
        el.classList.remove('active');
      }
    });

    if (method === 'cash') {
      if (cashSection) cashSection.style.display = 'block';
      if (bankSection) bankSection.style.display = 'none';
      if (cashHint) cashHint.textContent = 'Full expense amount settled via cash in drawer.';
      if (cashInput) cashInput.value = totalAmount > 0 ? totalAmount.toFixed(2) : '0.00';
      const container = document.getElementById('edit-exp-bank-payments-container');
      if (container) container.innerHTML = '';
      const warn = document.getElementById('edit-exp-bank-duplicate-warning');
      if (warn) warn.style.display = 'none';
    } else if (method === 'bank') {
      if (cashSection) cashSection.style.display = 'none';
      if (bankSection) bankSection.style.display = 'block';
      if (cashInput) cashInput.value = '0.00';
      const container = document.getElementById('edit-exp-bank-payments-container');
      if (container && container.children.length === 0) {
        addEditExpenseBankRow('', totalAmount > 0 ? totalAmount.toFixed(2) : '');
      }
    } else if (method === 'cash_bank') {
      if (cashSection) cashSection.style.display = 'block';
      if (bankSection) bankSection.style.display = 'block';
      if (cashHint) cashHint.textContent = 'Portion settled via cash in drawer.';
      const container = document.getElementById('edit-exp-bank-payments-container');
      if (container && container.children.length === 0) {
        addEditExpenseBankRow('', '');
      }
    }
    calcEditExpenseBreakdown();
  }

  function addEditExpenseBankRow(bankId = '', amount = '') {
    const container = document.getElementById('edit-exp-bank-payments-container');
    if (!container) return;
    editExpenseBankRowCount++;
    const rowId = `edit-exp-bank-row-${editExpenseBankRowCount}`;

    const row = document.createElement('div');
    row.id = rowId;
    row.className = 'bank-payment-row';
    row.innerHTML = `
      <select class="form-control select-bank edit-exp-bank-select" required onchange="Operations.validateEditExpenseBanks(); Operations.calcEditExpenseBreakdown();">
        ${renderBankSelectOptions(bankId)}
      </select>
      <input type="number" step="0.01" class="form-control bank-amount-input edit-exp-bank-amount" min="0.01" required placeholder="Amount (₹)" value="${amount}" oninput="Operations.calcEditExpenseBreakdown();">
      <button type="button" class="btn-remove-bank" title="Remove Bank Payment" onclick="Operations.removeEditExpenseBankRow('${rowId}')">✕</button>
    `;
    container.appendChild(row);
    validateEditExpenseBanks();
    calcEditExpenseBreakdown();
  }

  function removeEditExpenseBankRow(rowId) {
    const row = document.getElementById(rowId);
    if (row) row.remove();
    validateEditExpenseBanks();
    calcEditExpenseBreakdown();
  }

  function validateEditExpenseBanks() {
    const selects = document.querySelectorAll('#edit-exp-bank-payments-container .edit-exp-bank-select');
    const warning = document.getElementById('edit-exp-bank-duplicate-warning');
    const selected = [];
    let hasDuplicate = false;

    selects.forEach(s => {
      const val = s.value;
      if (val) {
        if (selected.includes(val)) {
          hasDuplicate = true;
          s.style.borderColor = 'var(--color-error)';
        } else {
          selected.push(val);
          s.style.borderColor = '';
        }
      } else {
        s.style.borderColor = '';
      }
    });

    if (warning) warning.style.display = hasDuplicate ? 'block' : 'none';
    return !hasDuplicate;
  }

  function calcEditExpenseBreakdown() {
    const totalAmount = parseFloat(document.getElementById('edit-exp-amount')?.value) || 0;
    const method = document.querySelector('input[name="edit-exp-payment-method"]:checked')?.value || 'cash';
    const cashInput = document.getElementById('edit-exp-form-cash');
    const warnEl = document.getElementById('edit-exp-calc-warning');
    const submitBtn = document.getElementById('btn-submit-edit-exp');

    let cashPaid = 0;
    let bankPaid = 0;

    if (method === 'cash') {
      cashPaid = totalAmount;
      if (cashInput && document.activeElement !== cashInput) {
        cashInput.value = totalAmount > 0 ? totalAmount.toFixed(2) : '0.00';
      }
    } else if (method === 'bank') {
      cashPaid = 0;
      document.querySelectorAll('#edit-exp-bank-payments-container .edit-exp-bank-amount').forEach(inp => {
        bankPaid += parseFloat(inp.value) || 0;
      });
    } else if (method === 'cash_bank') {
      cashPaid = parseFloat(cashInput?.value) || 0;
      document.querySelectorAll('#edit-exp-bank-payments-container .edit-exp-bank-amount').forEach(inp => {
        bankPaid += parseFloat(inp.value) || 0;
      });
    }

    const totalPaid = Math.round((cashPaid + bankPaid) * 100) / 100;
    const diff = Math.round((totalAmount - totalPaid) * 100) / 100;

    const totalEl = document.getElementById('edit-exp-calc-total');
    const cashEl = document.getElementById('edit-exp-calc-cash');
    const bankEl = document.getElementById('edit-exp-calc-bank');
    const paidEl = document.getElementById('edit-exp-calc-paid');
    const remEl = document.getElementById('edit-exp-calc-remaining');
    const cashRow = document.getElementById('edit-exp-calc-cash-row');
    const bankRow = document.getElementById('edit-exp-calc-bank-row');

    if (totalEl) totalEl.textContent = formatCurrency(totalAmount);
    if (cashEl) cashEl.textContent = formatCurrency(cashPaid);
    if (bankEl) bankEl.textContent = formatCurrency(bankPaid);
    if (paidEl) paidEl.textContent = formatCurrency(totalPaid);
    if (remEl) remEl.textContent = formatCurrency(diff);

    if (cashRow) cashRow.style.display = (method === 'bank') ? 'none' : 'flex';
    if (bankRow) bankRow.style.display = (method === 'cash') ? 'none' : 'flex';

    const hasDuplicate = !validateEditExpenseBanks();
    const isMismatch = Math.abs(diff) > 0.005;
    const hasZeroTotal = totalAmount <= 0;

    let hasEmptyBank = false;
    if (method === 'bank' || method === 'cash_bank') {
      const bankRows = document.querySelectorAll('#edit-exp-bank-payments-container .bank-payment-row');
      if (bankRows.length === 0) {
        hasEmptyBank = true;
      } else {
        bankRows.forEach(r => {
          const sel = r.querySelector('.edit-exp-bank-select')?.value;
          const amt = parseFloat(r.querySelector('.edit-exp-bank-amount')?.value) || 0;
          if (!sel || amt <= 0) hasEmptyBank = true;
        });
      }
    }

    if (warnEl) {
      if (hasZeroTotal) {
        warnEl.textContent = '⚠️ Please enter an expense amount.';
        warnEl.style.display = 'block';
      } else if (hasDuplicate) {
        warnEl.textContent = '⚠️ Each bank account can only be selected once per expense.';
        warnEl.style.display = 'block';
      } else if (hasEmptyBank) {
        warnEl.textContent = '⚠️ Please select a bank account and enter a valid amount for all bank rows.';
        warnEl.style.display = 'block';
      } else if (isMismatch) {
        warnEl.textContent = `⚠️ Payment breakdown must equal the expense amount (${formatCurrency(totalAmount)}).`;
        warnEl.style.display = 'block';
      } else {
        warnEl.style.display = 'none';
      }
    }

    if (remEl) {
      remEl.style.color = isMismatch ? 'var(--color-error)' : 'var(--color-success)';
    }

    if (submitBtn) {
      submitBtn.disabled = hasZeroTotal || hasDuplicate || hasEmptyBank || isMismatch;
    }
  }

  // Expense: Create
  async function submitAddExpense(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-expense');
    setButtonLoading(btn, true);

    const amount = parseFloat(document.getElementById('exp-form-amount').value) || 0;
    const category = document.getElementById('exp-form-category')?.value || 'general';
    const employeeID = (category === 'employee_advance') ? (document.getElementById('exp-form-emp')?.value || null) : null;
    const refID = document.getElementById('exp-form-ref')?.value?.trim() || null;
    const date = document.getElementById('exp-form-date')?.value || null;
    const method = document.querySelector('input[name="exp-payment-method"]:checked')?.value || 'cash';

    if (amount <= 0) {
      setButtonLoading(btn, false);
      showModalError('modal-add-expense', 'Please enter a valid expense amount greater than zero.');
      return;
    }

    if (category === 'employee_advance' && !employeeID) {
      setButtonLoading(btn, false);
      showModalError('modal-add-expense', 'Please select a staff member for the advance payment.');
      return;
    }

    let cashAmount = 0;
    const bankPayments = [];

    if (method === 'cash') {
      cashAmount = amount;
    } else if (method === 'bank') {
      cashAmount = 0;
      const bankRows = document.querySelectorAll('#exp-bank-payments-container .bank-payment-row');
      if (bankRows.length === 0) {
        setButtonLoading(btn, false);
        showModalError('modal-add-expense', 'Please add at least one bank account payment.');
        return;
      }
      for (const r of bankRows) {
        const bankId = r.querySelector('.exp-bank-select')?.value;
        const bAmt = parseFloat(r.querySelector('.exp-bank-amount')?.value) || 0;
        if (!bankId || bAmt <= 0) {
          setButtonLoading(btn, false);
          showModalError('modal-add-expense', 'Please ensure all bank payment entries have an account and a positive amount.');
          return;
        }
        bankPayments.push({ bank_account_id: bankId, amount: bAmt });
      }
    } else if (method === 'cash_bank') {
      cashAmount = parseFloat(document.getElementById('exp-form-cash')?.value) || 0;
      if (cashAmount <= 0) {
        setButtonLoading(btn, false);
        showModalError('modal-add-expense', 'Cash amount must be greater than zero for Cash + Bank payment method.');
        return;
      }
      const bankRows = document.querySelectorAll('#exp-bank-payments-container .bank-payment-row');
      if (bankRows.length === 0) {
        setButtonLoading(btn, false);
        showModalError('modal-add-expense', 'Please add at least one bank account payment.');
        return;
      }
      for (const r of bankRows) {
        const bankId = r.querySelector('.exp-bank-select')?.value;
        const bAmt = parseFloat(r.querySelector('.exp-bank-amount')?.value) || 0;
        if (!bankId || bAmt <= 0) {
          setButtonLoading(btn, false);
          showModalError('modal-add-expense', 'Please ensure all bank payment entries have an account and a positive amount.');
          return;
        }
        bankPayments.push({ bank_account_id: bankId, amount: bAmt });
      }
    }

    const payload = {
      item: document.getElementById('exp-form-item').value.trim(),
      category: category,
      employee_id: employeeID,
      reference_id: refID,
      payment_date: date,
      amount: amount,
      total_amount: amount,
      payment_method: method,
      cash_amount: cashAmount,
      bank_payments: bankPayments
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
    loadSalaries();
    loadAttendance();
    loadDashboardMetrics();
    loadCashBank();
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
    if (document.getElementById('edit-exp-category')) {
      document.getElementById('edit-exp-category').value = e.category || 'general';
    }
    const totalAmount = parseFloat(e.total_amount) || 0;
    document.getElementById('edit-exp-amount').value = totalAmount > 0 ? totalAmount : '0';

    const method = e.payment_method || 'cash';
    const radio = document.querySelector(`input[name="edit-exp-payment-method"][value="${method}"]`);
    if (radio) radio.checked = true;

    const container = document.getElementById('edit-exp-bank-payments-container');
    if (container) container.innerHTML = '';

    handleEditExpensePaymentMethodChange(method);

    if (method === 'cash') {
      const cashInp = document.getElementById('edit-exp-form-cash');
      if (cashInp) cashInp.value = totalAmount.toFixed(2);
    } else if (method === 'bank') {
      const banks = e.payment_breakdown?.banks || [];
      if (container) container.innerHTML = '';
      if (banks.length > 0) {
        banks.forEach(b => {
          addEditExpenseBankRow(b.bank_account_id, parseFloat(b.amount) || '');
        });
      } else {
        addEditExpenseBankRow('', totalAmount > 0 ? totalAmount.toFixed(2) : '');
      }
    } else if (method === 'cash_bank') {
      const cashInp = document.getElementById('edit-exp-form-cash');
      const cashAmt = parseFloat(e.payment_breakdown?.cash_amount) || 0;
      if (cashInp) cashInp.value = cashAmt.toFixed(2);
      const banks = e.payment_breakdown?.banks || [];
      if (container) container.innerHTML = '';
      if (banks.length > 0) {
        banks.forEach(b => {
          addEditExpenseBankRow(b.bank_account_id, parseFloat(b.amount) || '');
        });
      } else {
        addEditExpenseBankRow('', '');
      }
    }

    calcEditExpenseBreakdown();
    openModal('modal-edit-expense');
  }

  async function submitEditExpense(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-edit-exp');
    const id = document.getElementById('edit-exp-id').value;
    setButtonLoading(btn, true);

    const amount = parseFloat(document.getElementById('edit-exp-amount').value) || 0;
    const category = document.getElementById('edit-exp-category')?.value || 'general';
    const method = document.querySelector('input[name="edit-exp-payment-method"]:checked')?.value || 'cash';

    if (amount <= 0) {
      setButtonLoading(btn, false);
      showModalError('modal-edit-expense', 'Please enter a valid expense amount greater than zero.');
      return;
    }

    let cashAmount = 0;
    const bankPayments = [];

    if (method === 'cash') {
      cashAmount = amount;
    } else if (method === 'bank') {
      cashAmount = 0;
      const bankRows = document.querySelectorAll('#edit-exp-bank-payments-container .bank-payment-row');
      if (bankRows.length === 0) {
        setButtonLoading(btn, false);
        showModalError('modal-edit-expense', 'Please add at least one bank account payment.');
        return;
      }
      for (const r of bankRows) {
        const bankId = r.querySelector('.edit-exp-bank-select')?.value;
        const bAmt = parseFloat(r.querySelector('.edit-exp-bank-amount')?.value) || 0;
        if (!bankId || bAmt <= 0) {
          setButtonLoading(btn, false);
          showModalError('modal-edit-expense', 'Please ensure all bank payment entries have an account and a positive amount.');
          return;
        }
        bankPayments.push({ bank_account_id: bankId, amount: bAmt });
      }
    } else if (method === 'cash_bank') {
      cashAmount = parseFloat(document.getElementById('edit-exp-form-cash')?.value) || 0;
      if (cashAmount <= 0) {
        setButtonLoading(btn, false);
        showModalError('modal-edit-expense', 'Cash amount must be greater than zero for Cash + Bank payment method.');
        return;
      }
      const bankRows = document.querySelectorAll('#edit-exp-bank-payments-container .bank-payment-row');
      if (bankRows.length === 0) {
        setButtonLoading(btn, false);
        showModalError('modal-edit-expense', 'Please add at least one bank account payment.');
        return;
      }
      for (const r of bankRows) {
        const bankId = r.querySelector('.edit-exp-bank-select')?.value;
        const bAmt = parseFloat(r.querySelector('.edit-exp-bank-amount')?.value) || 0;
        if (!bankId || bAmt <= 0) {
          setButtonLoading(btn, false);
          showModalError('modal-edit-expense', 'Please ensure all bank payment entries have an account and a positive amount.');
          return;
        }
        bankPayments.push({ bank_account_id: bankId, amount: bAmt });
      }
    }

    const payload = {
      item: document.getElementById('edit-exp-item').value.trim(),
      category: category,
      amount: amount,
      total_amount: amount,
      payment_method: method,
      cash_amount: cashAmount,
      bank_payments: bankPayments
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
    loadCashBank();
  }

  // Expense: View Details Modal
  async function viewExpense(id) {
    const res = await window.API.get(`/tenant/expenses/${id}`);
    if (!res.ok || !res.data) {
      showToast(res.error || 'Failed to load expense details', 'error');
      return;
    }
    const e = res.data;
    const isAdvance = e.category === 'employee_advance';
    const breakdown = e.payment_breakdown || {};
    const method = e.payment_method || 'cash';

    let methodBadge = '<span class="status-badge paid">Cash</span>';
    if (method === 'bank') {
      methodBadge = '<span class="status-badge cleared">Bank Only</span>';
    } else if (method === 'cash_bank') {
      methodBadge = '<span class="status-badge pending" style="background: rgba(147, 51, 234, 0.1); color: #7e22ce;">Cash + Bank</span>';
    }

    let paymentsHtml = '';
    if (method === 'cash') {
      paymentsHtml = `
        <li style="display:flex;justify-content:space-between;padding:0.35rem 0;border-bottom:1px solid var(--color-border-subtle);">
          <span>Cash in Drawer</span>
          <strong style="font-family:var(--font-mono);">${formatCurrency(e.total_amount)}</strong>
        </li>
      `;
    } else if (method === 'bank') {
      const banks = breakdown.banks || [];
      paymentsHtml = banks.map(b => `
        <li style="display:flex;justify-content:space-between;padding:0.35rem 0;border-bottom:1px solid var(--color-border-subtle);">
          <span>${escapeHTML(b.bank_name || 'Bank Account')}</span>
          <strong style="font-family:var(--font-mono);">${formatCurrency(b.amount)}</strong>
        </li>
      `).join('');
    } else if (method === 'cash_bank') {
      const banks = breakdown.banks || [];
      const cashAmt = breakdown.cash_amount || 0;
      paymentsHtml = `
        <li style="display:flex;justify-content:space-between;padding:0.35rem 0;border-bottom:1px solid var(--color-border-subtle);">
          <span>Cash in Drawer</span>
          <strong style="font-family:var(--font-mono);">${formatCurrency(cashAmt)}</strong>
        </li>
      ` + banks.map(b => `
        <li style="display:flex;justify-content:space-between;padding:0.35rem 0;border-bottom:1px solid var(--color-border-subtle);">
          <span>${escapeHTML(b.bank_name || 'Bank Account')}</span>
          <strong style="font-family:var(--font-mono);">${formatCurrency(b.amount)}</strong>
        </li>
      `).join('');
    }

    document.getElementById('view-details-title').textContent = `Expense Details - ${e.item || 'Expense'}`;
    document.getElementById('view-details-content').innerHTML = `
      <div style="display:flex;flex-direction:column;gap:0.75rem;font-size:0.875rem;">
        <div><strong>Item / Purpose:</strong> ${escapeHTML(e.item)}</div>
        <div><strong>Category:</strong> <span class="badge" style="font-size:0.75rem;background:rgba(99,102,241,0.1);color:#6366f1;">${escapeHTML((e.category || 'general').replace('_', ' ').toUpperCase())}</span></div>
        ${isAdvance ? `<div><strong>Staff Member:</strong> <span style="font-weight:600;color:var(--color-error);">${escapeHTML(e.employee_name || 'Employee')}</span></div>` : ''}
        <div><strong>Date Recorded:</strong> ${formatDateTime(e.created_at)}</div>
        <div><strong>Payment Method:</strong> ${methodBadge}</div>
        <div style="font-size:1.125rem;font-weight:700;color:var(--color-text-main);margin-top:0.25rem;">
          Total Amount: <span style="color:var(--color-error);">${formatCurrency(e.total_amount)}</span>
        </div>
        <div style="margin-top:0.5rem;border-top:1px solid var(--color-border);padding-top:0.5rem;">
          <strong style="display:block;margin-bottom:0.35rem;">Payment Breakdown:</strong>
          <ul style="margin:0;padding:0;list-style:none;">
            ${paymentsHtml}
          </ul>
        </div>
      </div>
    `;
    openModal('modal-view-details');
  }

  // Attendance: Mark
  async function submitMarkAttendance(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-attendance');
    setButtonLoading(btn, true);

    const ot = parseFloat(document.getElementById('att-form-ot').value) || 0;
    const payload = {
      employee_id: document.getElementById('att-form-emp').value,
      date: document.getElementById('att-form-date').value || getTodayString(),
      status: document.getElementById('att-form-status').value,
      ot: ot,
      ot_amount: ot,
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
    document.getElementById('edit-att-ot').value = (parseFloat(a.ot_amount) > 0 ? a.ot_amount : (a.ot || '0'));
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
      ot_amount: ot,
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

    const amount = parseFloat(document.getElementById('ot-form-amount').value) || 0;
    if (amount <= 0) {
      setButtonLoading(btn, false);
      showModalError('modal-record-ot', 'Overtime amount must be greater than zero.');
      return;
    }

    const payload = {
      employee_id: document.getElementById('ot-form-emp').value,
      date: document.getElementById('ot-form-date').value || getTodayString(),
      amount: amount,
      reference_id: document.getElementById('ot-form-ref')?.value?.trim() || '',
      note: document.getElementById('ot-form-note')?.value?.trim() || ''
    };

    const res = await window.API.post('/tenant/overtime', payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-record-ot', res.error);
      return;
    }

    closeModal('modal-record-ot');
    showToast('Overtime amount recorded successfully');
    loadAttendance();
    loadSalaries();
    loadDashboardMetrics();
  }

  // Advance: Record
  async function submitRecordAdvance(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-advance');
    setButtonLoading(btn, true);

    const amount = parseFloat(document.getElementById('adv-form-amount').value) || 0;
    if (amount <= 0) {
      setButtonLoading(btn, false);
      showModalError('modal-record-advance', 'Advance amount must be greater than zero.');
      return;
    }

    const payload = {
      employee_id: document.getElementById('adv-form-emp').value,
      date: document.getElementById('adv-form-date').value || getTodayString(),
      amount: amount,
      payment_method: document.getElementById('adv-form-method').value,
      reference_id: document.getElementById('adv-form-ref')?.value?.trim() || '',
      note: document.getElementById('adv-form-note')?.value?.trim() || ''
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
    loadSalaries();
    loadDashboardMetrics();
  }

  // Salary: Pay
  function openPaySalary(empId, empName, balance) {
    document.getElementById('pay-salary-empid').value = empId;
    document.getElementById('pay-salary-empname').textContent = empName;
    document.getElementById('pay-salary-pending').textContent = formatCurrency(balance);
    document.getElementById('pay-salary-amount').value = Math.max(0, parseFloat(balance) || 0);
    const dateInput = document.getElementById('pay-salary-date');
    if (dateInput) dateInput.value = getTodayString();
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
      reference_id: document.getElementById('pay-salary-ref')?.value?.trim() || '',
      payment_date: document.getElementById('pay-salary-date')?.value || null,
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

  // Salary Statement / Ledger
  let currentSalaryStatement = null;

  async function openSalaryStatement(empId, empName) {
    const res = await window.API.get(`/tenant/salaries/${empId}/statement`);
    if (!res.ok || !res.data) {
      showToast(res.error || 'Failed to load employee salary statement', 'error');
      return;
    }

    const s = res.data;
    currentSalaryStatement = s;

    document.getElementById('stmt-salary-empname').textContent = `${s.employee_name || empName || 'Staff Member'} — Salary Ledger`;
    document.getElementById('stmt-salary-base').textContent = formatCurrency(s.base_salary);
    document.getElementById('stmt-salary-ot').textContent = formatCurrency(s.total_overtime);
    document.getElementById('stmt-salary-gross').textContent = formatCurrency(s.gross_salary);
    document.getElementById('stmt-salary-advances').textContent = formatCurrency(s.total_advances);
    document.getElementById('stmt-salary-paid').textContent = formatCurrency(s.total_paid);
    document.getElementById('stmt-salary-net').textContent = formatCurrency(s.net_payable);

    // Overtime entries
    const otTbody = document.getElementById('stmt-salary-ot-tbody');
    const otCount = document.getElementById('stmt-salary-ot-count');
    if (otTbody) {
      if (Array.isArray(s.overtimes) && s.overtimes.length > 0) {
        if (otCount) otCount.textContent = `${s.overtimes.length} entries`;
        otTbody.innerHTML = s.overtimes.map(o => `
          <tr>
            <td>${formatDate(o.overtime_date || o.created_at)}</td>
            <td style="color:var(--color-success);font-weight:600;">+${formatCurrency(o.amount)}</td>
            <td>${escapeHTML(o.reference_id || '—')}</td>
            <td>${escapeHTML(o.notes || '—')}</td>
          </tr>
        `).join('');
      } else {
        if (otCount) otCount.textContent = '0 entries';
        otTbody.innerHTML = `<tr><td colspan="4" style="text-align:center;color:var(--color-text-muted);padding:1rem;">No overtime records.</td></tr>`;
      }
    }

    // Advances entries
    const advTbody = document.getElementById('stmt-salary-adv-tbody');
    const advCount = document.getElementById('stmt-salary-adv-count');
    if (advTbody) {
      if (Array.isArray(s.advances) && s.advances.length > 0) {
        if (advCount) advCount.textContent = `${s.advances.length} entries`;
        advTbody.innerHTML = s.advances.map(a => `
          <tr>
            <td>${formatDate(a.advance_date || a.created_at)}</td>
            <td style="color:var(--color-error);font-weight:600;">-${formatCurrency(a.amount)}</td>
            <td><span class="badge">${escapeHTML((a.payment_method || 'cash').toUpperCase())}</span></td>
            <td>${escapeHTML(a.reference_id || '—')}</td>
            <td>${escapeHTML(a.notes || '—')}</td>
          </tr>
        `).join('');
      } else {
        if (advCount) advCount.textContent = '0 entries';
        advTbody.innerHTML = `<tr><td colspan="5" style="text-align:center;color:var(--color-text-muted);padding:1rem;">No advance records.</td></tr>`;
      }
    }

    // Payout entries
    const payTbody = document.getElementById('stmt-salary-pay-tbody');
    const payCount = document.getElementById('stmt-salary-pay-count');
    if (payTbody) {
      if (Array.isArray(s.payments) && s.payments.length > 0) {
        if (payCount) payCount.textContent = `${s.payments.length} entries`;
        payTbody.innerHTML = s.payments.map(p => `
          <tr>
            <td>${formatDate(p.payment_date || p.created_at)}</td>
            <td style="color:#3b82f6;font-weight:600;">-${formatCurrency(p.amount)}</td>
            <td><span class="badge">${escapeHTML((p.payment_method || 'cash').toUpperCase())}</span></td>
            <td>${escapeHTML(p.reference_id || p.id ? p.id.substring(0, 8) : '—')}</td>
            <td>${escapeHTML(p.note || '—')}</td>
          </tr>
        `).join('');
      } else {
        if (payCount) payCount.textContent = '0 entries';
        payTbody.innerHTML = `<tr><td colspan="5" style="text-align:center;color:var(--color-text-muted);padding:1rem;">No payment records.</td></tr>`;
      }
    }

    openModal('modal-salary-statement');
  }

  function openPaySalaryFromStatement() {
    if (!currentSalaryStatement) return;
    const s = currentSalaryStatement;
    closeModal('modal-salary-statement');
    openPaySalary(s.employee_id, s.employee_name, s.net_payable);
  }

  // Payment History Inspector
  async function showPurchasePaymentHistory(purchaseId, itemName, totalAmount) {
    document.getElementById('pmt-hist-title').textContent = 'Purchase Payment History';
    document.getElementById('pmt-hist-subtitle').textContent = `Procurement Item: ${itemName}`;
    document.getElementById('pmt-hist-total').textContent = formatCurrency(totalAmount);

    const tbody = document.getElementById('pmt-hist-tbody');
    tbody.innerHTML = `<tr><td colspan="6" style="text-align:center;padding:1.5rem;color:var(--color-text-muted);">Loading payment records...</td></tr>`;
    openModal('modal-payment-history');

    const res = await window.API.get(`/tenant/purchases/${purchaseId}/payments`);
    if (!res.ok || !Array.isArray(res.data)) {
      tbody.innerHTML = `<tr><td colspan="6" style="text-align:center;color:var(--color-error);padding:1.5rem;">Failed to load payment history.</td></tr>`;
      return;
    }

    const payments = res.data;
    let paidSum = 0;
    payments.forEach(p => { paidSum += parseFloat(p.amount) || 0; });
    const pending = Math.max(0, totalAmount - paidSum);

    document.getElementById('pmt-hist-paid').textContent = formatCurrency(paidSum);
    document.getElementById('pmt-hist-pending').textContent = formatCurrency(pending);

    let statusHtml = '<span class="status-badge danger">UNPAID</span>';
    if (pending <= 0 && totalAmount > 0) {
      statusHtml = '<span class="status-badge paid">PAID</span>';
    } else if (paidSum > 0) {
      statusHtml = '<span class="status-badge pending">PARTIALLY PAID</span>';
    }
    document.getElementById('pmt-hist-status').innerHTML = statusHtml;

    if (payments.length === 0) {
      tbody.innerHTML = `<tr><td colspan="6" style="text-align:center;padding:1.5rem;color:var(--color-text-muted);">No payment transactions recorded yet.</td></tr>`;
      return;
    }

    tbody.innerHTML = payments.map(p => `
      <tr>
        <td><strong>${escapeHTML(p.reference_id || (p.id ? p.id.substring(0, 8) : 'PAY'))}</strong></td>
        <td>${formatDate(p.payment_date || p.created_at)}</td>
        <td style="font-weight:700;color:var(--color-success);">${formatCurrency(p.amount)}</td>
        <td><span class="badge">${escapeHTML((p.payment_method || 'cash').toUpperCase())}</span></td>
        <td><span class="status-badge ${p.status === 'completed' || !p.status ? 'paid' : 'pending'}">${escapeHTML((p.status || 'completed').toUpperCase())}</span></td>
        <td>${escapeHTML(p.notes || '—')}</td>
      </tr>
    `).join('');
  }

  async function showLineSalePaymentHistory(saleId, customerName, totalAmount, totalCashIn, balance) {
    document.getElementById('pmt-hist-title').textContent = 'Line Sale Payment History';
    document.getElementById('pmt-hist-subtitle').textContent = `Customer: ${customerName}`;
    document.getElementById('pmt-hist-total').textContent = formatCurrency(totalAmount);

    const tbody = document.getElementById('pmt-hist-tbody');
    tbody.innerHTML = `<tr><td colspan="6" style="text-align:center;padding:1.5rem;color:var(--color-text-muted);">Loading payment records...</td></tr>`;
    openModal('modal-payment-history');

    const res = await window.API.get(`/tenant/line-sales/${saleId}/payments`);
    if (!res.ok || !Array.isArray(res.data)) {
      tbody.innerHTML = `<tr><td colspan="6" style="text-align:center;color:var(--color-error);padding:1.5rem;">Failed to load payment history.</td></tr>`;
      return;
    }

    const payments = res.data;
    let paidSum = 0;
    payments.forEach(p => { paidSum += parseFloat(p.amount) || 0; });
    const pending = Math.max(0, (parseFloat(balance) || (totalAmount - paidSum)));

    document.getElementById('pmt-hist-paid').textContent = formatCurrency(paidSum > 0 ? paidSum : totalCashIn);
    document.getElementById('pmt-hist-pending').textContent = formatCurrency(pending);

    let statusHtml = '<span class="status-badge danger">UNPAID</span>';
    if (pending <= 0) {
      statusHtml = '<span class="status-badge paid">PAID</span>';
    } else if (paidSum > 0 || totalCashIn > 0) {
      statusHtml = '<span class="status-badge pending">PARTIALLY PAID</span>';
    }
    document.getElementById('pmt-hist-status').innerHTML = statusHtml;

    if (payments.length === 0) {
      tbody.innerHTML = `<tr><td colspan="6" style="text-align:center;padding:1.5rem;color:var(--color-text-muted);">No payment records found. (Initial cash in: ${formatCurrency(totalCashIn)})</td></tr>`;
      return;
    }

    tbody.innerHTML = payments.map(p => `
      <tr>
        <td><strong>${escapeHTML(p.reference_id || (p.id ? p.id.substring(0, 8) : 'PAY'))}</strong></td>
        <td>${formatDate(p.payment_date || p.created_at)}</td>
        <td style="font-weight:700;color:var(--color-success);">${formatCurrency(p.amount)}</td>
        <td><span class="badge">${escapeHTML((p.payment_method || 'cash').toUpperCase())}</span></td>
        <td><span class="status-badge ${p.status === 'completed' || !p.status ? 'paid' : 'pending'}">${escapeHTML((p.status || 'completed').toUpperCase())}</span></td>
        <td>${escapeHTML(p.notes || '—')}</td>
      </tr>
    `).join('');
  }

  async function showCounterSalePaymentHistory(saleId, itemName, totalAmount) {
    document.getElementById('pmt-hist-title').textContent = 'Counter Sale Payment History';
    document.getElementById('pmt-hist-subtitle').textContent = `Item: ${itemName}`;
    document.getElementById('pmt-hist-total').textContent = formatCurrency(totalAmount);

    const tbody = document.getElementById('pmt-hist-tbody');
    tbody.innerHTML = `<tr><td colspan="6" style="text-align:center;padding:1.5rem;color:var(--color-text-muted);">Loading payment records...</td></tr>`;
    openModal('modal-payment-history');

    const res = await window.API.get(`/tenant/counter-sales/${saleId}/payments`);
    if (!res.ok || !Array.isArray(res.data)) {
      tbody.innerHTML = `<tr><td colspan="6" style="text-align:center;color:var(--color-error);padding:1.5rem;">Failed to load payment history.</td></tr>`;
      return;
    }

    const payments = res.data;
    let paidSum = 0;
    payments.forEach(p => { paidSum += parseFloat(p.amount) || 0; });
    const pending = Math.max(0, totalAmount - paidSum);

    document.getElementById('pmt-hist-paid').textContent = formatCurrency(paidSum);
    document.getElementById('pmt-hist-pending').textContent = formatCurrency(pending);

    let statusHtml = '<span class="status-badge danger">UNPAID</span>';
    if (pending <= 0) {
      statusHtml = '<span class="status-badge paid">PAID</span>';
    } else if (paidSum > 0) {
      statusHtml = '<span class="status-badge pending">PARTIALLY PAID</span>';
    }
    document.getElementById('pmt-hist-status').innerHTML = statusHtml;

    if (payments.length === 0) {
      tbody.innerHTML = `<tr><td colspan="6" style="text-align:center;padding:1.5rem;color:var(--color-text-muted);">No payment records found.</td></tr>`;
      return;
    }

    tbody.innerHTML = payments.map(p => `
      <tr>
        <td><strong>${escapeHTML(p.reference_id || (p.id ? p.id.substring(0, 8) : 'PAY'))}</strong></td>
        <td>${formatDate(p.payment_date || p.created_at)}</td>
        <td style="font-weight:700;color:var(--color-success);">${formatCurrency(p.amount)}</td>
        <td><span class="badge">${escapeHTML((p.payment_method || 'cash').toUpperCase())}</span></td>
        <td><span class="status-badge ${p.status === 'completed' || !p.status ? 'paid' : 'pending'}">${escapeHTML((p.status || 'completed').toUpperCase())}</span></td>
        <td>${escapeHTML(p.notes || '—')}</td>
      </tr>
    `).join('');
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
    const recv = parseFloat(c.current_balance) || 0;
    const payable = parseFloat(c.outstanding_payable) || 0;
    const purchases = parseFloat(c.total_purchases) || 0;
    const paid = parseFloat(c.total_paid) || 0;

    document.getElementById('view-details-title').textContent = `Customer / Supplier Profile - ${c.customer_name}`;
    document.getElementById('view-details-content').innerHTML = `
      <div style="display:flex;flex-direction:column;gap:0.875rem;font-size:0.875rem;">
        <div><strong>Business / Contact:</strong> ${escapeHTML(c.customer_name)}</div>
        <div><strong>Phone:</strong> ${escapeHTML(c.phone || '—')}</div>
        <div><strong>Account Status:</strong> <span class="status-badge ${c.status === 'active' ? 'paid' : 'pending'}">${escapeHTML(c.status)}</span></div>
        
        <div style="background:var(--color-surface-hover);border:1px solid var(--color-border);border-radius:var(--radius-sm);padding:0.75rem 1rem;margin-top:0.25rem;">
          <div style="font-size:0.75rem;font-weight:700;color:var(--color-text-muted);text-transform:uppercase;margin-bottom:0.25rem;">Customer Sales Balance</div>
          <div style="font-size:1.125rem;font-weight:700;color:${recv > 0 ? 'var(--color-warning)' : 'inherit'};">
            ${formatCurrency(recv)} <span style="font-size:0.75rem;font-weight:400;color:var(--color-text-muted);">(Receivable Due)</span>
          </div>
        </div>

        <div style="background:rgba(239, 68, 68, 0.05);border:1px solid rgba(239, 68, 68, 0.2);border-radius:var(--radius-sm);padding:0.75rem 1rem;">
          <div style="font-size:0.75rem;font-weight:700;color:var(--color-error);text-transform:uppercase;margin-bottom:0.25rem;">Supplier Procurement Account</div>
          <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.25rem;">
            <span style="color:var(--color-text-muted);">Total Purchases:</span>
            <strong>${formatCurrency(purchases)}</strong>
          </div>
          <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.35rem;">
            <span style="color:var(--color-text-muted);">Total Payments Made:</span>
            <strong style="color:var(--color-success);">${formatCurrency(paid)}</strong>
          </div>
          <div style="display:flex;justify-content:space-between;align-items:center;padding-top:0.35rem;border-top:1px dashed rgba(239,68,68,0.3);font-size:1.0625rem;font-weight:700;color:var(--color-error);">
            <span>Outstanding Payable:</span>
            <span>${formatCurrency(payable)}</span>
          </div>
        </div>

        <div style="display:flex;gap:0.5rem;margin-top:0.5rem;">
          <button type="button" class="btn btn-secondary btn-sm" onclick="Operations.closeModal('modal-view-details'); Operations.openCustomerStatement('${c.id}');" style="flex:1;">
            View Statement →
          </button>
          ${payable > 0 ? `
            <button type="button" class="btn btn-primary btn-sm" onclick="Operations.closeModal('modal-view-details'); Operations.openMakePaymentModal({customerId: '${c.id}', customerName: '${escapeHTML(c.customer_name)}', pending: ${payable}});" style="flex:1;">
              Make Payment
            </button>
          ` : ''}
        </div>
      </div>
    `;
    openModal('modal-view-details');
  }

  // ==========================================
  // Customer Statement & Supplier Payment Logic
  // ==========================================

  async function openCustomerStatement(customerId) {
    activeStatementCustomerID = customerId;
    const titleEl = document.getElementById('stmt-customer-name');
    const subEl = document.getElementById('stmt-customer-sub');
    const totalPurchEl = document.getElementById('stmt-total-purchases');
    const totalPaidEl = document.getElementById('stmt-total-paid');
    const outstandingEl = document.getElementById('stmt-outstanding-payable');
    const tbody = document.getElementById('stmt-table-body');
    const btnPay = document.getElementById('btn-stmt-make-payment');

    if (titleEl) titleEl.textContent = 'Customer Account Statement';
    if (tbody) tbody.innerHTML = `<tr><td colspan="7" style="text-align:center;padding:2rem;color:var(--color-text-muted);">Loading account statement...</td></tr>`;

    openModal('modal-customer-statement');

    const res = await window.API.get(`/tenant/customers/${customerId}/statement`);
    if (!res.ok || !res.data) {
      showToast(res.error || 'Failed to fetch customer statement', 'error');
      if (tbody) tbody.innerHTML = `<tr><td colspan="7" style="text-align:center;color:var(--color-error);padding:2rem;">Failed to load statement: ${escapeHTML(res.error || 'Unknown error')}</td></tr>`;
      return;
    }

    const stmt = res.data;
    if (titleEl) titleEl.textContent = `${stmt.customer_name} - Statement`;
    if (subEl) subEl.textContent = `Phone: ${stmt.phone || '—'} | Ledger Statement`;
    if (totalPurchEl) totalPurchEl.textContent = formatCurrency(stmt.total_purchases);
    if (totalPaidEl) totalPaidEl.textContent = formatCurrency(stmt.total_paid);
    if (outstandingEl) outstandingEl.textContent = formatCurrency(stmt.outstanding_payable);

    const pending = parseFloat(stmt.outstanding_payable) || 0;
    if (btnPay) {
      btnPay.style.display = pending > 0 ? 'inline-block' : 'none';
    }

    if (!Array.isArray(stmt.entries) || stmt.entries.length === 0) {
      tbody.innerHTML = `
        <tr>
          <td colspan="7" style="text-align:center;padding:2.5rem;color:var(--color-text-muted);">
            No procurement or settlement transactions recorded yet.
          </td>
        </tr>
      `;
      return;
    }

    tbody.innerHTML = stmt.entries.map(entry => {
      const isPurchase = entry.entry_type === 'purchase';
      const typeBadge = isPurchase
        ? '<span class="status-badge cleared" style="font-size:0.7rem;">PURCHASE</span>'
        : '<span class="status-badge paid" style="font-size:0.7rem;">SETTLEMENT</span>';

      const purchaseAmt = parseFloat(entry.purchase_amount) || 0;
      const paidAmt = parseFloat(entry.paid_amount) || 0;
      const balAmt = parseFloat(entry.balance) || 0;

      return `
        <tr>
          <td style="white-space:nowrap;">${entry.formatted_date || formatDate(entry.date)}</td>
          <td><strong>${escapeHTML(entry.description)}</strong></td>
          <td>${typeBadge}</td>
          <td style="font-weight:600;font-family:var(--font-mono);">${purchaseAmt > 0 ? formatCurrency(purchaseAmt) : '—'}</td>
          <td style="font-weight:600;color:var(--color-success);font-family:var(--font-mono);">${paidAmt > 0 ? formatCurrency(paidAmt) : '—'}</td>
          <td style="font-weight:700;color:${balAmt > 0 ? 'var(--color-error)' : 'var(--color-success)'};font-family:var(--font-mono);">${formatCurrency(balAmt)}</td>
          <td style="font-size:0.75rem;color:var(--color-text-muted);">
            ${entry.payment_method ? escapeHTML(entry.payment_method.toUpperCase()) : '—'}
            ${entry.bank_name ? `(${escapeHTML(entry.bank_name)})` : ''}
          </td>
        </tr>
      `;
    }).join('');
  }

  function openMakePaymentFromStatement() {
    if (!activeStatementCustomerID) return;
    const outstandingText = document.getElementById('stmt-outstanding-payable')?.textContent || '0';
    const pendingVal = parseFloat(outstandingText.replace(/[^0-9.-]+/g, '')) || 0;
    const title = document.getElementById('stmt-customer-name')?.textContent || 'Customer';
    const cleanName = title.replace(' - Statement', '');

    openMakePaymentModal({
      customerId: activeStatementCustomerID,
      customerName: cleanName,
      pending: pendingVal
    });
  }

  function openMakePaymentModal({customerId = null, purchaseId = null, customerName = '', pending = 0, item = ''}) {
    supplierPaymentTarget = {
      customerId: customerId,
      purchaseId: purchaseId,
      pending: parseFloat(pending) || 0,
      customerName: customerName || 'Vendor',
      item: item || ''
    };

    const targetNameEl = document.getElementById('supp-pay-target-name');
    const purchaseInfoEl = document.getElementById('supp-pay-purchase-info');
    const itemNameEl = document.getElementById('supp-pay-item-name');
    const outstandingEl = document.getElementById('supp-pay-current-outstanding');
    const custIdInput = document.getElementById('supp-pay-customer-id');
    const purchIdInput = document.getElementById('supp-pay-purchase-id');
    const amountInput = document.getElementById('supp-pay-amount');
    const methodSelect = document.getElementById('supp-pay-method');
    const noteInput = document.getElementById('supp-pay-note');
    const bankGroup = document.getElementById('supp-pay-bank-group');
    const bankSelect = document.getElementById('supp-pay-bank');

    if (targetNameEl) targetNameEl.textContent = customerName || 'Vendor';
    if (custIdInput) custIdInput.value = customerId || '';
    if (purchIdInput) purchIdInput.value = purchaseId || '';

    if (purchaseInfoEl && itemNameEl) {
      if (purchaseId && item) {
        purchaseInfoEl.style.display = 'flex';
        itemNameEl.textContent = item;
      } else {
        purchaseInfoEl.style.display = 'none';
      }
    }

    if (outstandingEl) {
      outstandingEl.textContent = formatCurrency(supplierPaymentTarget.pending);
    }

    if (amountInput) {
      amountInput.value = supplierPaymentTarget.pending > 0 ? supplierPaymentTarget.pending.toFixed(2) : '';
      amountInput.max = supplierPaymentTarget.pending > 0 ? supplierPaymentTarget.pending.toString() : '';
    }

    if (methodSelect) methodSelect.value = 'cash';
    if (bankGroup) bankGroup.style.display = 'none';
    if (noteInput) noteInput.value = '';

    if (bankSelect) {
      bankSelect.innerHTML = renderBankSelectOptions();
    }

    calcSupplierPaymentRemaining();
    openModal('modal-make-supplier-payment');
  }

  function handleSupplierPayMethodChange() {
    const method = document.getElementById('supp-pay-method')?.value || 'cash';
    const bankGroup = document.getElementById('supp-pay-bank-group');
    if (bankGroup) {
      bankGroup.style.display = method === 'bank' ? 'block' : 'none';
    }
  }

  function calcSupplierPaymentRemaining() {
    const cur = supplierPaymentTarget.pending;
    const paying = parseFloat(document.getElementById('supp-pay-amount')?.value) || 0;
    const remaining = Math.max(0, cur - paying);

    const elCur = document.getElementById('supp-pay-calc-current');
    const elPay = document.getElementById('supp-pay-calc-paying');
    const elRem = document.getElementById('supp-pay-calc-remaining');
    const warn = document.getElementById('supp-pay-warning');

    if (elCur) elCur.textContent = formatCurrency(cur);
    if (elPay) elPay.textContent = formatCurrency(paying);
    if (elRem) elRem.textContent = formatCurrency(remaining);

    if (warn) {
      warn.style.display = paying > cur && cur > 0 ? 'block' : 'none';
    }
  }

  async function submitSupplierPayment(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-supplier-pay');
    const paying = parseFloat(document.getElementById('supp-pay-amount')?.value) || 0;
    const cur = supplierPaymentTarget.pending;

    if (paying <= 0) {
      showToast('Payment amount must be greater than zero', 'error');
      return;
    }
    if (cur > 0 && paying > cur) {
      showToast('Payment amount cannot exceed current outstanding payable balance', 'error');
      return;
    }

    const method = document.getElementById('supp-pay-method')?.value || 'cash';
    const bankID = document.getElementById('supp-pay-bank')?.value || null;
    const note = document.getElementById('supp-pay-note')?.value?.trim() || '';

    if (method === 'bank' && !bankID) {
      showToast('Please select a bank account for bank payment', 'error');
      return;
    }

    setButtonLoading(btn, true);

    const custId = supplierPaymentTarget.customerId;
    const purchId = supplierPaymentTarget.purchaseId;

    let url = '';
    const payload = {
      amount: paying,
      payment_method: method,
      bank_id: method === 'bank' ? bankID : null,
      note: note
    };

    if (purchId) {
      url = `/tenant/purchases/${purchId}/payments`;
      payload.purchase_id = purchId;
      if (custId) payload.customer_id = custId;
    } else if (custId) {
      url = `/tenant/customers/${custId}/payments`;
      payload.customer_id = custId;
    } else {
      showToast('Missing customer or purchase context for payment', 'error');
      setButtonLoading(btn, false);
      return;
    }

    const res = await window.API.post(url, payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showToast(res.error || 'Failed to record supplier payment', 'error');
      return;
    }

    closeModal('modal-make-supplier-payment');
    showToast(`Payment of ${formatCurrency(paying)} recorded successfully!`);

    // Reload relevant views
    loadPurchases();
    loadCustomers();
    loadCashBank();
    loadReceivablesDues();
    loadDashboardMetrics();

    // If statement modal is open for this customer, refresh it immediately
    const stmtModal = document.getElementById('modal-customer-statement');
    if (stmtModal && stmtModal.style.display !== 'none' && activeStatementCustomerID) {
      openCustomerStatement(activeStatementCustomerID);
    }
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
      case 'settings':
        loadTenantSettings();
        break;
      case 'subscription':
        loadTenantSubscription();
        break;
      case 'transactions':
        loadTenantTransactions();
        break;
      case 'profile':
        switchTab('settings');
        break;
    }
  }

  function openMobileSidebar() {
    const sidebar = document.getElementById('app-sidebar');
    const backdrop = document.getElementById('sidebar-backdrop');
    if (sidebar) {
      sidebar.classList.add('open', 'active');
    }
    if (backdrop) {
      backdrop.classList.add('active', 'open');
    }
    document.body.style.overflow = 'hidden';
  }

  function closeMobileSidebar() {
    const sidebar = document.getElementById('app-sidebar');
    const backdrop = document.getElementById('sidebar-backdrop');
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
    const sidebar = document.getElementById('app-sidebar');
    if (sidebar && (sidebar.classList.contains('open') || sidebar.classList.contains('active'))) {
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

    // Wire Mobile Sidebar Toggle, Close Button & Backdrop
    const mobileToggle = document.getElementById('mobile-sidebar-toggle');
    const mobileClose = document.getElementById('mobile-sidebar-close');
    const backdrop = document.getElementById('sidebar-backdrop');
    if (mobileToggle) {
      mobileToggle.addEventListener('click', toggleMobileSidebar);
    }
    if (mobileClose) {
      mobileClose.addEventListener('click', closeMobileSidebar);
    }
    if (backdrop) {
      backdrop.addEventListener('click', closeMobileSidebar);
    }

    // Keyboard & Resize listeners
    document.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') {
        closeMobileSidebar();
      }
    });

    window.addEventListener('resize', () => {
      if (window.innerWidth > 992) {
        closeMobileSidebar();
      }
    });

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
    viewExpense,
    viewLineSale,
    viewCounterSale,
    viewEmployee,
    viewCustomer,
    calcLineSaleDue,
    handleCounterSaleMethodChange,
    calcCounterSaleDue,
    filterCustomerDropdown,
    handlePurchaseCustomerChange,
    calcPurchaseBalance,
    addPurchaseBankRow,
    removePurchaseBankRow,
    validatePurchaseBanks,
    handleExpensePaymentMethodChange,
    addExpenseBankRow,
    removeExpenseBankRow,
    validateExpenseBanks,
    calcExpenseBreakdown,
    handleEditExpensePaymentMethodChange,
    addEditExpenseBankRow,
    removeEditExpenseBankRow,
    validateEditExpenseBanks,
    calcEditExpenseBreakdown,
    openCustomerStatement,
    openMakePaymentFromStatement,
    openMakePaymentModal,
    handleSupplierPayMethodChange,
    calcSupplierPaymentRemaining,
    submitSupplierPayment,
    handleExpenseCategoryChange,
    openSalaryStatement,
    openPaySalaryFromStatement,
    showPurchasePaymentHistory,
    showLineSalePaymentHistory,
    showCounterSalePaymentHistory,
    handleLineSaleCustomerChange,
    addLineSaleBankRow,
    removeLineSaleBankRow,
    validateLineSaleBanks,
    addCounterSaleBankRow,
    removeCounterSaleBankRow,
    validateCounterSaleBanks,
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
    loadDashboardMetrics,
    loadTenantSettings,
    submitTenantSettings,
    loadTenantSubscription,
    loadAvailablePlans,
    scrollToPlans,
    openPurchaseModal,
    submitPurchasePlan,
    loadTenantTransactions
  };
})();

window.Operations = Operations;
