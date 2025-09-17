# Secrets Management for Docker Compose and CI

This document describes the secure handling of credentials for local/dev, CI, and production-like environments.

Goals:
- Remove hardcoded credentials from code and Compose
- Support both environment variables and file-based secrets (Docker/Kubernetes)
- Backward compatible local developer experience
- CI-safe defaults with no secret leakage in logs

## Summary of changes

Application code now supports the common “*_FILE” pattern for secrets:
- JWT secrets
  - Go: JWT_SECRET_FILE
  - Node (gateway, label): INTERNAL_JWT_SECRET_FILE
  - Python (pricing): INTERNAL_JWT_SECRET_FILE
- Database URL
  - All Go services + pricing-svc: DATABASE_URL_FILE
- Identity admin seed
  - identity-svc: ADMIN_PASSWORD_FILE

If a *_FILE env var is set and points to a readable file, the service reads and trims the content; otherwise it falls back to the corresponding env var, and finally to dev-safe defaults.

Additionally, docker-compose.yaml no longer hardcodes credentials. It uses env-expansion with defaults, e.g. `${POSTGRES_PASSWORD:-postgres}`. CI can override these securely.

## How to use in local development

By default, nothing changes for developers. The Compose stack uses defaults suitable for local use.

For improved local security without breaking changes, you can:
- Provide environment overrides via an `.env` file (ignored by git) to set values, e.g. `POSTGRES_PASSWORD=changeme`.
- Or use file-based secrets by setting *_FILE variables and mounting files using `docker compose -f docker-compose.yml -f docker-compose.override.yml up` with an override file that binds `/run/secrets/*` files.

Example docker-compose.override.yml snippet:

```yaml
services:
  identity-svc:
    environment:
      JWT_SECRET_FILE: /run/secrets/jwt_secret
      ADMIN_PASSWORD_FILE: /run/secrets/admin_password
    volumes:
      - ./ops/secrets/jwt_secret:/run/secrets/jwt_secret:ro
      - ./ops/secrets/admin_password:/run/secrets/admin_password:ro
  shipment-svc:
    environment:
      DATABASE_URL_FILE: /run/secrets/database_url
    volumes:
      - ./ops/secrets/database_url:/run/secrets/database_url:ro
  tracking-svc:
    environment:
      DATABASE_URL_FILE: /run/secrets/database_url
    volumes:
      - ./ops/secrets/database_url:/run/secrets/database_url:ro
  pricing-svc:
    environment:
      INTERNAL_JWT_SECRET_FILE: /run/secrets/internal_jwt_secret
      DATABASE_URL_FILE: /run/secrets/database_url
    volumes:
      - ./ops/secrets/internal_jwt_secret:/run/secrets/internal_jwt_secret:ro
      - ./ops/secrets/database_url:/run/secrets/database_url:ro
  gateway-api:
    environment:
      INTERNAL_JWT_SECRET_FILE: /run/secrets/internal_jwt_secret
    volumes:
      - ./ops/secrets/internal_jwt_secret:/run/secrets/internal_jwt_secret:ro
  label-svc:
    environment:
      INTERNAL_JWT_SECRET_FILE: /run/secrets/internal_jwt_secret
    volumes:
      - ./ops/secrets/internal_jwt_secret:/run/secrets/internal_jwt_secret:ro
```

Ensure the files in `ops/secrets/*` are chmod 600 and excluded from git (see .gitignore).

## How to use in CI

Recommended: use environment variables provided by GitHub Actions Secrets (masked) and avoid writing secret values to disk unless necessary.

- Set these repo/org secrets:
  - JWT_SECRET
  - INTERNAL_JWT_SECRET
  - ADMIN_PASSWORD
  - POSTGRES_PASSWORD (optional when using DATABASE_URL)
  - DATABASE_URL (preferred: a single URL consumed by all services)

- Update your workflow to export these as environment variables for the `docker compose up` step:

```yaml
env:
  JWT_SECRET: ${{ secrets.JWT_SECRET }}
  INTERNAL_JWT_SECRET: ${{ secrets.INTERNAL_JWT_SECRET }}
  ADMIN_PASSWORD: ${{ secrets.ADMIN_PASSWORD }}
  DATABASE_URL: ${{ secrets.DATABASE_URL }}
  POSTGRES_PASSWORD: ${{ secrets.POSTGRES_PASSWORD }}
```

Because services now support *_FILE, you can alternatively write secrets to ephemeral files and pass *_FILE env vars if preferred. Avoid echoing secret content to logs.

## Rotation runbook (JWT secrets and admin password)

1. Prepare new secrets
   - Generate a strong JWT secret (at least 32 bytes). Example: `openssl rand -hex 32`
   - Choose a new admin password and store in your secrets manager
2. Staging/CI rollout
   - Update CI secrets (JWT_SECRET / INTERNAL_JWT_SECRET / ADMIN_PASSWORD)
   - Deploy or run CI stack; validate login and service-to-service calls
3. Production rollout
   - Update secrets in your secret backend (e.g., Kubernetes Secret, Vault)
   - Restart deployments to pick up new secret values
4. Validation
   - Check identity-svc login and token issuance
   - Verify gateway↔backend internal auth (pricing/label) works
   - Monitor error rates and 401s in Grafana
5. Cleanup
   - Remove/rotate old secrets from the manager; update runbooks with new rotation date

## Best practices and guardrails

- Never commit real secrets to the repo; only examples and placeholders
- Prefer *_FILE envs in containerized environments; they reduce risk of accidental log exposure
- Keep secrets out of application logs; the code changes avoid printing secret contents
- Scope separate secrets for external JWTs (end-users) vs internal service JWTs (S2S)
- For Postgres, avoid embedding credentials in `DATABASE_URL` when possible; use password files in prod
- In Kubernetes, use `secretKeyRef`; this code already works with that pattern

## References
- Docker official images: *_FILE support (Postgres)
- 12-factor app: config via env vars
- OWASP: Sensitive data exposure and secrets handling

