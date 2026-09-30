---
description: Implement a database/schema change safely
---

# Database Change

Implement a database/schema change safely.

## Process

1. Inspect current migrations.
2. Inspect affected models.
3. Inspect affected repositories/queries.
4. Determine whether a new migration is required.
5. Create the migration using existing naming conventions.
6. Add/update models if required.
7. Update repositories/services.
8. Add tests.
9. Verify constraints and tenant isolation.
10. Verify migration behavior.
11. Run relevant tests.

## Constraints

- Never silently modify an already-applied migration.
- Preserve existing constraints.
- Avoid destructive changes unless explicitly requested.
- Do not remove data without explicit requirement.