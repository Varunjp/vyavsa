# Vyavsa Load & Performance Testing Suite

This directory contains the production-oriented load testing suite for the Vyavsa multi-tenant SaaS application, built with **Grafana k6**.

---

## 1. Directory Structure

```text
load-tests/
├── config.js               # Central configuration, env vars, safety gates & thresholds
├── README.md               # Complete operational documentation
├── data/
│   └── test-users.json     # Generated synthetic multi-tenant dataset (ignored by git)
├── scenarios/
│   ├── health.js           # Baseline health & readiness probe test
│   ├── authentication.js   # Tenant user login, Bcrypt hashing & token lifecycle
│   ├── tenant.js           # Authenticated multi-tenant APIs (read, dashboard, write)
│   ├── mixed-workload.js   # Realistic multi-tenant mixed SaaS workload
│   ├── ramp-up.js          # Gradual step ramp-up test
│   ├── stress.js           # Breaking-point stress test
│   ├── spike.js            # High-traffic surge & recovery test
│   └── soak.js             # Longevity & stability soak test
└── utils/
    ├── auth.js             # Authentication helpers (login, refresh, logout)
    ├── data.js             # Multi-tenant user loader & payload generators
    └── helpers.js          # Common helpers (headers, checks, random generators)
```

---

## 2. Environment Variables & Safety Gates

Tests are governed by configuration variables. **Never hardcode secrets, tokens, or credentials.**

| Variable | Default Value | Description |
|---|---|---|
| `BASE_URL` | `http://localhost:8080` | Target API gateway URL |
| `LOAD_TEST_ENABLED` | `true` | Safety switch. Tests abort if `false`. |
| `ENVIRONMENT` / `APP_ENV` | `development` | Target environment. Aborts immediately if set to `production`. |
| `TEST_EMAIL` | `admin@demostore.com` | Fallback single-tenant user email |
| `TEST_PASSWORD` | `Store@12345` | Fallback single-tenant user password |
| `TEST_TENANT_ID` | `11111111-1111-1111-1111-111111111111` | Fallback single-tenant ID |
| `ACCESS_TOKEN` | `""` | Optional pre-issued access token bypass |
| `VUS` | Scenario default | Number of concurrent Virtual Users |
| `DURATION` | Scenario default | Test duration (e.g., `60s`, `5m`, `30m`) |
| `MIX_READ` | `40` | Percentage weight of read operations in mixed workload |
| `MIX_DASHBOARD` | `25` | Percentage weight of dashboard operations in mixed workload |
| `MIX_WRITE` | `15` | Percentage weight of write mutations in mixed workload |
| `MIX_AUTH` | `10` | Percentage weight of auth/login in mixed workload |
| `MIX_HEALTH` | `10` | Percentage weight of health probes in mixed workload |

### Database & Environment Safety
Tests refuse to execute if:
1. `ENVIRONMENT=production` or `APP_ENV=production`.
2. `LOAD_TEST_ENABLED` is not set to `true` or `1`.

---

## 3. Synthetic Multi-Tenant Test Data Management

The load test suite uses synthetic test tenants tagged with `loadtest_%@vyavsa-test.internal` and `LoadTest Tenant %03d`.

### Step 1: Seed Synthetic Tenants and Users
```bash
# Using Makefile (creates 50 tenants with 2 users each by default)
make load-test-setup

# Or custom tenant volume
make load-test-setup TENANTS=100 USERS_PER_TENANT=5
```
This generates `load-tests/data/test-users.json` containing credentials, sample customer IDs, sample bank IDs, and valid tokens.

### Step 2: Check Current Synthetic Data Status
```bash
make load-test-status
```

### Step 3: Clean Up All Synthetic Data
```bash
make load-test-teardown
```
All synthetic tenants and their cascaded database records (users, banks, customers, transactions) are deleted cleanly.

---

## 4. Executing Load Tests

### A. Makefile Commands
```bash
# 1. Baseline Test (GET /health and GET /ready)
make load-test-baseline

# 2. Authentication Test (POST /api/v1/auth/tenant/login across users)
make load-test-auth

# 3. Multi-Tenant API Test (Read, Dashboard, Writes)
make load-test-tenant

# 4. Realistic Mixed Workload Test (40% Read, 25% Dash, 15% Write, 10% Auth, 10% Health)
make load-test-mixed

# 5. Gradual Ramp-Up Test (Step concurrency 25 -> 50 -> 100 -> 150 VUs)
make load-test-rampup

# 6. Stress Test (Determine maximum stable RPS and saturation limits)
make load-test-stress

# 7. Spike Test (Sudden burst from 10 to 250 VUs)
make load-test-spike

# 8. Soak Test (Endurance test for memory/connection leak detection)
make load-test-soak

# Complete Standard Verification Run
make load-test
```

### B. Direct k6 Commands
```bash
# Baseline Health Test (10 VUs for 2m)
BASE_URL=http://localhost:8080 VUS=10 DURATION=2m k6 run load-tests/scenarios/health.js

# Authentication Test
BASE_URL=http://localhost:8080 VUS=20 DURATION=1m k6 run load-tests/scenarios/authentication.js

# Authenticated Multi-Tenant Test
BASE_URL=http://localhost:8080 VUS=25 DURATION=2m k6 run load-tests/scenarios/tenant.js

# Mixed Workload Test
BASE_URL=http://localhost:8080 VUS=30 DURATION=2m k6 run load-tests/scenarios/mixed-workload.js
```

---

## 5. Performance Thresholds & SLOs

| Category | HTTP Error Rate | p95 Latency | p99 Latency | Notes |
|---|---|---|---|---|
| Health Probes | < 0.1% | < 30ms | < 60ms | Framework and routing overhead |
| Tenant Reads | < 1.0% | < 300ms | < 600ms | Multi-tenant filtered SQL queries |
| Dashboard Aggregates | < 1.0% | < 400ms | < 800ms | Financial metrics and stats |
| Mutating Writes | < 1.0% | < 500ms | < 1000ms | PostgreSQL transactions with plan cache |
| Authentication | < 2.0% | < 1000ms | < 2000ms | CPU-bound Bcrypt hashing |
| Stress Tests | < 5.0% | < 1500ms | < 3000ms | Near saturation capacity |

---

## 6. Real-Time Observability & Monitoring

While executing tests, monitor:
- **Grafana Dashboard**: `http://localhost:3000` (Credentials: `admin` / `admin`)
- **Prometheus UI**: `http://localhost:9090`
- **Application Metrics**: `http://localhost:8080/metrics`
- **PostgreSQL Connection Pool**:
  - `billbook_db_pool_acquired_connections`
  - `billbook_db_pool_idle_connections`
  - `billbook_db_pool_empty_acquire_total`
- **Redis Pool Metrics**:
  - `billbook_redis_pool_hits_total`
  - `billbook_redis_pool_misses_total`
