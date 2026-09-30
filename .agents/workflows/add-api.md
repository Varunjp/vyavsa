---
description: Implement a new Vyavsa API endpoint
---

# Add API

Implement a new Vyavsa API endpoint.

## Process

1. Inspect similar existing endpoints.
2. Define the request/response contract.
3. Add route.
4. Add handler.
5. Add validation.
6. Add service logic.
7. Add repository/database changes if required.
8. Add authorization and tenant checks.
9. Add tests.
10. Run formatting and relevant tests.
11. Review the API diff.

## Requirements

- Follow existing API response/error conventions.
- Validate client input.
- Never trust client-supplied tenant ownership.
- Preserve existing authentication.
- Do not expose internal errors.