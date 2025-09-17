# Service Catalog (MVP-first)

## gateway-api
- Responsibilities: routing, auth, RBAC, rate limiting, OpenAPI docs
- Tech: Node.js (NestJS) or Go (Fiber), OpenAPI-first, OTel, Redis cache

## identity-svc
- Responsibilities: users, roles, MFA, SSO (OIDC/SAML adapters), sessions
- Tech: Go (Chi) or Python (FastAPI); PostgreSQL; Redis for sessions; Vault

## shipment-svc
- Responsibilities: orders, packages, workflow state, audit trail
- Tech: Go; PostgreSQL (partitioned by month on created_at); RabbitMQ outbox

## pricing-svc
- Responsibilities: rate engine (distance/weight/service), surcharges, promos
- Tech: Python (FastAPI); rules engine; PostgreSQL; cache hot rules in Redis

## label-svc
- Responsibilities: PDF/PNG/ZPL label generation, QR/barcode
- Tech: Node.js (Headless Chrome/Puppeteer for PDF); object storage for artifacts

## tracking-svc
- Responsibilities: event ingestion, GPS, ETA, geofencing
- Tech: Go; TimescaleDB (Postgres extension) for telemetry; Redis for ETA cache

## hub-svc
- Responsibilities: hub scan ingestion, sortation events, custody chain
- Tech: Go; PostgreSQL; bulk endpoints + CSV import; idempotent ingest

## notification-svc
- Responsibilities: email/SMS abstraction, templates, webhook events
- Tech: Node.js; provider adapters (SendGrid/Twilio-like); retry + DLQ

## Future (Phase 2+)
- invoice-svc, payment-svc, ledger-svc, analytics-svc, carrier-adapter-svc

## Cross-Cutting Concerns
- Auth middleware library; tracing; logging; error schema; pagination spec
- Event schema registry; idempotency keys; correlation IDs

