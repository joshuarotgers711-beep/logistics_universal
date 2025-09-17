# MVP Scope – Phase 1 (Months 1–3)

## In-Scope Features
- AuthN/Z: JWT, refresh rotation, roles (Shipper, Driver, Manager, Admin), TOTP MFA
- Shipment lifecycle: create, rate, label, pickup, transit events, delivery with POD
- Scanning: QR/barcodes; GPS capture; offline mode for scans + sync later
- Tracking: 30s updates; hub scan ingestion; ETA calc v1
- Notifications: email/SMS on key events (created, out for delivery, delivered)
- PWA: create/manage orders, track shipments; driver console (basic)

## Out-of-Scope (Future)
- Full financials (beyond minimal invoice stub)
- Multi-carrier integrations (use internal rate/label for MVP)
- Advanced route optimization (heuristics only)

## Acceptance Criteria (E2E)
- A shipper can create shipment with dimensions/weight/address and get rate + label
- A driver can accept job, scan pickup, and record GPS/signature
- Hub scans update status and chain-of-custody
- Delivery captured with geofence validation and POD evidence
- Shipper receives notifications; can view real-time tracking + ETA

## Service-Level Deliverables
- gateway-api: OAuth-compatible JWT validation, RBAC, rate limiting; OpenAPI
- identity-svc: users/roles, MFA (TOTP), refresh token rotation, session mgmt
- shipment-svc: endpoints to create/update shipments; audit trail
- pricing-svc: distance/weight/service-level rate engine; surcharges v1
- label-svc: PDF/PNG/ZPL label generation; QR payload spec; webhooks for artifacts
- tracking-svc: event ingestion; ETA v1; geofence check; device heartbeat
- hub-svc: batch scan ingestion; facility events
- notification-svc: provider-agnostic email/SMS; templates

## Initial API Examples (abbrev)
- POST /v1/shipments
- POST /v1/shipments/{id}/rate
- POST /v1/shipments/{id}/label
- POST /v1/shipments/{id}/events/pickup
- POST /v1/shipments/{id}/events/deliver
- GET /v1/track/{tracking_number}

## Data Model (MVP)
- Shipment(id, shipper_id, service_level, status, from_addr, to_addr, value)
- Package(id, shipment_id, weight_kg, dims_cm, barcode, hazmat)
- TrackingEvent(id, shipment_id, type, ts, lat, lon, hub_id, notes)
- Label(id, shipment_id, format, url, checksum)
- User(id, role, email, password_hash, mfa_secret)

## Non-Functional (MVP)
- P95 latency < 800ms core endpoints; 99.9% uptime
- Auditing for shipment status changes and auth events
- Basic DDoS/rate limiting at gateway; WAF in front (managed)

