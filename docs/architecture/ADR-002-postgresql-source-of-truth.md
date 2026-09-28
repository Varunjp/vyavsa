# ADR-002: PostgreSQL as Single Source of Truth & Financial Consistency

## Status
Accepted

## Context
Accounting and small-business bill-books require absolute data consistency, ACID transactions, and deterministic calculations. Floating-point rounding errors and distributed eventual consistency are unacceptable for financial balances, taxes, and ledgers.

## Decision
- **PostgreSQL 16** is the single authoritative source of truth.
- All monetary amounts use `NUMERIC(14,2)` to prevent IEEE-754 floating-point inaccuracies.
- UUIDv4 primary keys (`gen_random_uuid()`) for all entities to prevent ID enumeration and ease offline generation.
- Timezone-aware timestamps (`TIMESTAMPTZ`) in UTC across all tables.
- Database transactions wrap all multi-step financial mutations (e.g. sale + payment + ledger update).
- Automated migrations manage schema evolution without `AutoMigrate()`.

## Consequences
- **Positive**: Complete auditability, atomic consistency, mathematical precision for financial computations.
- **Negative**: Schema changes require explicit up/down migration scripts.
