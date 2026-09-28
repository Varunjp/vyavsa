# ADR-005: Schema Refinements and Multi-Tenant Isolation Strategy

## Status
Accepted

## Context
Multi-tenancy requires strict mathematical boundaries ensuring one tenant's business data is never leaked or visible to another. The provided initial schema contained mixed casing, missing foreign keys, missing cascade/restrict policies, floating-point ambiguity, and unindexed tenant queries.

## Decision
1. **Naming Standard**: Consistent `snake_case` in PostgreSQL and idiomatic CamelCase/PascalCase in Go.
2. **Foreign Key Integrity**:
   - Tenant-scoped records reference `tenants(id) ON DELETE CASCADE`.
   - Critical lookup references (e.g. `line_sale.customer_id`, `platform_subscriptions.current_plan_id`) use `ON DELETE RESTRICT` to protect against orphan records and accidental ledger deletions.
   - Bank linkages use `ON DELETE SET NULL` to preserve historical sales if a bank account is closed.
3. **Compound Tenant Indexing**:
   - Every tenant table has compound indexes starting with `tenant_id` to guarantee fast index-scans for tenant queries (e.g. `(tenant_id, created_at DESC)`, `(tenant_id, customer_id)`, `(tenant_id, date)`).
4. **Data Integrity Constraints**:
   - Check constraints enforce status enums (e.g. `active`, `inactive`, `suspended`) and prevent negative values for monetary and count fields (`total_amount >= 0`, `quantity > 0`).
   - Unique constraints prevent duplicate emails per tenant `(tenant_id, email)` and multiple daily stats for the same date `(tenant_id, date)`.
5. **Enforcement Principle**:
   - Handlers extract `tenant_id` exclusively from authenticated session/JWT tokens.
   - Handlers never trust a `tenant_id` provided in request query or JSON payload.
   - All repository queries must enforce `WHERE tenant_id = $1`.

## Consequences
- **Positive**: Strict data isolation, guaranteed referential integrity, zero data leakage across tenants, optimized database performance.
- **Negative**: Foreign key cascades and checks require disciplined deletion workflows (soft deletes/deactivations preferred for business entities).
