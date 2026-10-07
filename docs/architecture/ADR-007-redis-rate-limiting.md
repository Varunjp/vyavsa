# ADR-007: Distributed Redis Rate Limiting

## Status
Accepted

## Context
Vyavsa is a multi-tenant SaaS application that requires robust defense against brute-force attacks on authentication endpoints (login, registration, OTP verification, password resets) as well as API quota enforcement for tenant accounts and public endpoints. In a multi-instance deployment, rate limiting in local process memory allows clients to multiply limits across replicas. A distributed, atomic, low-overhead mechanism backed by shared infrastructure is necessary.

## Decision
1. **Shared State Store**: Redis 7 is used as the shared state store via `internal/repository/redis/rate_limit_redis.go`.
2. **Atomic Algorithm**: Implemented fixed-window counting via an atomic Redis Lua script (`INCR` + `EXPIRE` + `TTL`) ensuring race-free increment and exact remaining TTL calculation in a single network round trip.
3. **Multi-Tenant Key Strategy**:
   - Authenticated tenant endpoints are scoped by `tenant_id` (`ratelimit:tenant:<tenant_id>`). All users under the tenant share the tenant's API quota, preventing limit bypass via account rotation.
   - Public and authentication endpoints are scoped by client IP (`ratelimit:auth:<ip>`, `ratelimit:security:<ip>`, `ratelimit:general:<ip>`).
   - Platform admin endpoints are scoped by admin `user_id` (`ratelimit:platform:<user_id>`).
4. **Differentiated Tiers**:
   - `general`: 100 req / 60s
   - `auth`: 20 req / 60s
   - `security`: 5 req / 60s
   - `tenant`: 300 req / 60s
   - `platform`: 300 req / 60s
   - Operational endpoints (`/health`, `/ready`, `/metrics`) are explicitly exempt.
5. **Failure Policy**:
   - Configurable via `RATE_LIMIT_FAIL_OPEN` (default `true` for general/tenant business APIs).
   - Configurable via `RATE_LIMIT_AUTH_FAIL_OPEN` (default `false` for security/auth endpoints to prevent brute force if Redis is unavailable).
6. **Observability**: Prometheus metrics (`billbook_rate_limit_requests_total`, `billbook_rate_limit_rejected_total`, `billbook_rate_limit_redis_errors_total`) and structured warning logs without PII leakage.

## Consequences
- **Positive**: Consistent, horizontal enforcement across any number of API instances; atomic Lua script eliminates concurrency race conditions; strict security protection for authentication endpoints.
- **Negative**: Adds a fast O(1) Redis call per incoming HTTP API request; mitigated by reusing connection pool and avoiding extra round-trips.
