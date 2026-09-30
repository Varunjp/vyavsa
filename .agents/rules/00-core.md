---
trigger: always_on
---

---
trigger: always_on
---

# Vyavsa Core Engineering Rules

You are working on Vyavsa, a production-oriented multi-tenant SaaS application for small businesses.

## Source of truth

- Inspect the existing repository before changing code.
- Existing implementation and project conventions take priority over assumptions.
- Reuse existing patterns, utilities, services, middleware, repositories, configuration, and API conventions.
- Do not silently introduce architectural changes.

## Scope

- Make the smallest change that correctly solves the task.
- Do not modify unrelated files.
- Do not refactor unrelated code while implementing a feature.
- Avoid adding dependencies unless clearly necessary.
- Preserve backward compatibility unless the task explicitly changes the contract.

## Architecture

Follow the existing flow:

router
→ middleware
→ handler
→ validation
→ service
→ repository
→ database

Do not bypass established layers without a strong reason.

## Go

- Write idiomatic Go.
- Keep handlers thin.
- Keep business logic in services.
- Keep database access in repositories.
- Reuse existing error handling.
- Propagate context.
- Handle errors explicitly.
- Avoid unnecessary abstractions.

## Completion

Before finishing:

1. Format changed Go files.
2. Run relevant tests.
3. Run relevant static checks when practical.
4. Inspect the final diff.
5. Fix failures caused by the change.

Do not claim a check passed unless it was actually run.

## Communication

Be concise.

Final response should contain only:

- Changes made
- Tests/checks run
- Important notes or remaining issues

Do not provide lengthy explanations unless requested.