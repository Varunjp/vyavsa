# Distributed Redis Rate Limiting

## 1. Overview & Architecture

Vyavsa enforces distributed rate limiting across all API entry points using a shared Redis instance. This ensures identical enforcement across multiple horizontally scaled application instances without relying on process-local memory.

```text
HTTP Request
     ↓
Rate Limiter Middleware (internal/middleware/ratelimit.go)
     ↓
Rate Limiter Service    (internal/service/rate_limiter_service.go)
     ↓
Rate Limiter Repository (internal/repository/redis/rate_limit_redis.go)
     ↓
Redis (Atomic Lua script: INCR + EXPIRE + TTL)
     ↓
Allow / Reject (HTTP 429 Too Many Requests + Retry-After)
     ↓
Handler (Business Logic)
```

### Distributed Multi-Instance Correctness
When multiple application instances run behind a reverse proxy or load balancer, every instance points to the same Redis cluster/instance. Counters are synchronized in Redis via atomic Lua script execution (`rateLimitLuaScript`), preventing race conditions, double increments, and quota bypass across replicas.

## 2. Algorithm & Atomic Operations

The rate limiter employs an atomic **Fixed-Window Counter** with automatic key expiration executed inside a single Redis Lua script round-trip:

```lua
local current = redis.call('INCR', KEYS[1])
if current == 1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
end
local ttl = redis.call('TTL', KEYS[1])
if ttl == -1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
    ttl = tonumber(ARGV[1])
end
return {current, ttl}
```

- **Atomicity**: The script executes atomically in Redis single-threaded engine, preventing any race condition between reading, incrementing, and setting TTL.
- **Single Round-Trip**: Returns `{current_count, remaining_ttl}` in a single network round trip.
- **Self-Healing TTL**: If a key ever exists without an expiration (`ttl == -1`), expiration is automatically reinstated.

## 3. Key Design Strategy & Multi-Tenant Isolation

Keys are prefixed and segmented by tier and identifier:

```text
ratelimit:{tier}:{identifier}
```

| Tier | Identifier Source | Sample Key | Purpose & Isolation |
|---|---|---|---|
| `general` | Client IP (`c.ClientIP()`) | `ratelimit:general:192.0.2.1` | General public API throttling |
| `auth` | Client IP (`c.ClientIP()`) | `ratelimit:auth:192.0.2.1` | Login, registration, token refresh abuse protection |
| `security` | Client IP (`c.ClientIP()`) | `ratelimit:security:192.0.2.1` | High-risk OTP generation/verification & password resets |
| `tenant` | Tenant UUID (`auth.GetTenantID(c)`) | `ratelimit:tenant:123e4567-e89b-12d3-a456-426614174000` | **Multi-tenant isolation**: All users within the same tenant share the tenant quota. A tenant cannot bypass limits by creating multiple user accounts. |
| `platform` | Admin User UUID (`auth.GetUserID(c)`) | `ratelimit:platform:987fcdeb-51a2-43d7-9012-345678901234` | Platform administrator endpoints quota |

## 4. Protected Endpoints & Default Limits

| Endpoint Group | Tier | Default Limit | Window | Fail Policy |
|---|---|---|---|---|
| `/api/v1/ping`, `/api/v1/plans` | `general` | 100 req | 60s | Fail Open |
| `/api/v1/auth/platform/login` | `auth` | 20 req | 60s | Fail Closed |
| `/api/v1/auth/tenant/login` | `auth` | 20 req | 60s | Fail Closed |
| `/api/v1/auth/tenant/register` | `auth` | 20 req | 60s | Fail Closed |
| `/api/v1/auth/refresh`, `/revoke` | `auth` | 20 req | 60s | Fail Closed |
| `/api/v1/auth/forgot-password` | `security` | 5 req | 60s | Fail Closed |
| `/api/v1/auth/verify-reset-otp` | `security` | 5 req | 60s | Fail Closed |
| `/api/v1/auth/reset-password` | `security` | 5 req | 60s | Fail Closed |
| `/api/v1/tenant/*`, `/api/v1/reports/*` | `tenant` | 300 req | 60s | Fail Open |
| `/api/v1/platform/*` | `platform` | 300 req | 60s | Fail Open |

### Explicitly Exempt Endpoints
Operational and observability endpoints are **never** subject to rate limiting:
- `/health` (Liveness probe)
- `/ready` (Readiness probe)
- `/metrics` (Prometheus metrics scraper)
- `/static/*` and HTML template rendering routes

## 5. Environment Configuration

All rate limiting variables are defined in `.env` and `internal/config/config.go`:

```env
# Master Toggle
RATE_LIMIT_ENABLED=true

# Failure Policies
RATE_LIMIT_FAIL_OPEN=true           # General and tenant business APIs fail open if Redis is down
RATE_LIMIT_AUTH_FAIL_OPEN=false     # Security/Auth endpoints fail closed to prevent brute-force attacks during Redis outages

# Tiers & Limits
RATE_LIMIT_GENERAL_REQUESTS=100
RATE_LIMIT_GENERAL_WINDOW=60s
RATE_LIMIT_AUTH_REQUESTS=20
RATE_LIMIT_AUTH_WINDOW=60s
RATE_LIMIT_SECURITY_REQUESTS=5
RATE_LIMIT_SECURITY_WINDOW=60s
RATE_LIMIT_TENANT_REQUESTS=300
RATE_LIMIT_TENANT_WINDOW=60s
RATE_LIMIT_PLATFORM_REQUESTS=300
RATE_LIMIT_PLATFORM_WINDOW=60s
```

## 6. HTTP Response Headers & Status Codes

On every rate-limited endpoint, standard informational headers are returned:
- `X-RateLimit-Limit`: Maximum requests permitted in the window.
- `X-RateLimit-Remaining`: Remaining requests before exhaustion.
- `X-RateLimit-Reset`: Unix timestamp when the current window resets.

When the rate limit is exceeded:
- **HTTP Status**: `429 Too Many Requests`
- **Header**: `Retry-After: <seconds>`
- **Response Body**:
  ```json
  {
    "success": false,
    "message": "rate limit exceeded, please retry after 45 seconds",
    "error": {
      "code": "TOO_MANY_REQUESTS",
      "message": "rate limit exceeded, please retry after 45 seconds"
    }
  }
  ```

## 7. Redis Failure Behavior

1. **General / Business APIs** (`RATE_LIMIT_FAIL_OPEN=true`):
   If Redis is down or unreachable, the request is permitted to avoid business disruption. A warning is logged and Prometheus metric `billbook_rate_limit_redis_errors_total{tier="general"}` is incremented.
2. **Security & Authentication APIs** (`RATE_LIMIT_AUTH_FAIL_OPEN=false`):
   If Redis is unreachable, security-sensitive endpoints (login, password reset, OTP) fail closed by returning `503 Service Unavailable` with `SERVICE_UNAVAILABLE` error code. This eliminates the vulnerability where attackers take down Redis to brute force credentials.

## 8. Observability

### Prometheus Metrics
- `billbook_rate_limit_requests_total{tier, status}`: Total count of evaluated requests by tier and outcome (`allowed`, `rejected`).
- `billbook_rate_limit_rejected_total{tier, route}`: Total count of rejected requests with route breakdown (low cardinality route patterns, e.g. `/api/v1/auth/tenant/login`).
- `billbook_rate_limit_redis_errors_total{tier}`: Counter of Redis failures during rate limiting evaluations.

### Structured Logging
Rate limit violations emit structured warning logs:
```text
level=WARN msg="rate limit exceeded" tier=auth route=/api/v1/auth/tenant/login method=POST retry_after_seconds=45
```
Raw client PII and credentials are never logged.

## 9. Local Verification

To run tests locally:
```bash
# Run Redis unit tests against local Redis
go test -v ./internal/repository/redis/... -run TestRateLimitRedis

# Run Service tests
go test -v ./internal/service/... -run TestRateLimiterService

# Run Middleware tests
go test -v ./internal/middleware/... -run TestRateLimiterMiddleware

# Run full HTTP pipeline integration tests
go test -v ./tests/integration/... -run TestRateLimitHTTPPipelineIntegration
```
