---
trigger: model_decision
description: Use when modifying logging, metrics, tracing, monitoring, health checks, background jobs, or production diagnostics.
---

# Observability

- Reuse existing logging and metrics infrastructure.
- Add metrics only when they provide useful operational information.
- Use stable metric names and labels.
- Avoid high-cardinality labels.
- Do not log secrets, passwords, OTPs, tokens, or sensitive personal/business data.
- Log failures with enough context to diagnose them.
- Preserve existing health/readiness conventions.
- Do not add noisy logs inside high-frequency paths without justification.