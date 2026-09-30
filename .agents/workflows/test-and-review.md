---
description: Review the current implementation before considering the task complete
---

# Test and Review

Review the current implementation before considering the task complete.

## Check

- Compilation
- gofmt
- relevant unit tests
- integration tests when available
- error handling
- authorization
- tenant isolation
- input validation
- concurrency safety
- database correctness
- API compatibility
- sensitive data exposure
- unnecessary changes

Then inspect the final git diff.

Fix problems caused by the current task.

Do not expand the scope into unrelated refactoring.

## Final response

Return only:

- Issues found
- Fixes made
- Tests/checks run
- Remaining issues