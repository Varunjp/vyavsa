# Vyavsa - Small Business Bill Book SaaS

Vyavsa is a production-grade, multi-tenant business and accounting management SaaS backend engineered in Go. It empowers small and local businesses with invoicing, counter sales, purchases, expenses, employee attendance, payroll, and real-time ledger accounting.

---

## 🏛️ Architecture Overview

The system is designed as a **Modular Monolith** adhering to Clean Architecture principles and domain-driven design boundaries:

```text
HTTP Request
     ↓
[Middleware Pipeline] (Request ID, Security Headers, CORS, Recovery, slog Request Logger)
     ↓
[HTTP Handlers]       (internal/handler) - Request parsing, validation, tenant context
     ↓
[Services]            (internal/service) - Pure business rules, domain transactions
     ↓
[Repositories]        (internal/repository) - Persistence contracts
     ↓
[PostgreSQL / Redis]  (internal/database, internal/cache) - pgxpool & go-redis
```

### Key Architectural Tenets
- **PostgreSQL 16** is the single source of authoritative truth with strict `NUMERIC(14,2)` monetary math and `TIMESTAMPTZ` audit timestamps.
- **Tenant Isolation**: Every tenant-owned resource enforces `WHERE tenant_id = $1` at the persistence layer, verified via JWT identity.
- **Redis 7**: High-performance caching, token revocation, and rate limiting with graceful degradation.
- **Observability**: Native Prometheus metrics (`GET /metrics`), structured `log/slog` logging with request IDs, and provisioned Grafana dashboards.
- **Graceful Lifecycle**: Zero-downtime shutdown handling for `SIGTERM` and `SIGINT` signals.

---

## 🚀 Quick Start (Docker Compose)

The entire platform—including PostgreSQL, Redis, API, Prometheus, and Grafana—can be launched with a single command:

```bash
docker compose up --build -d
```

### Verify Service Health

Once launched, verify all services are running and healthy:

```bash
docker compose ps
```

Expected output:
```text
NAME                IMAGE                    STATUS                   PORTS
vyavsa_api          vyavsa-api               Up (healthy)             0.0.0.0:8080->8080/tcp
vyavsa_grafana      grafana/grafana:10.4.0   Up                       0.0.0.0:3000->3000/tcp
vyavsa_postgres     postgres:16-alpine       Up (healthy)             0.0.0.0:5432->5432/tcp
vyavsa_prometheus   prom/prometheus:v2.51.0  Up                       0.0.0.0:9090->9090/tcp
vyavsa_redis        redis:7-alpine           Up (healthy)             0.0.0.0:6379->6379/tcp
```

### Smoke Test Endpoints

```bash
# 1. Process Liveness Check (Independent of database)
curl -i http://localhost:8080/health

# 2. Dependency Readiness Check (Verifies PostgreSQL & Redis)
curl -i http://localhost:8080/ready

# 3. Prometheus Metrics Scraping
curl -i http://localhost:8080/metrics

# 4. API v1 Ping
curl -i http://localhost:8080/api/v1/ping
```

---

## 📊 Observability & Monitoring

| Service | URL | Credentials / Notes |
| :--- | :--- | :--- |
| **API Server** | [http://localhost:8080](http://localhost:8080) | Versioned at `/api/v1` |
| **Prometheus** | [http://localhost:9090](http://localhost:9090) | Inspect scrape targets at `/targets` |
| **Grafana** | [http://localhost:3000](http://localhost:3000) | `admin` / `admin` (Provisioned dashboard: **Vyavsa API Overview**) |

---

## 🛠️ Local Development (Without Docker)

### Prerequisites
- Go 1.24+ (Go 1.26 supported)
- Running PostgreSQL 16 instance (`billbook` database)
- Running Redis 7 instance

### Run Local API Server

```bash
# Copy and configure environment variables
cp .env.example .env

# Run database migrations
make migrate-up

# Start local server
make run
```

---

## 🧪 Testing

```bash
# Run all unit and integration tests
make test

# Run tests with Go Race Detector
make test-race

# Run linter and formatting
make lint
make fmt
```

---

## 📦 Database Migrations

Migrations are written in pure SQL in `migrations/`:
- `000001_initial_schema.up.sql`
- `000001_initial_schema.down.sql`

When running the application, automated migrations run on startup by default (`DATABASE_AUTO_MIGRATE=true`). You can also execute migrations via CLI:

```bash
# Apply pending migrations
make migrate-up

# Roll back the latest migration
make migrate-down
```

---

## 📧 Transactional Email Delivery

Vyavsa routes transactional emails (password reset OTPs, notifications) through a provider-agnostic interface:
- **Development**: Gmail SMTP (`EMAIL_PROVIDER=smtp`) or simulated logger (`EMAIL_PROVIDER=log`).
- **Production**: Resend (`EMAIL_PROVIDER=resend`) with domain verification, fail-safe startup checks, and Prometheus metrics tracking.

Detailed configuration and domain verification guide: [Production Email Delivery with Resend](docs/email-delivery-resend.md).

---

## 📖 Architecture Decision Records (ADRs)
- [ADR-001: Modular Monolith Architecture](docs/architecture/ADR-001-modular-monolith.md)
- [ADR-002: PostgreSQL as Single Source of Truth](docs/architecture/ADR-002-postgresql-source-of-truth.md)
- [ADR-003: Redis Usage & Graceful Degradation](docs/architecture/ADR-003-redis-usage.md)
- [ADR-004: Observability-First Architecture](docs/architecture/ADR-004-observability-first.md)
- [ADR-005: Schema Refinements & Multi-Tenancy](docs/architecture/ADR-005-schema-and-multi-tenancy.md)

