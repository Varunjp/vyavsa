# Fix Bug

Fix the reported Vyavsa bug.

## Process

1. Reproduce or locate the failure.
2. Inspect the relevant code path.
3. Identify the root cause.
4. Make the smallest correct fix.
5. Add a regression test when appropriate.
6. Run the focused test.
7. Run broader relevant tests.
8. Run gofmt on changed Go files.
9. Review the diff.

## Constraints

- Fix the root cause.
- Do not hide the failure.
- Do not rewrite unrelated code.
- Do not weaken tests.

## Final response

Return:

- Root cause
- Fix
- Tests run
- Remaining issues