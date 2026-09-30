---
trigger: model_decision
description: Use when modifying Docker, Docker Compose, AWS, EC2, Nginx, TLS, CI/CD, environment configuration, deployment scripts, or production infrastructure.
---

# Deployment

- Inspect existing deployment configuration before changing it.
- Do not hardcode secrets.
- Use environment variables or the existing secret-management mechanism.
- Preserve existing service names, ports, networks, volumes, and deployment conventions unless the task requires changes.
- Verify Docker configuration before changing application code to compensate for infrastructure problems.
- Do not expose internal services publicly without explicit requirement.
- Preserve TLS configuration.
- Do not commit private keys, certificates, credentials, or `.env` secrets.
- Avoid destructive commands in production.
- Never run destructive database/storage commands unless explicitly requested.

For deployment changes:

1. Inspect current configuration.
2. Identify affected services.
3. Make the smallest required change.
4. Validate configuration.
5. Check application compatibility.
6. Document any required environment/config changes.