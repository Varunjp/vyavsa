# Production Email Delivery with Resend

Vyavsa uses **Resend** as its production transactional email delivery provider, while supporting **Gmail SMTP** and simulated **log** providers for local development and test automation.

This document details the architecture, provider selection rules, and step-by-step instructions to configure, verify, and operate Resend in production.

---

## Architecture Overview

All transactional emails (such as Password Reset OTPs and account notifications) flow through a unified architecture that isolates business logic and email templates from the underlying transport mechanism:

```text
API Handler (e.g. POST /api/v1/auth/forgot-password)
    │
    ▼
Auth Service (Validates rate limits, user status, generates OTP)
    │
    ▼
Async Mailer (internal/mailer.AsyncMailer)
    │
    ▼
Background Worker (internal/worker.EmailWorker with Redis Queue / in-memory buffer)
    │
    ▼
Email Service (internal/mailer.EmailService - formats branded HTML & text templates)
    │
    ▼
EmailSender Interface (internal/mailer.EmailSender)
    │
    ├─► ResendSender (Production - official github.com/resend/resend-go/v2 SDK)
    ├─► SMTPSender   (Development - standard net/smtp with STARTTLS/TLS)
    └─► LogSender    (Test/Local - simulated delivery via structured logs)
```

---

## Environment Configuration & Selection

Email provider selection occurs centrally during application startup in `internal/config`:

| Environment Variable | Description | Example (Development) | Example (Production) |
| :--- | :--- | :--- | :--- |
| `APP_ENV` | Application environment | `development` | `production` |
| `EMAIL_PROVIDER` | Selected transport provider (`resend`, `smtp`, `log`) | `smtp` | `resend` |
| `EMAIL_FROM` | Sender identity (RFC 5322 or plain email) | `no-reply@vyavsa.com` | `Vyavsa <no-reply@yourdomain.com>` |
| `EMAIL_FROM_NAME` | Sender display name | `Vyavsa Support` | `Vyavsa Support` |
| `RESEND_API_KEY` | Resend API key (required when `EMAIL_PROVIDER=resend`) | *(not required)* | `re_1234567890abcdef...` |
| `SMTP_HOST` | SMTP server host (required when `EMAIL_PROVIDER=smtp`) | `smtp.gmail.com` | *(not required)* |
| `SMTP_PORT` | SMTP port (587 for STARTTLS, 465 for TLS) | `587` | *(not required)* |
| `SMTP_USERNAME` | SMTP username | `user@gmail.com` | *(not required)* |
| `SMTP_PASSWORD` | SMTP password / Google App Password | `abcd efgh ijkl mnop` | *(not required)* |

### Fail-Safe Validation Rules

1. **Production Safety**:
   - In `APP_ENV=production`, `EMAIL_PROVIDER` **must** be explicitly configured to `resend`.
   - The application **refuses to start** if `EMAIL_PROVIDER` is missing, set to `smtp`, or set to `log` in production.
   - The application will **never** silently fall back to development SMTP in production.
2. **Missing Credentials**:
   - When `EMAIL_PROVIDER=resend`, `RESEND_API_KEY` is mandatory.
   - When `EMAIL_PROVIDER=smtp`, `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, and `SMTP_PASSWORD` are mandatory.
3. **Secret Redaction**:
   - API keys matching `re_*` and passwords are automatically redacted from all log entries and error messages.
4. **Permanent Failure Fast-Abort**:
   - Provider errors indicating invalid recipients or authentication errors (HTTP 400, 401, 403, 422) are classified as permanent, stopping futile worker retries immediately to avoid burning rate limits.
   - Transient errors (HTTP 429 rate limits, network timeouts, HTTP 5xx) are retried with exponential backoff up to the configured retry count.

---

## Step-by-Step Production Resend Setup Guide

Follow these steps to establish production email delivery with your custom verified domain:

### Step 1: Create a Resend Account
1. Visit [https://resend.com](https://resend.com) and create an account or sign in.
2. Complete your organization profile.

### Step 2: Add and Verify Your Custom Domain
1. In the Resend Dashboard, navigate to **Domains** and click **Add Domain**.
2. Enter your production sending domain (e.g., `vyavsa.com` or `mail.vyavsa.com`).
3. Select your primary region (e.g., `us-east-1` or `eu-west-1`).

### Step 3: Configure Required DNS Records
Resend provides DNS records for authentication and deliverability. Add these to your DNS provider (Cloudflare, AWS Route 53, Namecheap, etc.):

| Type | Name / Host | Value / Target | Purpose |
| :--- | :--- | :--- | :--- |
| **TXT** | `resend._domainkey` | `k=rsa; p=...` | DKIM authentication |
| **TXT** | `@` (or subdomain) | `v=spf1 include:amazonses.com ~all` | SPF authorization |
| **MX** | `feedback` (or subdomain) | `feedback-smtp.us-east-1.amazonses.com` (Priority 10) | Bounce & feedback routing |

*Note: Allow up to 24–48 hours for DNS propagation, though verification usually takes a few minutes. Check status in the Resend dashboard until status shows **Verified**.*

### Step 4: Generate API Key
1. In the Resend Dashboard, navigate to **API Keys**.
2. Click **Create API Key**.
3. Name the key (e.g., `vyavsa-production-api`).
4. Set permissions to **Sending access** and restrict to your verified domain.
5. Copy the generated key (`re_...`) immediately and store it securely in your secret manager (e.g., AWS Secrets Manager, Doppler, or HashiCorp Vault).

### Step 5: Configure Sender Address
Ensure your configured sender address in `EMAIL_FROM` matches your verified domain:
```env
EMAIL_FROM="Vyavsa Support <no-reply@vyavsa.com>"
```
*Sending from an unverified domain will result in an immediate HTTP 422 Unprocessable Entity error.*

### Step 6: Deploy Production Environment Variables
Set the following environment variables in your production environment (ECS, Kubernetes, EC2, or Docker Compose):

```bash
APP_ENV=production
EMAIL_PROVIDER=resend
RESEND_API_KEY=re_your_verified_api_key_here
EMAIL_FROM=Vyavsa Support <no-reply@yourverifieddomain.com>
EMAIL_FROM_NAME="Vyavsa Support"
```

### Step 7: Verify Production Delivery
1. Start the Vyavsa backend service. Check application startup logs:
   ```text
   {"level":"INFO","msg":"initializing Resend email provider","from":"no-reply@yourverifieddomain.com","from_name":"Vyavsa Support"}
   ```
2. Trigger a password recovery request:
   ```bash
   curl -X POST https://api.yourdomain.com/api/v1/auth/forgot-password \
     -H "Content-Type: application/json" \
     -d '{"email": "verified-user@yourdomain.com"}'
   ```
3. Check application logs for delivery confirmation with the Resend message ID:
   ```text
   {"level":"INFO","msg":"email sent successfully via Resend","to":"verified-user@yourdomain.com","subject":"Reset your Vyavsa password","resend_message_id":"49a3999c-0ce1-4ea6-ab68-afcd6dc2e794"}
   ```
4. Verify email receipt in the recipient's inbox and inspect delivery status in the Resend dashboard under **Emails**.

---

## Observability & Metrics

Resend email delivery is monitored via Prometheus metrics exposed at `/metrics`:

- `billbook_email_delivery_total{provider="resend", status="success"}`: Counter for successful deliveries.
- `billbook_email_delivery_total{provider="resend", status="failure"}`: Counter for failed delivery attempts.
- `billbook_email_delivery_duration_seconds{provider="resend"}`: Latency histogram for email API calls.
- `billbook_email_worker_jobs_total`: Total background worker email jobs processed.
- `billbook_email_worker_retry_total`: Total retries executed by the background worker.
- `billbook_email_worker_failures_total`: Total terminal email delivery failures.
