---
description: Prepare the current branch for a production-quality pull request
---

# Prepare PR

Prepare the current branch for a production-quality pull request.

## Process

1. Inspect git status.
2. Inspect the complete diff.
3. Identify all files changed by the feature.
4. Check for accidental changes.
5. Run gofmt on changed Go files.
6. Run relevant tests.
7. Run relevant static checks.
8. Check for secrets or credentials.
9. Check API/database compatibility.
10. Check tenant isolation and authorization.
11. Summarize the implementation.

## Final response

Generate:

### PR Title

A concise conventional title.

### Summary

2-5 bullets describing the actual implementation.

### Technical Changes

Grouped by backend/database/frontend/auth/etc.

### Testing

Actual commands/checks executed.

### Security

Security-relevant changes and verification.

### Migration / Deployment

Only if applicable.

### Notes

Only important remaining considerations.