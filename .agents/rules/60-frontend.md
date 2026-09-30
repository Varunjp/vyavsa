---
trigger: model_decision
description: Use when modifying HTML templates, CSS, JavaScript, authentication pages, dashboard UI, forms, or frontend API integration.
---

# Frontend

- Inspect existing templates and JavaScript before creating new code.
- Reuse existing components, styles, API helpers, and validation patterns.
- Use Go HTML templates where already established.
- Do not introduce Tailwind unless explicitly requested.
- Keep UI modern, minimal, responsive, and consistent with existing pages.
- Do not duplicate API/authentication JavaScript utilities.
- Handle loading, success, and error states.
- Never expose secrets or private tokens unnecessarily in browser JavaScript.
- Use the existing authentication/cookie strategy.
- Keep frontend changes limited to the requested feature.

For forms:

- Validate obvious input errors on the client when appropriate.
- Always perform authoritative validation on the server.
- Display useful user-facing errors without exposing internal details.