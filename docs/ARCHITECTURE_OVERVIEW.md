# Architecture Overview

## Architecture Style
- Microservices with clear bounded contexts, each owning its data
- API Gateway exposes REST APIs; OpenAPI-first contracts
- Async messaging via RabbitMQ initially (Kafka-ready design)
- PostgreSQL per service (logical separation; shared cluster ok for MVP)
- Redis for cache/session; Vault for secrets; object storage for media
- Containerized (Docker), orchestrated by Kubernetes; IaC with Helm + GitOps

## Core Services (MVP)
- gateway-api: request routing, auth, rate limiting, API docs
- identity-svc: users, roles, permissions, sessions, MFA, SSO adapters
- shipment-svc: orders, packages, lifecycle, audit trail
- pricing-svc: rate engine, pricing rules, surcharges, promotions
- label-svc: label rendering (PDF/PNG/ZPL), barcode/QR generation
- tracking-svc: events, GPS streaming, ETA calc, geofencing
- notification-svc: email/SMS abstraction, templates, webhooks
- hub-svc: hub scans, sortation events, chain-of-custody
- invoice-svc: invoices, line items, taxes, adjustments (Phase 2)
- payment-svc: tokenization, charges, refunds, settlements (Phase 2)
- ledger-svc: double-entry accounting (Phase 2)

## Data Ownership
- identity-svc: users, roles, tokens
- shipment-svc: shipments, packages, statuses, addresses
- pricing-svc: pricing rules, customer contracts
- tracking-svc: tracking events, device telemetry
- label-svc: label templates, artifacts
- invoice/payment/ledger: financial records (Phase 2)

## Messaging & Events
- Use domain events (e.g., ShipmentCreated, PackageScanned, Delivered)
- Outbox pattern for reliable publish; idempotent consumers
- Example flows:
  - ShipmentCreated → pricing-svc calculates quote → shipment-svc updates
  - InTransitLocationUpdate → tracking-svc updates ETA → notification-svc

## Security
- JWT access tokens (short TTL), refresh rotation, jti tracking
- RBAC enforcement at gateway and service level
- MFA (TOTP initially, SMS optional); SSO (OIDC/SAML) via identity-svc
- Vault-managed secrets; KMS-backed encryption keys; TLS everywhere

## Observability
- Structured logs (JSON), trace context propagation (W3C), OpenTelemetry
- Central metrics (Prometheus) + dashboards (Grafana); alerting (Alertmanager)

## Deployment Topology
- Namespaces per env; horizontal autoscaling; pod disruption budgets
- Blue/green or canary deploys via GitOps (Argo CD) and progressive delivery

## API Contract Strategy
- OpenAPI in repo; codegen for clients/servers; backward compatible changes
- API versioning (v1, v2); deprecation policy; contract tests in CI

## PWA & Mobile
- PWA: React (Vite) with offline cache and background sync
- Mobile: React Native (Expo) with barcode scanning, GPS, camera

## High-Level Flow (Create → Deliver)
1) Shipper creates shipment (gateway → shipment-svc)
2) pricing-svc computes rate; label-svc generates label
3) Driver pickup: scan + GPS + signature → tracking-svc
4) Transit: hub scans; ETA updates; exceptions handled
5) Delivery: scan, photo/signature, geofence → proof + notifications

## Future Readiness
- Multi-carrier adapters behind port/interface
- Kafka-compatible event contracts; schema registry
- Data warehouse feeds for analytics; CDC for financials

