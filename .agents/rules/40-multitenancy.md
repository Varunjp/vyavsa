---
trigger: model_decision
description: Use when modifying tenant-scoped features, tenant data, users, employees, transactions, billing, reports, or any resource belonging to a business.
---

# Multi-Tenancy

Vyavsa is multi-tenant.

Tenant isolation is a security boundary.

## Rules

- Determine tenant identity from trusted authentication context.
- Do not trust tenant IDs supplied by clients.
- Every tenant-owned database query must be tenant-scoped.
- Every create operation must associate the resource with the authenticated tenant.
- Every update/delete operation must verify tenant ownership.
- Never return another tenant's records.
- Never use an unscoped lookup for tenant-owned resources.

Prefer:

tenant_id + resource_id

over:

resource_id only

when accessing tenant-owned data.

## Testing

For tenant-scoped features, include cross-tenant authorization tests when practical.

A request from tenant A must never access tenant B's data.