# ADR-001: Modular Monolith Architecture

## Status
Accepted

## Context
The Vyavsa platform is a multi-tenant business and accounting management SaaS for small and local businesses. The system requires high scalability, strict tenant isolation, and clear domain boundaries, but premature microservices would introduce excessive operational overhead, network latency, distributed transaction complexity, and deployment friction.

## Decision
We adopt a **Modular Monolith** architecture with Clean Architecture principles:
- Single deployable binary (`cmd/api`) containing bounded contexts within `internal/domain`.
- Clear layering: `HTTP Handler -> Service -> Repository Interface -> Postgres/Redis Implementation`.
- Zero circular dependencies.
- Strict isolation of business logic from transport (Gin) and database drivers (pgx).
- Modules communicate via in-process Go interfaces, enabling individual extraction into independent microservices in the future if scale warrants it.

## Consequences
- **Positive**: Rapid feature development, atomic transactions across business domains without distributed 2PC, simplified local development and CI/CD, single container deployment.
- **Negative**: Must strictly enforce boundary discipline in code reviews to prevent leaking cross-module state or raw SQL across domains.
