---
trigger: model_decision
description: Use when modifying PostgreSQL schema, migrations, GORM models, repositories, queries, indexes, constraints, or transactions.
---

# Database

- PostgreSQL is the source database.
- Inspect the current schema and migrations before making changes.
- Never assume the schema from memory.
- Follow existing migration naming and structure.
- Never edit an already-applied migration unless the project explicitly requires it.
- New schema changes require a new migration.
- Preserve foreign keys, unique constraints, indexes, and tenant isolation.
- Prefer database constraints for invariants that must always hold.
- Use transactions for multi-step operations that must be atomic.
- Avoid N+1 queries.
- Select only required data when appropriate.
- Use parameterized queries or existing ORM mechanisms.
- Do not log credentials, tokens, OTPs, passwords, or sensitive payloads.

Before completing database work:

- Verify migration direction.
- Verify rollback/down migration when the project uses down migrations.
- Verify affected queries.
- Run relevant tests.