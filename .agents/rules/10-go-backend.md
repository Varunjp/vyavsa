---
trigger: model_decision
description: Use when modifying Go backend code, handlers, services, repositories, middleware, configuration, or business logic.
---

# Go Backend

- Inspect similar existing code before implementing.
- Follow the project's existing package structure.
- Prefer existing interfaces and implementations.
- Keep HTTP handlers responsible for HTTP concerns only.
- Keep business rules in services.
- Keep persistence in repositories.
- Pass context through service and repository calls.
- Return errors instead of hiding failures.
- Wrap errors using the project's existing convention.
- Avoid global mutable state.
- Avoid unnecessary goroutines.
- Protect shared state from races.
- Reuse existing logging and configuration utilities.
- Do not create duplicate helper functions when an equivalent already exists.

For new code:

1. Find the closest existing implementation.
2. Follow its structure.
3. Implement only the required behavior.
4. Add focused tests.
5. Run gofmt and relevant tests.