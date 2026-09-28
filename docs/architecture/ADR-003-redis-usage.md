# ADR-003: Redis Usage and Graceful Cache Degradation

## Status
Accepted

## Context
The application needs temporary state storage for JWT revocation/sessions, distributed rate-limiting, and cache-aside for high-read master data (e.g. platform plans). However, transient Redis outages should not cause total platform paralysis if the core database is healthy.

## Decision
- Redis 7 is used as an ephemeral cache and coordination store.
- **Authoritative financial records are never stored solely in Redis.** PostgreSQL remains the authority.
- The Redis client is designed for **graceful degradation**: if Redis is temporarily unreachable during startup or operation, the application logs a warning and falls back to database-backed operations rather than crashing immediately.
- Explicit TTLs and targeted cache invalidation keys (`tenant:{tenant_id}:...`).

## Consequences
- **Positive**: Low read latency, distributed rate limiting capability across horizontal API instances, resilient to temporary Redis restarts.
- **Negative**: Caches must be explicitly invalidated upon mutation to avoid stale views.
