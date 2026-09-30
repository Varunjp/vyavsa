---
trigger: model_decision
description: Use when modifying HTTP routes, REST APIs, request/response structures, validation, middleware, API errors, or API documentation.
---

# API

- Inspect existing routes and API conventions before adding endpoints.
- Follow existing URL naming and HTTP method conventions.
- Reuse existing request validation.
- Reuse existing response and error formats.
- Do not introduce a second response format.
- Validate all client-controlled input.
- Return appropriate HTTP status codes.
- Do not expose internal errors, stack traces, SQL errors, secrets, or sensitive implementation details.

## Compatibility

Before changing an existing API:

- Find its callers.
- Check frontend usage.
- Check tests.
- Preserve compatibility unless the task explicitly requires a breaking change.

## New endpoint

For a new endpoint:

1. Define request/response contract.
2. Add route.
3. Add handler.
4. Add validation.
5. Call service layer.
6. Add authorization.
7. Add tests.
8. Update API documentation if the project uses it.