---
trigger: model_decision
description: Use when adding tests, debugging test failures, or modifying behavior that requires verification.
---

# Testing

Tests should verify behavior, not implementation details.

Prioritize:

1. Happy path.
2. Validation failures.
3. Authorization failures.
4. Tenant isolation.
5. Important edge cases.
6. Error handling.

For security-sensitive functionality also test:

- expired credentials
- invalid credentials
- replay attempts
- unauthorized access
- cross-tenant access
- rate/attempt limits when applicable

When fixing a bug:

- Reproduce the failure.
- Add or update a regression test.
- Implement the fix.
- Run the focused test.
- Run broader relevant tests afterward.

Do not weaken or remove a test merely to make the suite pass.