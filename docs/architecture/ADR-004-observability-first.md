# ADR-004: Observability-First Architecture

## Status
Accepted

## Context
Production incidents require rapid root-cause analysis without leaking sensitive credentials, PII, or high-cardinality label explosions into metrics backends.

## Decision
- Structured logging using Go's standard library `log/slog` (JSON in production, human-readable text in development).
- Context-driven attribute propagation: every incoming request is assigned or preserves `X-Request-ID`. Logs and traces attach `request_id`, `tenant_id`, and `user_id` automatically.
- Passwords, JWT tokens, and client secrets are strictly redacted and never logged.
- Prometheus metrics (`GET /metrics`) track HTTP request rates, status codes, latency histograms, in-flight requests, and PostgreSQL pool stats (`acquired`, `idle`, `max`, `empty_acquires`).
- Label cardinality is strictly guarded: only registered Gin route patterns (e.g. `/api/v1/customers/:id`) are used as metric labels; raw URL paths with UUIDs are rejected.
- Grafana dashboards are provisioned via code (`deployments/grafana/`).

## Consequences
- **Positive**: Complete observability from day 1, reproducible debugging via request IDs, pre-configured Grafana monitoring out-of-the-box.
- **Negative**: Developers must remember to pass `context.Context` through all service and repository calls.
