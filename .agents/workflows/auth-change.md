---
description: Implement an authentication/security change safely
---

# Authentication Change

Implement an authentication/security change safely.

## Process

1. Inspect the complete existing authentication flow.
2. Identify affected middleware, handlers, services, repositories, Redis, tokens, cookies, and frontend code.
3. Preserve existing authentication architecture.
4. Implement the smallest required change.
5. Verify authorization boundaries.
6. Verify tenant isolation.
7. Add security-focused tests.
8. Test expiry/replay/invalid credentials where applicable.
9. Run formatting and relevant tests.
10. Review the final diff for security issues.

## Never

- Log credentials.
- Store plaintext passwords.
- Trust client tenant IDs for authorization.
- Expose internal authentication errors.
- Create duplicate authentication mechanisms unnecessarily.