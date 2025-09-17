# Phase 1 MVP Backlog (M1.1–M1.6)

Legend: [E] Epic, [S] Story, [T] Task; IDs used for dependencies.

## M1.1 Foundation (Repo, CI, IaC, Observability)
- [E] M1.1-E1 Platform bootstrap
  - [S] M1.1-S1 Monorepo structure (apis, services, docs)
    - AC: Repo scaffolding; CONTRIBUTING; CODEOWNERS; PR template
    - Depends: —
    - [T] M1.1-T1 Create service folders and README stubs
  - [S] M1.1-S2 CI skeleton per service
    - AC: Lint/test/build pipelines; container build with SBOM/signing
    - Depends: M1.1-S1
    - [T] M1.1-T2 GitHub Actions/CI config per service
  - [S] M1.1-S3 IaC baseline
    - AC: Helm charts templates; namespaces; HPA; PDB; Ingress
    - Depends: M1.1-S1
    - [T] M1.1-T3 Base Helm chart + values for dev/stg/prod
  - [S] M1.1-S4 Observability baseline
    - AC: OTel SDKs; Prometheus/Grafana; structured logging
    - Depends: M1.1-S2
    - [T] M1.1-T4 Add OTel middlewares to templates

## M1.2 AuthN/AuthZ (JWT, Refresh, MFA, RBAC)
- [E] M1.2-E1 Identity service
  - [S] M1.2-S1 Login + refresh + revoke
    - AC: 200 on valid credentials; 401 on invalid; refresh rotation
    - Depends: M1.1-S2
  - [S] M1.2-S2 TOTP MFA
    - AC: Enroll secret; verify code; lockout on 5 failed attempts
    - Depends: M1.2-S1
  - [S] M1.2-S3 RBAC roles (Shipper/Driver/Manager/Admin)
    - AC: Role claims in JWT; middleware enforces x-roles on endpoints
    - Depends: M1.2-S1
  - [S] M1.2-S4 Admin user management (create/get)
    - AC: ADMIN can create users; audit log for changes
    - Depends: M1.2-S3

## M1.3 Create + Rate + Label
- [E] M1.3-E1 Shipment creation
  - [S] M1.3-S1 Create shipment endpoint
    - AC: Validates addresses; persists shipment; returns trackingNumber
    - Depends: M1.2-S3
  - [S] M1.3-S2 Address validation rules v1
    - AC: Normalizes postal codes; rejects invalid country/state
    - Depends: M1.3-S1
- [E] M1.3-E2 Pricing integration
  - [S] M1.3-S3 Quote calculation via pricing-svc
    - AC: Returns currency, base, surcharges, total; P95 < 300ms
    - Depends: M1.3-S1
- [E] M1.3-E3 Label generation
  - [S] M1.3-S4 Label create (PDF default; PNG/ZPL optional)
    - AC: Returns label id/url/checksum; stored in object storage
    - Depends: M1.3-S3

## M1.4 Pickup & Transit
- [E] M1.4-E1 Driver app basics
  - [S] M1.4-S1 Pickup scan + GPS + signature
    - AC: 202 accepted; event persisted; shipment status → IN_TRANSIT
    - Depends: M1.3-S4
  - [S] M1.4-S2 Offline queue + sync
    - AC: Events queued offline; background sync on connectivity
    - Depends: M1.4-S1
- [E] M1.4-E2 Hub scans
  - [S] M1.4-S3 Ingest hub transit events
    - AC: Valid hubCode; updates chain-of-custody
    - Depends: M1.4-S1

## M1.5 Delivery & Proof
- [E] M1.5-E1 Delivery completion
  - [S] M1.5-S1 Delivery scan with geofence
    - AC: Validates within 100m; status → DELIVERED
    - Depends: M1.4-S3
  - [S] M1.5-S2 POD photo/signature
    - AC: Base64 media accepted; stored; linked to event
    - Depends: M1.5-S1
- [E] M1.5-E2 Notifications
  - [S] M1.5-S3 Event notifications (created, out-for-delivery, delivered)
    - AC: Emails/SMS dispatched via notification-svc; retry+DLQ
    - Depends: M1.4-S3

## M1.6 Hardening & Launch
- [E] M1.6-E1 Performance & resiliency
  - [S] M1.6-S1 Load test core flows
    - AC: P95 < 800ms; error rate < 0.5%
    - Depends: All core stories
  - [S] M1.6-S2 Security review
    - AC: Threat model; pentest; secrets validated; SAST/SCA clean
    - Depends: All core stories
- [E] M1.6-E2 Ops readiness
  - [S] M1.6-S3 Runbooks & on-call
    - AC: Runbooks published; dashboards and alerts configured
    - Depends: M1.1-S4

---

# Technical Tasks (Cross-cutting)
- [T] CT-1 Messaging (RabbitMQ) provision + client libs
  - AC: Durable exchanges/queues; outbox pattern template
  - Depends: M1.1-S3
- [T] CT-2 Database schemas & migrations (Postgres)
  - AC: Tables per docs/DATABASE_SCHEMA.md; Flyway/Liquibase set up
  - Depends: M1.1-S3
- [T] CT-3 Caching (Redis) for sessions and hot data
  - AC: TTL policies; connection pooling; metrics
  - Depends: M1.1-S4
- [T] CT-4 API Gateway policies
  - AC: Rate limiting; JWT validation; CORS; request size limits
  - Depends: M1.2-S3

# Service Scaffolding Tasks
- gateway-api (NestJS)
  - [T] GW-1 Project setup (Nest, OpenAPI, OTel)
  - [T] GW-2 Auth guard + RBAC decorators
  - [T] GW-3 Routes: shipments, tracking, labels proxy
- identity-svc (Go or Python)
  - [T] ID-1 Project setup
  - [T] ID-2 JWT issuance/validation; refresh rotation
  - [T] ID-3 TOTP MFA; user CRUD (admin)
- shipment-svc (Go)
  - [T] SH-1 Project setup + migrations
  - [T] SH-2 Create/get/patch shipments; audit log
  - [T] SH-3 Emit ShipmentCreated event (RabbitMQ outbox)
- pricing-svc (FastAPI)
  - [T] PR-1 Project setup
  - [T] PR-2 Quote endpoint; distance calc; rules engine v1
  - [T] PR-3 Cache hot rules in Redis
- label-svc (NestJS)
  - [T] LB-1 Project setup + Puppeteer
  - [T] LB-2 Generate QR/barcodes; render PDF/PNG/ZPL
  - [T] LB-3 Upload artifacts to object storage
- tracking-svc (Go)
  - [T] TR-1 Project setup + TimescaleDB schema
  - [T] TR-2 Ingest events; compute ETA v1; geofence check
  - [T] TR-3 Public tracking endpoint

# Dependencies Summary
- M1.3-S1 → M1.3-S3 → M1.3-S4 → M1.4-S1 → M1.4-S3 → M1.5-S1 → M1.5-S2
- M1.2-S1 → M1.2-S2 → M1.2-S3 → (enables all protected endpoints)
- CT-1, CT-2 complete before event-driven flows (SH-3, TR-2)

