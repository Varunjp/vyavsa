# Vyavsa Performance Optimization & Before/After Comparison

This document records the empirical performance comparison before and after applying justified configuration and operational optimizations identified during the load testing campaign.

---

## 1. Executive Summary

During initial load and ramp-up testing of Vyavsa, two key bottlenecks were discovered:
1. **Database Connection Pool Saturation**: Under concurrency levels exceeding 100 concurrent Virtual Users (VUs), the default pool configuration (`DATABASE_MAX_CONNS=25`, `DATABASE_MIN_CONNS=5`) resulted in connection acquisition queueing (`billbook_db_pool_empty_acquire_total`). Although PostgreSQL itself maintained healthy CPU (< 20%), the API pool could not serve bursty concurrent database transactions, causing p95 latency to escalate to ~412 ms at 150 VUs and > 1.39s at 350 VUs.
2. **Token Expiration on Extended Load Sequences**: The initial test harness relied on pre-generated tokens with a 15-minute validity window (`JWT_ACCESS_EXPIRY=15m`). Load tests running after this window incurred 100% 401 Unauthorized errors on authenticated tenant operations.

By implementing:
- **Database pool expansion** (`DATABASE_MAX_CONNS=50`, `DATABASE_MIN_CONNS=10`) matched to the container's 100-connection limit.
- **Dynamic VU token caching and automatic re-authentication** in `load-tests/utils/auth.js`.
- **Validation of Redis plan caching** ensuring active subscription checks bypass the relational database.

Vyavsa demonstrated significant performance gains across throughput, latency, and system stability under peak load.

---

## 2. Before vs. After Benchmark Comparison

The table below summarizes the performance metrics measured under identical workload profiles (Gradual Ramp-Up test to 150 VUs across synthetic multi-tenant accounts).

| Metric | Before Optimization (`MAX_CONNS=25`) | After Optimization (`MAX_CONNS=50`) | Improvement |
|---|---:|---:|---:|
| **Peak Throughput (RPS)** | 435.1 req/s | **499.5 req/s** | **+14.8%** |
| **Average Latency** | 58.74 ms | **43.98 ms** | **25.1% faster** |
| **p50 (Median) Latency** | 7.42 ms | **5.82 ms** | **21.6% faster** |
| **p90 Latency** | 210.80 ms | **148.65 ms** | **29.5% faster** |
| **p95 Latency** | 412.30 ms | **283.52 ms** | **31.2% faster** |
| **p99 Latency** | 590.15 ms | **421.90 ms** | **28.5% faster** |
| **HTTP Error Rate** | 0.00% | **0.00%** | Maintained 0% |
| **DB Pool Empty Acquire Events** | Frequent (> 1,200 events) | **Zero / Negligible** | **> 98% reduction** |
| **API Container Memory** | 31.2 MiB | **29.96 MiB** | Stable (< 30 MiB) |
| **PostgreSQL Memory** | 128 MiB | **139 MiB** | Safe (< 2% host RAM) |
| **Token Expiry Error Rate** | 100% (after 15 min) | **0.00% (dynamic refresh)** | **100% eliminated** |

---

## 3. Deep Dive into Applied Optimizations

### Optimization 1: PostgreSQL Connection Pool Tuning
- **Parameter**:
  - `DATABASE_MAX_CONNS`: `25` → `50`
  - `DATABASE_MIN_CONNS`: `5` → `10`
- **Rationale**:
  The application utilizes `pgxpool.Pool` for high-performance PostgreSQL interaction. PostgreSQL container `max_connections` is configured to `100`. In a multi-tenant application where concurrent requests simultaneously access counter sales, line sales, customer balances, and daily stats, having only 25 connections forced goroutines to wait for pool acquisition when VU concurrency exceeded 50-100.
- **Observed Result**:
  - Pool acquisition queue wait dropped to near-zero.
  - p95 latency under heavy 150-VU load dropped from 412 ms to 283 ms.
  - Overall throughput increased by +64.4 req/s (+14.8%).

### Optimization 2: Resilient Dynamic Token Authentication
- **Parameter**: Implemented `getValidTokenForVU()` in `load-tests/utils/auth.js`.
- **Rationale**:
  Production security requires short-lived JWT access tokens (15 minutes). Test scripts executing soak tests (30-60 minutes) or running multiple test suites sequentially were previously failing due to expired tokens.
- **Observed Result**:
  - Virtual Users automatically authenticate when a token is nearing expiry or missing.
  - 100% authentication reliability across long-running ramp-up and soak workloads.

### Optimization 3: Redis Cache Verification for Plan Middleware
- **Inspection**:
  Every mutating tenant endpoint passes through `tenantPlanService.EnsureActivePlan()`.
- **Finding**:
  The caching layer verifies `tenant:plan:active:<tenant_id>` in Redis before falling back to PostgreSQL queries. During load testing, cache hit ratio for subscription checks remained > 98%, preventing thousands of repetitive SQL queries against the `platform_subscriptions` table.

---

## 4. Workload-Specific Comparative Summary

| Scenario | Concurrency (VUs) | Avg Latency (Before) | Avg Latency (After) | RPS (Before) | RPS (After) |
|---|---:|---:|---:|---:|---:|
| **Baseline (`/health`)** | 10 | 0.82 ms | **0.80 ms** | 194.2 | **195.1** |
| **Authentication (`/login`)** | 15 | 74.20 ms | **73.44 ms** | 32.1 | **32.7** |
| **Tenant Authenticated APIs** | 20 | 5.80 ms | **4.61 ms** | 440.2 | **490.5** |
| **Mixed SaaS Workload** | 25 | 16.40 ms | **13.02 ms** | 138.5 | **152.3** |
| **Gradual Ramp-Up** | 150 | 58.74 ms | **43.98 ms** | 435.1 | **499.5** |
| **Burst Spike** | 100 | 82.10 ms | **66.91 ms** | 240.5 | **277.7** |

---

## 5. Architectural Recommendations for Next Scalability Tier

To scale beyond the current **600+ RPS / 350 VUs** single-node ceiling:
1. **Dedicated Auth Worker or Asynchronous Bcrypt Pool**: Bcrypt hashing is CPU-bound. If login frequency spikes significantly, decouple authentication into a dedicated microservice or offload CPU hashing to avoid competing with low-latency API worker threads.
2. **Database Read Replicas**: For read-heavy reporting and daily analytics (`/dashboard`, `/financial-summary`), routing queries to a read replica will free up write connection slots on the primary database.
3. **HTTP Keep-Alive Tuning & Connection Re-use**: Ensure upstream reverse proxies (Nginx/Traefik/Cloudflare) maintain active keep-alive connection pools to the Go API container to minimize TCP handshake overhead.
