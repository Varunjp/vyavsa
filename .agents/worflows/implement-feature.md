# Implement Feature

Implement the requested Vyavsa feature end-to-end.

## Process

1. Inspect repository structure.
2. Identify the existing implementation closest to the requested feature.
3. Trace the relevant flow:
   router → handler → service → repository → database
4. Identify required frontend/API/database changes.
5. Implement the smallest complete solution.
6. Reuse existing utilities and patterns.
7. Add focused tests.
8. Run gofmt on changed Go files.
9. Run relevant tests.
10. Review the final diff.
11. Fix issues caused by the implementation.

## Constraints

- Do not refactor unrelated code.
- Do not introduce unnecessary dependencies.
- Do not silently change existing API contracts.
- Preserve authentication and tenant isolation.
- Do not claim tests passed unless they were executed.

## Final response

Return:

### Changes
- ...

### Tests
- ...

### Notes
- ...