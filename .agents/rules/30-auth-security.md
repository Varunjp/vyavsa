---
trigger: model_decision
description: Use when modifying authentication, authorization, JWT, sessions, cookies, OTP, password recovery, passwords, RBAC, login, registration, or security-sensitive code.
---

# Authentication and Security

Security-sensitive code must preserve the existing authentication architecture.

## Authentication

- Inspect the existing auth implementation first.
- Reuse existing token, password, session, Redis, mailer, middleware, and error utilities.
- Do not create a second authentication mechanism unnecessarily.
- Never store plaintext passwords.
- Never log passwords, access tokens, refresh tokens, OTPs, reset tokens, or secrets.

## Authorization

- Authentication and authorization are separate concerns.
- Verify authenticated identity before protected operations.
- Verify the user's role before privileged operations.
- Never trust tenant_id, user_id, role, or ownership identifiers supplied directly by the client.
- Prevent IDOR and privilege escalation.

## Multi-tenant security

Every tenant-scoped operation must verify tenant ownership from trusted authentication context.

Never allow:

request tenant_id
→ direct database access

without authorization/ownership validation.

## JWT / sessions

- Reuse existing JWT and refresh-token implementation.
- Preserve existing expiry and rotation behavior.
- Keep sensitive tokens in the project's established secure storage.
- Do not expose tokens unnecessarily to frontend JavaScript.
- Preserve existing cookie security settings.

## OTP

When modifying OTP functionality:

- Preserve expiration.
- Preserve attempt limits.
- Preserve resend/rate limits.
- Prevent OTP reuse.
- Avoid account enumeration.
- Never log OTP values.
- Invalidate OTP after successful verification.

## Password reset

Password recovery must remain:

forgot password
→ OTP
→ verification
→ short-lived reset authorization
→ password reset
→ invalidate relevant existing authentication state

Do not allow a verified OTP to become a reusable password-reset credential.

## Validation

Validate authentication inputs at the API boundary.

Do not reveal whether an account exists when the existing API uses anti-enumeration behavior.