/**
 * Vyavsa Operations Management Module
 * Client-side Controller for Tenant Admin and Tenant User Business Modules
 */

const Operations = (() => {
  let currentUser = null;
  let currentRole = 'user'; // 'admin' or 'user'
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
      const d = new Date(dStr);
      if (isNaN(d.getTime())) return dStr;
      return d.toLocaleDateString('en-IN', { day: '2-digit', month: 'short', year: 'numeric' });
    } catch (e) {
      return dStr;
    }
  }

  function getTodayString() {
    return new Date().toISOString().split('T')[0];
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
    toast.innerHTML = `<span>${type === 'error' ? '⚠️' : '✓'}</span><span>${message}</span>`;
    container.appendChild(toast);

    setTimeout(() => {
      toast.style.opacity = '0';
      toast.style.transition = 'opacity 0.3s ease';
      setTimeout(() => toast.remove(), 300);
    }, 4000);
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
      const form = modal.querySelector('form');
      if (form) form.insertBefore(errEl, form.firstChild);
    }
    errEl.textContent = msg;
    errEl.style.display = 'block';
  }

  // Preload dropdown dependencies
  async function loadDependencies() {
    if (currentRole === 'admin') {
      const [empRes, custRes, bankRes] = await Promise.all([
        window.API.get('/tenant/employees?page_size=100'),
        window.API.get('/tenant/customers?page_size=100'),
        window.API.get('/tenant/banks?page_size=100')
      ]);

      if (empRes.ok && Array.isArray(empRes.data)) cachedEmployees = empRes.data;
      if (custRes.ok && Array.isArray(custRes.data)) cachedCustomers = custRes.data;
      if (bankRes.ok && Array.isArray(bankRes.data)) cachedBanks = bankRes.data;
    } else {
      // Regular user may need employees for attendance/OT/advance
      const attRes = await window.API.get('/tenant/attendance?date=' + getTodayString());
      if (attRes.ok && Array.isArray(attRes.data)) {
        cachedEmployees = attRes.data.map(a => ({ id: a.employee_id, name: a.employee_name }));
      }
    }

    populateSelectDropdowns();
  }

  function populateSelectDropdowns() {
    // Populate employee selects
    document.querySelectorAll('.select-employee').forEach(sel => {
      const cur = sel.value;
      sel.innerHTML = '<option value="">Select Employee...</option>' +
        cachedEmployees.map(e => `<option value="${e.id}">${e.name || 'Unnamed Employee'}</option>`).join('');
      if (cur) sel.value = cur;
    });

    // Populate customer selects
    document.querySelectorAll('.select-customer').forEach(sel => {
      const cur = sel.value;
      sel.innerHTML = '<option value="">Select Customer...</option>' +
        cachedCustomers.map(c => `<option value="${c.id}">${c.customer_name || 'Customer'}</option>`).join('');
      if (cur) sel.value = cur;
    });

    // Populate bank selects
    document.querySelectorAll('.select-bank').forEach(sel => {
      const cur = sel.value;
      sel.innerHTML = '<option value="">Select Bank Account...</option>' +
        cachedBanks.map(b => `<option value="${b.id}">${b.bank_name} (${b.account_number || 'Main'})</option>`).join('');
      if (cur) sel.value = cur;
    });
  }

  // ==========================================
  // Section Loaders
  // ==========================================

  // 1. Dashboard Financial Metrics
  async function loadDashboardMetrics() {
    const res = await window.API.get('/tenant/metrics');
    if (!res.ok) {
      console.warn('Failed to load metrics:', res.error);
      return;
    }

    const m = res.data;
    if (!m) return;

    const elNetCash = document.getElementById('kpi-net-cash');
    const elNetBank = document.getElementById('kpi-net-bank');
    const elReceivables = document.getElementById('kpi-receivables');
    const elPayables = document.getElementById('kpi-payables');
    const elPendingSalary = document.getElementById('kpi-pending-salary');

    if (elNetCash) elNetCash.textContent = formatCurrency(m.cash_balance);
    if (elNetBank) elNetBank.textContent = formatCurrency(m.bank_balance);
    if (elReceivables) elReceivables.textContent = formatCurrency(m.total_receivable);
    if (elPayables) elPayables.textContent = formatCurrency(m.total_payable);
    if (elPendingSalary) elPendingSalary.textContent = formatCurrency(m.pending_salary);

    // Today stats
    if (m.today_stats) {
      const ts = m.today_stats;
      const elTodaySales = document.getElementById('kpi-today-sales');
      const elTodayPurchases = document.getElementById('kpi-today-purchases');
      const elTodayExpenses = document.getElementById('kpi-today-expenses');
      const elTodayAttendance = document.getElementById('kpi-today-attendance');

      if (elTodaySales) elTodaySales.textContent = formatCurrency(ts.total_sales);
      if (elTodayPurchases) elTodayPurchases.textContent = formatCurrency(ts.purchase_amount);
      if (elTodayExpenses) elTodayExpenses.textContent = formatCurrency(ts.expense_amount);
      if (elTodayAttendance) elTodayAttendance.textContent = `${ts.attendance_present} Present / ${ts.attendance_absent} Absent`;
    }
  }

  // 2. Tenant Users
  async function loadTenantUsers(page = 1) {
    if (currentRole !== 'admin') return;
    const search = document.getElementById('search-users')?.value || '';
    const res = await window.API.get(`/tenant/users?page=${page}&page_size=15&search=${encodeURIComponent(search)}`);
    const tbody = document.getElementById('users-table-body');
    if (!tbody) return;

    if (!res.ok || !Array.isArray(res.data) || res.data.length === 0) {
      tbody.innerHTML = `<tr><td colspan="5" style="text-align:center;color:var(--color-text-muted);padding:2rem;">No tenant users found.</td></tr>`;
      return;
    }

    tbody.innerHTML = res.data.map(u => `
      <tr>
        <td><strong>${u.name}</strong></td>
        <td>${u.email}</td>
        <td><span class="status-badge ${u.role === 'admin' ? 'paid' : 'cleared'}">${u.role.toUpperCase()}</span></td>
        <td><span class="status-badge ${u.status === 'active' ? 'paid' : 'pending'}">${u.status}</span></td>
        <td style="text-align:right;">
          <button class="btn btn-sm btn-secondary" onclick="Operations.deleteUser('${u.id}')">Delete</button>
        </td>
      </tr>
    `).join('');
  }

  // 3. Employees
  async function loadEmployees(page = 1) {
    if (currentRole !== 'admin') return;
    const search = document.getElementById('search-employees')?.value || '';
    const res = await window.API.get(`/tenant/employees?page=${page}&page_size=15&search=${encodeURIComponent(search)}`);
    const tbody = document.getElementById('employees-table-body');
    if (!tbody) return;

    if (!res.ok || !Array.isArray(res.data) || res.data.length === 0) {
      tbody.innerHTML = `<tr><td colspan="6" style="text-align:center;color:var(--color-text-muted);padding:2rem;">No employees registered. Click "Add Employee" to onboard staff.</td></tr>`;
      return;
    }

    cachedEmployees = res.data;
    populateSelectDropdowns();

    tbody.innerHTML = res.data.map(e => `
      <tr>
        <td><strong>${e.name}</strong></td>
        <td>${e.phone || '—'}</td>
        <td style="font-weight:600;">${formatCurrency(e.salary)}</td>
        <td>${formatCurrency(e.ot_rate)}/hr</td>
        <td><span class="status-badge ${e.status === 'active' ? 'paid' : 'pending'}">${e.status}</span></td>
        <td style="text-align:right;">
          <button class="btn btn-sm btn-secondary" onclick="Operations.deleteEmployee('${e.id}')">Delete</button>
        </td>
      </tr>
    `).join('');
  }

  // 4. Customers
  async function loadCustomers(page = 1) {
    if (currentRole !== 'admin') return;
    const search = document.getElementById('search-customers')?.value || '';
    const res = await window.API.get(`/tenant/customers?page=${page}&page_size=15&search=${encodeURIComponent(search)}`);
    const tbody = document.getElementById('customers-table-body');
    if (!tbody) return;

    if (!res.ok || !Array.isArray(res.data) || res.data.length === 0) {
      tbody.innerHTML = `<tr><td colspan="6" style="text-align:center;color:var(--color-text-muted);padding:2rem;">No customers added yet.</td></tr>`;
      return;
    }

    cachedCustomers = res.data;
    populateSelectDropdowns();

    tbody.innerHTML = res.data.map(c => `
      <tr>
        <td><strong>${c.customer_name}</strong></td>
        <td>${c.phone || '—'}</td>
        <td>${formatCurrency(c.opening_balance)}</td>
        <td style="font-weight:700;color:${parseFloat(c.current_balance) > 0 ? 'var(--color-warning)' : 'var(--color-text-main)'};">
          ${formatCurrency(c.current_balance)}
        </td>
        <td><span class="status-badge ${c.status === 'active' ? 'paid' : 'pending'}">${c.status}</span></td>
        <td style="text-align:right;">
          <button class="btn btn-sm btn-secondary" onclick="Operations.deleteCustomer('${c.id}')">Delete</button>
        </td>
      </tr>
    `).join('');
  }

  // 5. Banks
  async function loadBanks(page = 1) {
    if (currentRole !== 'admin') return;
    const res = await window.API.get(`/tenant/banks?page=${page}&page_size=15`);
    const tbody = document.getElementById('banks-table-body');
    if (!tbody) return;

    if (!res.ok || !Array.isArray(res.data) || res.data.length === 0) {
      tbody.innerHTML = `<tr><td colspan="5" style="text-align:center;color:var(--color-text-muted);padding:2rem;">No bank accounts configured.</td></tr>`;
      return;
    }

    cachedBanks = res.data;
    populateSelectDropdowns();

    tbody.innerHTML = res.data.map(b => `
      <tr>
        <td><strong>${b.bank_name}</strong></td>
        <td><code>${b.account_number || '—'}</code></td>
        <td>${b.ifsc || '—'}</td>
        <td><span class="status-badge ${b.status === 'active' ? 'paid' : 'pending'}">${b.status}</span></td>
        <td style="text-align:right;">
          <button class="btn btn-sm btn-secondary" onclick="Operations.deleteBank('${b.id}')">Delete</button>
        </td>
      </tr>
    `).join('');
  }

  // 6. Line Sales
  async function loadLineSales(page = 1) {
    const isUser = currentRole === 'user';
    const dateInput = document.getElementById('filter-line-sales-date');
    const date = isUser ? getTodayString() : (dateInput?.value || '');
    const search = document.getElementById('search-line-sales')?.value || '';

    let url = `/tenant/line-sales?page=${page}&page_size=15`;
    if (date) url += `&date=${date}`;
    if (search) url += `&search=${encodeURIComponent(search)}`;

    const res = await window.API.get(url);
    const tbody = document.getElementById('line-sales-table-body');
    if (!tbody) return;

    if (!res.ok || !Array.isArray(res.data) || res.data.length === 0) {
      tbody.innerHTML = `<tr><td colspan="8" style="text-align:center;color:var(--color-text-muted);padding:2rem;">No line sales recorded for this period.</td></tr>`;
      return;
    }

    tbody.innerHTML = res.data.map(s => `
      <tr>
        <td><strong>${s.customer_name}</strong></td>
        <td>${s.route || 'General'}</td>
        <td>${formatDate(s.created_at)}</td>
        <td style="font-weight:700;">${formatCurrency(s.total_amount)}</td>
        <td style="color:var(--color-success);font-weight:600;">+ ${formatCurrency(s.total_cash_in)}</td>
        <td style="color:${parseFloat(s.balance) > 0 ? 'var(--color-warning)' : 'var(--color-text-muted)'};font-weight:700;">
          ${formatCurrency(s.balance)}
        </td>
        <td><span class="status-badge ${parseFloat(s.balance) <= 0 ? 'paid' : 'pending'}">${parseFloat(s.balance) <= 0 ? 'Settled' : 'Credit Due'}</span></td>
        <td style="text-align:right;">
          <button class="btn btn-sm btn-secondary" onclick="Operations.deleteLineSale('${s.id}')">Delete</button>
        </td>
      </tr>
    `).join('');
  }

  // 7. Counter Sales
  async function loadCounterSales(page = 1) {
    const isUser = currentRole === 'user';
    const dateInput = document.getElementById('filter-counter-sales-date');
    const date = isUser ? getTodayString() : (dateInput?.value || '');

    let url = `/tenant/counter-sales?page=${page}&page_size=15`;
    if (date) url += `&date=${date}`;

    const res = await window.API.get(url);
    const tbody = document.getElementById('counter-sales-table-body');
    if (!tbody) return;

    if (!res.ok || !Array.isArray(res.data) || res.data.length === 0) {
      tbody.innerHTML = `<tr><td colspan="7" style="text-align:center;color:var(--color-text-muted);padding:2rem;">No counter sales recorded.</td></tr>`;
      return;
    }

    tbody.innerHTML = res.data.map(s => `
      <tr>
        <td><strong>${s.item}</strong></td>
        <td>${formatDate(s.created_at)}</td>
        <td><span class="status-badge cleared">${s.payment_method}</span></td>
        <td style="font-weight:700;">${formatCurrency(s.total_amount)}</td>
        <td style="color:var(--color-success);">${formatCurrency(s.cash)}</td>
        <td style="color:var(--color-warning);">${formatCurrency(s.account)}</td>
        <td style="text-align:right;">
          <button class="btn btn-sm btn-secondary" onclick="Operations.deleteCounterSale('${s.id}')">Delete</button>
        </td>
      </tr>
    `).join('');
  }

  // 8. Purchases
  async function loadPurchases(page = 1) {
    const isUser = currentRole === 'user';
    const date = isUser ? getTodayString() : (document.getElementById('filter-purchases-date')?.value || '');

    let url = `/tenant/purchases?page=${page}&page_size=15`;
    if (date) url += `&date=${date}`;

    const res = await window.API.get(url);
    const tbody = document.getElementById('purchases-table-body');
    if (!tbody) return;

    if (!res.ok || !Array.isArray(res.data) || res.data.length === 0) {
      tbody.innerHTML = `<tr><td colspan="7" style="text-align:center;color:var(--color-text-muted);padding:2rem;">No purchases found.</td></tr>`;
      return;
    }

    tbody.innerHTML = res.data.map(p => `
      <tr>
        <td><strong>${p.item}</strong></td>
        <td>${p.quantity}</td>
        <td>${formatDate(p.created_at)}</td>
        <td style="font-weight:700;">${formatCurrency(p.total_amount)}</td>
        <td style="color:var(--color-success);">${formatCurrency(p.total_paid)}</td>
        <td style="color:var(--color-error);font-weight:700;">${formatCurrency(p.total_pending)}</td>
        <td style="text-align:right;">
          <button class="btn btn-sm btn-secondary" onclick="Operations.deletePurchase('${p.id}')">Delete</button>
        </td>
      </tr>
    `).join('');
  }

  // 9. Expenses
  async function loadExpenses(page = 1) {
    const isUser = currentRole === 'user';
    const date = isUser ? getTodayString() : (document.getElementById('filter-expenses-date')?.value || '');

    let url = `/tenant/expenses?page=${page}&page_size=15`;
    if (date) url += `&date=${date}`;

    const res = await window.API.get(url);
    const tbody = document.getElementById('expenses-table-body');
    if (!tbody) return;

    if (!res.ok || !Array.isArray(res.data) || res.data.length === 0) {
      tbody.innerHTML = `<tr><td colspan="4" style="text-align:center;color:var(--color-text-muted);padding:2rem;">No expenses logged.</td></tr>`;
      return;
    }

    tbody.innerHTML = res.data.map(e => `
      <tr>
        <td><strong>${e.item}</strong></td>
        <td>${formatDate(e.created_at)}</td>
        <td style="font-weight:700;color:var(--color-error);">- ${formatCurrency(e.total_amount)}</td>
        <td style="text-align:right;">
          <button class="btn btn-sm btn-secondary" onclick="Operations.deleteExpense('${e.id}')">Delete</button>
        </td>
      </tr>
    `).join('');
  }

  // 10. Attendance
  async function loadAttendance(page = 1) {
    const dateInput = document.getElementById('filter-attendance-date');
    const date = dateInput?.value || getTodayString();

    const res = await window.API.get(`/tenant/attendance?page=${page}&page_size=50&date=${date}`);
    const tbody = document.getElementById('attendance-table-body');
    if (!tbody) return;

    if (!res.ok || !Array.isArray(res.data) || res.data.length === 0) {
      tbody.innerHTML = `<tr><td colspan="6" style="text-align:center;color:var(--color-text-muted);padding:2rem;">No attendance recorded for ${date}. Click "Mark Attendance" above.</td></tr>`;
      return;
    }

    tbody.innerHTML = res.data.map(a => `
      <tr>
        <td><strong>${a.employee_name || 'Staff Member'}</strong></td>
        <td>${a.date}</td>
        <td><span class="status-badge ${a.status === 'present' ? 'paid' : a.status === 'absent' ? 'pending' : 'cleared'}">${a.status.toUpperCase()}</span></td>
        <td>${a.ot > 0 ? `<span style="font-weight:700;color:var(--color-primary);">${a.ot} hrs OT</span>` : '—'}</td>
        <td>${parseFloat(a.advance) > 0 ? `<span style="color:var(--color-error);font-weight:700;">${formatCurrency(a.advance)}</span>` : '—'}</td>
        <td style="text-align:right;">
          <button class="btn btn-sm btn-secondary" onclick="Operations.openEditAttendance('${a.employee_id}', '${a.date}', '${a.status}', '${a.ot}', '${a.advance}')">Edit</button>
        </td>
      </tr>
    `).join('');
  }

  // 11. Salaries & Pending Salaries
  async function loadSalaries() {
    if (currentRole !== 'admin') return;

    const [allRes, pendingRes] = await Promise.all([
      window.API.get('/tenant/salaries?page_size=50'),
      window.API.get('/tenant/salaries/pending?page_size=50')
    ]);

    const tbodyAll = document.getElementById('salaries-table-body');
    const tbodyPending = document.getElementById('pending-salaries-table-body');

    if (tbodyAll && allRes.ok && Array.isArray(allRes.data)) {
      if (allRes.data.length === 0) {
        tbodyAll.innerHTML = `<tr><td colspan="5" style="text-align:center;color:var(--color-text-muted);padding:2rem;">No salary records found.</td></tr>`;
      } else {
        tbodyAll.innerHTML = allRes.data.map(s => `
          <tr>
            <td><strong>${s.employee_name}</strong></td>
            <td>${formatCurrency(s.salary_rate)}</td>
            <td>${formatCurrency(s.ot_rate)}/hr</td>
            <td style="font-weight:700;color:${parseFloat(s.balance) > 0 ? 'var(--color-warning)' : 'var(--color-success)'};">
              ${formatCurrency(s.balance)}
            </td>
            <td style="text-align:right;">
              <button class="btn btn-sm btn-primary" onclick="Operations.openPaySalary('${s.employee_id}', '${s.employee_name}', '${s.balance}')">Pay Salary</button>
              <button class="btn btn-sm btn-secondary" onclick="Operations.openAdjustSalary('${s.employee_id}', '${s.employee_name}', '${s.balance}')">Adjust</button>
            </td>
          </tr>
        `).join('');
      }
    }

    if (tbodyPending && pendingRes.ok && Array.isArray(pendingRes.data)) {
      if (pendingRes.data.length === 0) {
        tbodyPending.innerHTML = `<tr><td colspan="4" style="text-align:center;color:var(--color-success);padding:2rem;">✓ All employee salaries are settled! No dues pending.</td></tr>`;
      } else {
        tbodyPending.innerHTML = pendingRes.data.map(s => `
          <tr>
            <td><strong>${s.employee_name}</strong></td>
            <td>${formatCurrency(s.salary_rate)}</td>
            <td style="font-weight:700;color:var(--color-error);">${formatCurrency(s.balance)}</td>
            <td style="text-align:right;">
              <button class="btn btn-sm btn-primary" onclick="Operations.openPaySalary('${s.employee_id}', '${s.employee_name}', '${s.balance}')">Disburse Payment</button>
            </td>
          </tr>
        `).join('');
      }
    }
  }

  // ==========================================
  // Form Submission Actions
  // ==========================================

  async function submitAddUser(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-user');
    setButtonLoading(btn, true);

    const payload = {
      name: document.getElementById('user-form-name').value.trim(),
      email: document.getElementById('user-form-email').value.trim(),
      password: document.getElementById('user-form-password').value,
      role: document.getElementById('user-form-role').value,
      status: 'active'
    };

    const res = await window.API.post('/tenant/users', payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-add-user', res.error || 'Failed to create user');
      return;
    }

    showToast('Tenant user created successfully');
    closeModal('modal-add-user');
    loadTenantUsers();
  }

  async function submitAddEmployee(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-employee');
    setButtonLoading(btn, true);

    const payload = {
      name: document.getElementById('emp-form-name').value.trim(),
      phone: document.getElementById('emp-form-phone').value.trim(),
      salary: document.getElementById('emp-form-salary').value || '0',
      ot_rate: document.getElementById('emp-form-ot').value || '0',
      status: 'active'
    };

    const res = await window.API.post('/tenant/employees', payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-add-employee', res.error || 'Failed to onboard employee');
      return;
    }

    showToast('Employee onboarded successfully');
    closeModal('modal-add-employee');
    loadEmployees();
    loadDependencies();
  }

  async function submitAddCustomer(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-customer');
    setButtonLoading(btn, true);

    const payload = {
      customer_name: document.getElementById('cust-form-name').value.trim(),
      phone: document.getElementById('cust-form-phone').value.trim(),
      opening_balance: document.getElementById('cust-form-opening').value || '0',
      status: 'active'
    };

    const res = await window.API.post('/tenant/customers', payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-add-customer', res.error || 'Failed to add customer');
      return;
    }

    showToast('Customer registered successfully');
    closeModal('modal-add-customer');
    loadCustomers();
    loadDashboardMetrics();
    loadDependencies();
  }

  async function submitAddBank(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-bank');
    setButtonLoading(btn, true);

    const payload = {
      bank_name: document.getElementById('bank-form-name').value.trim(),
      account_number: document.getElementById('bank-form-account').value.trim(),
      ifsc: document.getElementById('bank-form-ifsc').value.trim(),
      status: 'active'
    };

    const res = await window.API.post('/tenant/banks', payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-add-bank', res.error || 'Failed to add bank account');
      return;
    }

    showToast('Bank account registered successfully');
    closeModal('modal-add-bank');
    loadBanks();
    loadDependencies();
  }

  async function submitAddLineSale(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-line-sale');
    setButtonLoading(btn, true);

    const customerID = document.getElementById('ls-form-customer').value;
    const totalAmount = document.getElementById('ls-form-amount').value || '0';
    const totalCashIn = document.getElementById('ls-form-cash').value || '0';
    const bankID = document.getElementById('ls-form-bank').value;
    const bankAmount = document.getElementById('ls-form-bank-amount').value || '0';

    const payments = [];
    if (parseFloat(totalCashIn) > 0) {
      payments.push({ payment_method: 'cash', amount: totalCashIn });
    }
    if (parseFloat(bankAmount) > 0) {
      payments.push({ payment_method: 'bank', bank_id: bankID || null, amount: bankAmount });
    }

    const payload = {
      customer_id: customerID,
      route: document.getElementById('ls-form-route').value.trim(),
      salesman: document.getElementById('ls-form-salesman').value.trim(),
      note: document.getElementById('ls-form-note').value.trim(),
      total_amount: totalAmount,
      total_cash_in: totalCashIn,
      payments: payments
    };

    const res = await window.API.post('/tenant/line-sales', payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-add-line-sale', res.error || 'Failed to record line sale');
      return;
    }

    showToast('Line sale recorded successfully');
    closeModal('modal-add-line-sale');
    loadLineSales();
    loadDashboardMetrics();
  }

  async function submitAddCounterSale(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-counter-sale');
    setButtonLoading(btn, true);

    const payload = {
      item: document.getElementById('cs-form-item').value.trim(),
      price: document.getElementById('cs-form-price').value || '0',
      total_amount: document.getElementById('cs-form-total').value || '0',
      payment_method: document.getElementById('cs-form-method').value,
      cash: document.getElementById('cs-form-cash').value || '0',
      account: document.getElementById('cs-form-account').value || '0'
    };

    const res = await window.API.post('/tenant/counter-sales', payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-add-counter-sale', res.error || 'Failed to record counter sale');
      return;
    }

    showToast('Counter sale recorded successfully');
    closeModal('modal-add-counter-sale');
    loadCounterSales();
    loadDashboardMetrics();
  }

  async function submitAddPurchase(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-purchase');
    setButtonLoading(btn, true);

    const totalAmount = document.getElementById('purch-form-total').value || '0';
    const totalPaid = document.getElementById('purch-form-paid').value || '0';
    const method = document.getElementById('purch-form-method').value || 'cash';
    const bankID = document.getElementById('purch-form-bank').value;

    const payments = [];
    if (parseFloat(totalPaid) > 0) {
      payments.push({
        payment_method: method,
        bank_id: bankID || null,
        amount: totalPaid
      });
    }

    const payload = {
      item: document.getElementById('purch-form-item').value.trim(),
      quantity: parseInt(document.getElementById('purch-form-quantity').value || '1', 10),
      total_amount: totalAmount,
      total_paid: totalPaid,
      payments: payments
    };

    const res = await window.API.post('/tenant/purchases', payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-add-purchase', res.error || 'Failed to record purchase');
      return;
    }

    showToast('Purchase recorded successfully');
    closeModal('modal-add-purchase');
    loadPurchases();
    loadDashboardMetrics();
  }

  async function submitAddExpense(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-expense');
    setButtonLoading(btn, true);

    const payload = {
      item: document.getElementById('exp-form-item').value.trim(),
      total_amount: document.getElementById('exp-form-amount').value || '0',
      payment_method: document.getElementById('exp-form-method').value || 'cash',
      bank_id: document.getElementById('exp-form-bank').value || null
    };

    const res = await window.API.post('/tenant/expenses', payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-add-expense', res.error || 'Failed to record expense');
      return;
    }

    showToast('Expense recorded successfully');
    closeModal('modal-add-expense');
    loadExpenses();
    loadDashboardMetrics();
  }

  async function submitMarkAttendance(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-attendance');
    setButtonLoading(btn, true);

    const payload = {
      employee_id: document.getElementById('att-form-emp').value,
      date: document.getElementById('att-form-date').value || getTodayString(),
      status: document.getElementById('att-form-status').value,
      ot: document.getElementById('att-form-ot').value || '0',
      advance: document.getElementById('att-form-advance').value || '0'
    };

    const res = await window.API.post('/tenant/attendance', payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-mark-attendance', res.error || 'Failed to mark attendance');
      return;
    }

    showToast('Attendance recorded');
    closeModal('modal-mark-attendance');
    loadAttendance();
    loadDashboardMetrics();
  }

  async function submitRecordOvertime(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-ot');
    setButtonLoading(btn, true);

    const payload = {
      employee_id: document.getElementById('ot-form-emp').value,
      date: document.getElementById('ot-form-date').value || getTodayString(),
      ot: document.getElementById('ot-form-hours').value || '0'
    };

    const res = await window.API.post('/tenant/overtime', payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-record-ot', res.error || 'Failed to record overtime');
      return;
    }

    showToast('Overtime recorded');
    closeModal('modal-record-ot');
    loadAttendance();
    loadSalaries();
    loadDashboardMetrics();
  }

  async function submitRecordAdvance(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-advance');
    setButtonLoading(btn, true);

    const payload = {
      employee_id: document.getElementById('adv-form-emp').value,
      date: document.getElementById('adv-form-date').value || getTodayString(),
      amount: document.getElementById('adv-form-amount').value || '0',
      payment_method: document.getElementById('adv-form-method').value || 'cash'
    };

    const res = await window.API.post('/tenant/advances', payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-record-advance', res.error || 'Failed to record advance payment');
      return;
    }

    showToast('Advance payment logged');
    closeModal('modal-record-advance');
    loadAttendance();
    loadSalaries();
    loadDashboardMetrics();
  }

  async function submitPaySalary(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-pay-salary');
    setButtonLoading(btn, true);

    const empId = document.getElementById('pay-salary-empid').value;
    const payload = {
      payment_method: document.getElementById('pay-salary-method').value || 'cash',
      bank_id: document.getElementById('pay-salary-bank').value || null,
      amount: document.getElementById('pay-salary-amount').value || '0',
      note: document.getElementById('pay-salary-note').value.trim()
    };

    const res = await window.API.post(`/tenant/salaries/${empId}/pay`, payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-pay-salary', res.error || 'Failed to process salary payment');
      return;
    }

    showToast('Salary payment processed');
    closeModal('modal-pay-salary');
    loadSalaries();
    loadDashboardMetrics();
  }

  async function submitAdjustSalary(e) {
    e.preventDefault();
    const btn = document.getElementById('btn-submit-adjust-salary');
    setButtonLoading(btn, true);

    const empId = document.getElementById('adj-salary-empid').value;
    const payload = {
      balance: document.getElementById('adj-salary-balance').value || '0'
    };

    const res = await window.API.put(`/tenant/salaries/${empId}`, payload);
    setButtonLoading(btn, false);

    if (!res.ok) {
      showModalError('modal-adjust-salary', res.error || 'Failed to adjust salary balance');
      return;
    }

    showToast('Salary balance adjusted');
    closeModal('modal-adjust-salary');
    loadSalaries();
  }

  // Delete helpers
  async function deleteUser(id) {
    if (!confirm('Are you sure you want to delete this tenant user?')) return;
    const res = await window.API.delete(`/tenant/users/${id}`);
    if (res.ok) {
      showToast('User removed');
      loadTenantUsers();
    } else {
      alert(res.error || 'Failed to delete user');
    }
  }

  async function deleteEmployee(id) {
    if (!confirm('Are you sure you want to delete this employee?')) return;
    const res = await window.API.delete(`/tenant/employees/${id}`);
    if (res.ok) {
      showToast('Employee removed');
      loadEmployees();
    } else {
      alert(res.error || 'Failed to delete employee');
    }
  }

  async function deleteCustomer(id) {
    if (!confirm('Are you sure you want to delete this customer?')) return;
    const res = await window.API.delete(`/tenant/customers/${id}`);
    if (res.ok) {
      showToast('Customer removed');
      loadCustomers();
    } else {
      alert(res.error || 'Failed to delete customer');
    }
  }

  async function deleteBank(id) {
    if (!confirm('Are you sure you want to delete this bank account?')) return;
    const res = await window.API.delete(`/tenant/banks/${id}`);
    if (res.ok) {
      showToast('Bank account removed');
      loadBanks();
    } else {
      alert(res.error || 'Failed to delete bank');
    }
  }

  async function deleteLineSale(id) {
    if (!confirm('Are you sure you want to delete this line sale?')) return;
    const res = await window.API.delete(`/tenant/line-sales/${id}`);
    if (res.ok) {
      showToast('Line sale removed');
      loadLineSales();
      loadDashboardMetrics();
    } else {
      alert(res.error || 'Failed to delete sale');
    }
  }

  async function deleteCounterSale(id) {
    if (!confirm('Are you sure you want to delete this counter sale?')) return;
    const res = await window.API.delete(`/tenant/counter-sales/${id}`);
    if (res.ok) {
      showToast('Counter sale removed');
      loadCounterSales();
      loadDashboardMetrics();
    } else {
      alert(res.error || 'Failed to delete counter sale');
    }
  }

  async function deletePurchase(id) {
    if (!confirm('Are you sure you want to delete this purchase?')) return;
    const res = await window.API.delete(`/tenant/purchases/${id}`);
    if (res.ok) {
      showToast('Purchase removed');
      loadPurchases();
      loadDashboardMetrics();
    } else {
      alert(res.error || 'Failed to delete purchase');
    }
  }

  async function deleteExpense(id) {
    if (!confirm('Are you sure you want to delete this expense?')) return;
    const res = await window.API.delete(`/tenant/expenses/${id}`);
    if (res.ok) {
      showToast('Expense removed');
      loadExpenses();
      loadDashboardMetrics();
    } else {
      alert(res.error || 'Failed to delete expense');
    }
  }

  // Open modals with pre-fill
  function openPaySalary(empId, empName, balance) {
    document.getElementById('pay-salary-empid').value = empId;
    document.getElementById('pay-salary-empname').textContent = empName;
    document.getElementById('pay-salary-pending').textContent = formatCurrency(balance);
    document.getElementById('pay-salary-amount').value = parseFloat(balance) > 0 ? balance : '';
    openModal('modal-pay-salary');
  }

  function openAdjustSalary(empId, empName, balance) {
    document.getElementById('adj-salary-empid').value = empId;
    document.getElementById('adj-salary-empname').textContent = empName;
    document.getElementById('adj-salary-balance').value = balance;
    openModal('modal-adjust-salary');
  }

  function openEditAttendance(employeeId, date, status, ot, advance) {
    const selEmp = document.getElementById('att-form-emp');
    if (selEmp) selEmp.value = employeeId;
    const inpDate = document.getElementById('att-form-date');
    if (inpDate) inpDate.value = date;
    const selStatus = document.getElementById('att-form-status');
    if (selStatus) selStatus.value = status;
    const inpOt = document.getElementById('att-form-ot');
    if (inpOt) inpOt.value = ot || '0';
    const inpAdv = document.getElementById('att-form-advance');
    if (inpAdv) inpAdv.value = advance || '0';
    openModal('modal-mark-attendance');
  }

  // Tab navigation
  function switchTab(tabName) {
    document.querySelectorAll('.tab-content').forEach(el => el.style.display = 'none');
    document.querySelectorAll('.nav-tab-btn').forEach(el => el.classList.remove('active'));

    const target = document.getElementById(`tab-${tabName}`);
    if (target) target.style.display = 'block';

    const btn = document.querySelector(`.nav-tab-btn[data-tab="${tabName}"]`);
    if (btn) btn.classList.add('active');

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
    }
  }

  // Initialize
  async function init() {
    currentUser = window.Auth ? window.Auth.getUser() : null;
    currentRole = (currentUser && currentUser.role) ? currentUser.role : 'user';

    // Show or hide admin-specific elements based on RBAC
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

    // Wire tab navigation buttons
    document.querySelectorAll('.nav-tab-btn').forEach(btn => {
      btn.addEventListener('click', (e) => {
        e.preventDefault();
        const tab = btn.dataset.tab;
        switchTab(tab);
      });
    });

    // Default dates on inputs
    const today = getTodayString();
    document.querySelectorAll('input[type="date"]').forEach(inp => {
      if (!inp.value) inp.value = today;
    });

    // Wire filter listeners for real-time reactivity
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

    // Wire backdrop click to close modal
    document.querySelectorAll('.modal-overlay').forEach(modal => {
      modal.addEventListener('click', (e) => {
        if (e.target === modal) {
          closeModal(modal.id);
        }
      });
    });

    // Wire form submit listeners
    const formUser = document.getElementById('form-add-user');
    if (formUser) formUser.addEventListener('submit', submitAddUser);

    const formEmp = document.getElementById('form-add-employee');
    if (formEmp) formEmp.addEventListener('submit', submitAddEmployee);

    const formCust = document.getElementById('form-add-customer');
    if (formCust) formCust.addEventListener('submit', submitAddCustomer);

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

    // Initial data load
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
    openEditAttendance,
    loadLineSales,
    loadCounterSales,
    loadPurchases,
    loadExpenses,
    loadAttendance,
    loadEmployees,
    loadCustomers,
    loadTenantUsers,
    loadSalaries
  };
})();

window.Operations = Operations;
