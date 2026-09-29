# ADR-006: Secure OTP-Based Password Recovery Flow

## Status
Accepted

## Context
Users belonging to tenants and platform administrators need a secure, reliable way to recover their accounts if passwords are forgotten. The mechanism must strictly prevent:
1. Account enumeration (revealing whether an email is registered).
2. OTP brute forcing and replay attacks.
3. Cross-tenant privilege escalation or cross-account resets.
4. Continuing to use stale sessions or tokens after a credential reset.
5. Plaintext secret leaks in logs, telemetry, or storage.

## Decision

### 1. API Contract & Anti-Enumeration
Endpoints are exposed under `/api/v1/auth` (with aliases under `/auth`):
- `POST /api/v1/auth/forgot-password`: Accepts `{"email": "..."}`. Always returns `{"success": true, "message": "If an account exists with this email, an OTP has been sent."}` with HTTP 200 regardless of whether the account exists or is active.
- `POST /api/v1/auth/verify-reset-otp`: Accepts `{"email": "...", "otp": "..."}`. Returns `{"success": true, "data": {"reset_token": "..."}}`.
- `POST /api/v1/auth/reset-password`: Accepts `{"reset_token": "...", "new_password": "..."}`. Returns `{"success": true, "message": "Password reset successfully."}`.

### 2. Cryptographic OTP Generation & Redis Storage
- OTPs are 6-digit numeric strings generated using `crypto/rand.Int`.
- Plaintext OTPs are never persisted. Only the SHA-256 cryptographic hash (`otp_hash`) is stored in Redis at `password_reset:otp:<normalized_email>` with a 5-minute TTL.
- Comparison between submitted OTPs and the stored hash uses constant-time string comparison (`subtle.ConstantTimeCompare`) to mitigate timing attacks.
- Verification is capped at 5 attempts. Reaching 5 attempts deletes the OTP immediately.
- A successful verification deletes the OTP immediately (single-use).

### 3. Dedicated Password Reset Tokens
- After OTP verification, the system issues a short-lived (10-minute) dedicated reset JWT with claims:
  `{"sub": "<user_id>", "tenant_id": "<tenant_id>", "purpose": "password_reset", "jti": "<token_id>"}`.
- Standard API authentication middleware rejects any token containing `purpose != ""`, ensuring reset tokens cannot be used to call API endpoints.
- Single-use consumption is enforced by tracking the `token_id` in Redis at `password_reset:token:<token_id>`. Upon password reset, the key is deleted immediately.

### 4. Authentication Invalidation
- When a password is reset, `user:revoked_before:<user_id>` is recorded in Redis with a 7-day TTL (equal to the maximum refresh token lifetime).
- `middleware.Authenticate` and `AuthService.RefreshToken` check whether a token's `issued_at` timestamp is before `revoked_before`. Any pre-existing access tokens or refresh tokens are immediately rejected with HTTP 401.

### 5. Multi-Tenant Isolation
- Tenant user password updates are executed with explicit tenant constraints:
  `UPDATE tenant_user SET password_hash = $1 WHERE tenant_id = $2 AND id = $3`.
- User identities and tenant IDs are bound to cryptographic claims and verified against the repository, preventing cross-tenant password resets.

### 6. Rate Limiting & Cooldowns
- Max 3 requests per email per 15 minutes (`password_reset:rate_limit:email:<email>`).
- Max 10 requests per IP per 15 minutes (`password_reset:rate_limit:ip:<ip>`).
- 60-second cooldown per email between OTP requests (`password_reset:cooldown:<email>`).
- Exceeded rate limits return HTTP 429 `TOO_MANY_REQUESTS`.

### 7. Observability & Zero-Secret Logging
- Prometheus metrics track requests, dispatched emails, successful/failed verifications, and rate limits without exposing PII or secret values in labels.
- Structured logs omit OTPs, passwords, and reset tokens.

## Consequences
- **Positive**: Hardened against enumeration, brute force, replay, and session hijacking; strict multi-tenant boundaries; zero database migrations needed by leveraging Redis state and existing tables.
- **Negative**: Redis dependency required for recovery state (with graceful degraded handling in tests).
