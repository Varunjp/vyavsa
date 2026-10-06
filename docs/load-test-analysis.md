# Vyavsa Load & Performance Test Analysis

## 1. Application Architecture Overview

Vyavsa is a multi-tenant SaaS application designed for small businesses (retail, kirana, distributors). It is engineered as a clean modular monolith using the Go programming language and Gin Web Framework, backed by PostgreSQL 16 (relational data store) and Redis 7 (in-memory caching, token blacklisting, and worker queues).

### Core Architectural Layers:
1. **Network & Routing**: Gin HTTP Engine with centralized middleware chain:
   - `RequestID`: Correlation ID propagation (`X-Request-ID`)
   - `Recovery`: Panic recovery returning RFC-compliant 500 JSON error
   - `SecurityHeaders`: CSP, HSTS, X-Content-Type-Options
   - `CORS`: Configurable CORS origins
   - `RequestLogger`: Structured request/response logging + Prometheus duration tracking
2. **Authentication & Authorization**:
   - JWT HMAC-SHA256 (Access token: 15 min default, Refresh token: 7 days default)
   - Redis-backed Token Blacklist: JTI individual revocation + User-level epoch revocation
   - RBAC: Role-based authorization (`platform_admin`, `admin`, `user`)
   - Active Plan Middleware: Cache-first plan enforcement ensuring mutating tenant calls (POST, PUT, PATCH, DELETE) have an active subscription
3. **Business Services & Data Access**:
   - Strongly typed Service layer (`AuthService`, `TenantService`, `TenantOperationsService`, `PlatformPlanService`, `TenantPlanService`)
   - Repository layer implemented via `pgxpool.Pool` (PostgreSQL 16) with parameterized queries and transactional atomicity (`Transactor`)
   - Redis Cache Layer (`go-redis/v9`) for plan subscription caching, OTP verification, and rate limiting
   - Asynchronous background workers: `EmailWorker` (SMTP dispatch with retries) and `DailyStatsWorker` (asynchronous aggregation)
4. **Observability**:
   - Native Prometheus registry (`/metrics`) instrumented with HTTP metrics, business counters, DB pool metrics, and Redis pool metrics
   - Grafana dashboards with provisioned dashboards
   - cAdvisor and Node Exporter for container and OS host level observability

---

## 2. Implemented HTTP Endpoints Discovered

A comprehensive audit of `internal/server/server.go` reveals the complete set of implemented HTTP endpoints. **No non-existent endpoints are assumed.**

### A. Public & Operational Endpoints
| Method | Path | Auth Required | Description |
|---|---|---|---|
| GET | `/health` | No | Liveness probe (process responsive) |
| GET | `/ready` | No | Readiness probe (verifies Postgres & Redis pings) |
| GET | `/metrics` | No | Prometheus time-series metrics |
| GET | `/api/v1/ping` | No | Basic API heartbeat ping |
| GET | `/api/v1/plans` | No | Catalog of available subscription plans |
| GET | `/` | No | Public landing page (HTML) |
| GET | `/login`, `/register`, etc. | No | Web UI view templates |

### B. Authentication & Password Recovery Endpoints
| Method | Path | Auth Required | Description |
|---|---|---|---|
| POST | `/api/v1/auth/platform/login` | No | Authenticate platform admin (returns JWT pair) |
| POST | `/api/v1/auth/tenant/login` | No | Authenticate tenant user (returns JWT pair) |
| POST | `/api/v1/auth/tenant/register` | No | Self-service tenant registration (creates tenant + admin + trial) |
| POST | `/api/v1/tenants/register` | No | Alias for tenant self-service registration |
| POST | `/api/v1/auth/refresh` | No | Exchange refresh token for new access/refresh pair |
| POST | `/api/v1/auth/revoke` | No | Invalidate refresh token |
| POST | `/api/v1/auth/forgot-password` | No | Request 6-digit OTP to email (rate limited in Redis) |
| POST | `/api/v1/auth/verify-reset-otp` | No | Verify 6-digit OTP, receive password reset token |
| POST | `/api/v1/auth/reset-password` | No | Set new password with reset token |
| POST | `/api/v1/auth/logout` | Bearer JWT | Blacklist access token JTI in Redis |
| GET | `/api/v1/auth/me` | Bearer JWT | Retrieve authenticated user profile |

### C. Tenant Member Endpoints (`/api/v1/tenant/*`)
*Requires: Bearer JWT (`role: admin` or `role: user`) + Active Subscription Plan for mutations*

#### 1. Profile & Dashboard Analytics
| Method | Path | Role | Description |
|---|---|---|---|
| GET | `/api/v1/tenant/ping` | user / admin | Heartbeat for authenticated tenant |
| GET | `/api/v1/tenant/profile` | user / admin | Tenant business profile |
| GET | `/api/v1/tenant/financial-summary`| user / admin | Real-time cash, bank, receivable, payable balances |
| GET | `/api/v1/tenant/subscription` | user / admin | Current subscription details and plan status |
| GET | `/api/v1/tenant/dashboard` | user / admin | Aggregated financial summary & daily statistics |
| GET | `/api/v1/tenant/dashboard/today`| user / admin | Today's sales, attendance, and operational overview |
| GET | `/api/v1/tenant/metrics` | user / admin | Financial metrics and KPI cards |
| GET | `/api/v1/tenant/daily-stats` | user / admin | Query historical daily aggregated stats |

#### 2. Sales Operations
| Method | Path | Role | Description |
|---|---|---|---|
| POST | `/api/v1/tenant/line-sales` | user / admin | Create line sale (customer credit/cash ledger) |
| GET | `/api/v1/tenant/line-sales` | user / admin | List line sales (supports date, customer, pagination) |
| GET | `/api/v1/tenant/line-sales/:id` | user / admin | Get single line sale details |
| POST | `/api/v1/tenant/line-sales/:id/payments` | user / admin | Record payment against line sale (updates balance) |
| POST | `/api/v1/tenant/counter-sales` | user / admin | Create counter sale (retail counter billing) |
| GET | `/api/v1/tenant/counter-sales` | user / admin | List counter sales |
| POST | `/api/v1/tenant/counter-sales/:id/payments` | user / admin | Record payment against counter sale |

#### 3. Purchases & Expenses
| Method | Path | Role | Description |
|---|---|---|---|
| POST | `/api/v1/tenant/purchases` | user / admin | Record inventory purchase from supplier |
| GET | `/api/v1/tenant/purchases` | user / admin | List purchases |
| POST | `/api/v1/tenant/purchases/:id/payments`| user / admin | Record payment to supplier |
| POST | `/api/v1/tenant/expenses` | user / admin | Record operational expense (cash/bank outflow) |
| GET | `/api/v1/tenant/expenses` | user / admin | List expenses |

#### 4. Human Resources & Operations
| Method | Path | Role | Description |
|---|---|---|---|
| POST | `/api/v1/tenant/attendance` | user / admin | Record employee attendance |
| GET | `/api/v1/tenant/attendance` | user / admin | List employee attendance records |
| POST | `/api/v1/tenant/overtime` | user / admin | Record employee overtime hours |
| GET | `/api/v1/tenant/overtime` | user / admin | List employee overtime records |
| POST | `/api/v1/tenant/advances` | user / admin | Record salary advance disbursed |
| GET | `/api/v1/tenant/advances` | user / admin | List employee salary advances |

### D. Tenant Administrator Exclusive Endpoints
*Requires: Bearer JWT (`role: admin`)*

| Method | Path | Description |
|---|---|---|
| PUT | `/api/v1/tenant/settings` | Update tenant business settings |
| POST | `/api/v1/tenant/subscription/purchase` | Purchase or upgrade subscription plan |
| GET | `/api/v1/tenant/transactions` | List subscription payment transactions |
| POST | `/api/v1/tenant/users` | Create new staff user under tenant |
| GET | `/api/v1/tenant/users` | List tenant users |
| POST | `/api/v1/tenant/employees` | Create employee master record |
| GET | `/api/v1/tenant/employees` | List employee records |
| POST | `/api/v1/tenant/customers` | Create customer account |
| GET | `/api/v1/tenant/customers` | List customer accounts |
| GET | `/api/v1/tenant/customers/:id/balance` | Customer balance check |
| POST | `/api/v1/tenant/banks` | Register tenant bank account |
| GET | `/api/v1/tenant/banks` | List bank accounts and balances |
| GET | `/api/v1/tenant/salaries` | View salary ledger and pending salaries |
| POST | `/api/v1/tenant/salaries/:employee_id/pay` | Disburse employee salary |

### E. Platform Administrator Endpoints (`/api/v1/platform/*`)
*Requires: Bearer JWT (`role: platform_admin`)*

| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/platform/ping` | Platform admin ping |
| POST | `/api/v1/platform/plans` | Create subscription plan |
| GET | `/api/v1/platform/plans` | List platform subscription plans |
| POST | `/api/v1/platform/tenants` | Onboard new tenant manually |
| GET | `/api/v1/platform/tenants` | List all platform tenants |
| GET | `/api/v1/platform/dashboard/metrics` | Platform overview KPIs (MRR, active tenants) |
| GET | `/api/v1/platform/subscriptions` | List platform subscriptions |
| GET | `/api/v1/platform/transactions` | List platform billing transactions |

---

## 3. Authentication & Security Flow

1. **Public Login (`POST /api/v1/auth/tenant/login`)**:
   - Request: `{"email": "...", "password": "..."}`
   - Query: `tenant_user` table lookup by email.
   - Validation: Bcrypt password hash comparison (`bcrypt.CompareHashAndPassword`).
   - Token Generation: Signs HMAC-SHA256 JWT access token (15m expiry) and refresh token (7d expiry) containing claims: `user_id`, `tenant_id`, `email`, `role`, `user_type`, and unique `jti`.
2. **Authenticated Request Validation (`middleware.Authenticate`)**:
   - Extracts `Bearer <token>` from `Authorization` header.
   - Verifies JWT HMAC signature and expiration.
   - Checks Redis token blacklist: `GET blacklist:token:<jti>` (O(1)) and `GET blacklist:user:<user_id>` (O(1)).
   - Injects claims into `gin.Context`.
3. **Tenant & Plan Validation (`middleware.RequireTenantUser` + `middleware.RequireActivePlan`)**:
   - Validates user belongs to a tenant (`tenant_id != nil`).
   - For mutating methods (`POST`, `PUT`, `PATCH`, `DELETE`), checks active plan status via Redis cache (`plan_cache:tenant:<tenant_id>`), falling back to PostgreSQL if cache miss.
   - Read-only (`GET`) requests pass through unconditionally.

---

## 4. Dependencies & Infrastructure Impact

### A. PostgreSQL Dependencies
- **Connection Pool**: `pgxpool.Pool` configured with `DATABASE_MAX_CONNS` (default: 25) and `DATABASE_MIN_CONNS` (default: 5).
- **Concurrency & Transactions**: Operations modifying financials (payments, expenses, salary disbursement, line sales) acquire row locks and run within database transactions (`Transactor.WithinTransaction`).
- **Indexes**: Primary keys (UUID), foreign keys (`tenant_id`), composite unique indexes (`tenant_id, email`), and performance indexes (`idx_line_sale_tenant_date`, `idx_counter_sale_tenant_date`, `idx_attendance_tenant_date`).

### B. Redis Dependencies
- **Token Blacklisting**: Revocation checks on every authenticated request.
- **Plan Status Cache**: Fast O(1) plan status verification for mutating calls.
- **Worker Queues**: Redis lists/channels for asynchronous tasks.
- **Rate Limiting**: Password recovery brute-force protection.

### C. Observability Stack
- **Prometheus**: Metrics scraped every 15s from `/metrics`:
  - `billbook_http_requests_total`
  - `billbook_http_request_duration_seconds`
  - `billbook_http_requests_in_flight`
  - `billbook_db_pool_acquired_connections`, `billbook_db_pool_idle_connections`, `billbook_db_pool_empty_acquire_total`
  - `billbook_redis_pool_hits_total`, `billbook_redis_pool_misses_total`
  - `billbook_financial_tx_duration_seconds`
- **Grafana**: Dashboards configured at `:3000`.
- **cAdvisor / Node Exporter**: Container and host CPU/RAM/IO tracking.

---

## 5. Potential Performance Bottlenecks

1. **Bcrypt Computation on Auth Endpoints**: Bcrypt hashing is deliberately CPU intensive. A high concurrency of `/api/v1/auth/tenant/login` requests will rapidly saturate CPU cores before saturating network or database bandwidth.
2. **PostgreSQL Connection Pool Sizing**: With `DATABASE_MAX_CONNS=25`, if more than 25 concurrent requests perform simultaneous database queries or transactions, subsequent requests will queue waiting for an available connection (`empty_acquire_total` and `db_connection_wait_seconds`).
3. **Database Financial Transaction Locks**: Financial summaries and bank balance updates lock the `tenant_financial_summary` and `tenant_bank` rows for that specific tenant. High concurrency against the *same* tenant will serialize on row locks, whereas multi-tenant concurrency distributes locks across rows.
4. **Redis Connection Exhaustion**: If Redis connection pool hits max connections during high traffic spikes, latency on token blacklist verification will elevate across all authenticated endpoints.

---

## 6. Endpoints Selected for Load Testing

| Category | Endpoint | Rationale |
|---|---|---|
| **Baseline (Unauthenticated)** | `GET /health` | Establishes absolute baseline of Gin routing and HTTP stack overhead without DB/Redis |
| **Operational Health** | `GET /ready` | Validates end-to-end dependency pinging (Postgres + Redis) |
| **Authentication** | `POST /api/v1/auth/tenant/login` | Measures Bcrypt CPU cost and token issuance throughput |
| **Read-Heavy Authenticated** | `GET /api/v1/tenant/profile`<br>`GET /api/v1/tenant/customers`<br>`GET /api/v1/tenant/banks` | Core read queries scoped by `tenant_id` verifying DB indexes |
| **Dashboard / Aggregate** | `GET /api/v1/tenant/dashboard`<br>`GET /api/v1/tenant/financial-summary` | Queries combining financial summary balances and daily aggregates |
| **Mutating / Transaction** | `POST /api/v1/tenant/customers`<br>`POST /api/v1/tenant/line-sales`<br>`POST /api/v1/tenant/counter-sales` | Transactional writes verifying `RequireActivePlan` cache + DB transaction performance |

---

## 7. Recommended Load Test Scenarios

1. **Baseline**: 10 VUs for 2 minutes on `GET /health` (target: < 5ms p95, 0% error).
2. **Authentication Load**: 10-50 VUs logging in across multiple test accounts to evaluate Bcrypt CPU limits.
3. **Tenant Read & Dashboard Load**: 50-100 VUs querying profile, customers, banks, and dashboard.
4. **Multi-Tenant Mixed Workload**: 40% Read, 25% Dashboard/Stats, 15% Write/Create, 10% Auth, 10% Health across multiple tenants.
5. **Ramp-Up Test**: Step from 10 → 25 → 50 → 100 → 150 VUs to determine latency inflection points.
6. **Stress Test**: Increase load until error rate approaches 5% or p95 > 1000ms to identify system capacity.
7. **Spike Test**: Burst from 10 to 200+ VUs within 10s to test system resilience and recovery.
8. **Soak Test**: Sustained load over 10-15 minutes to verify connection pooling stability and absence of memory/goroutine leaks.
