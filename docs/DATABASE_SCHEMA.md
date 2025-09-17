# Database Design (High Level)

## Principles
- Normalize core entities; use JSONB for flexible attributes where needed
- Partition large tables (shipments, tracking_events) by time/hash
- Strict FK constraints; soft deletes with deleted_at only where necessary
- Row-level auditing; created_by/updated_by; immutable event log

## Key Tables (MVP)
- users(id, email, password_hash, role, mfa_secret, created_at)
- sessions(id, user_id, refresh_token_hash, expires_at, revoked_at)
- shipments(id, shipper_id, service_level, from_addr, to_addr, value, status, created_at)
- packages(id, shipment_id, weight_kg, length_cm, width_cm, height_cm, barcode, hazmat)
- pricing_rules(id, name, zone, service_level, weight_min, weight_max, base, per_km, surcharge_json)
- labels(id, shipment_id, format, storage_url, checksum, created_at)
- tracking_events(id, shipment_id, type, ts, lat, lon, hub_id, metadata)
- hubs(id, name, code, address)
- notifications(id, shipment_id, event, channel, status, sent_at)

## Indexing Strategy
- shipments(status, created_at desc) for dashboards
- packages(shipment_id), labels(shipment_id)
- tracking_events(shipment_id, ts desc), tracking_events(ts), gist(lat, lon) if PostGIS
- users(email unique), sessions(user_id, expires_at)

## Partitioning
- shipments PARTITION BY RANGE (created_at monthly)
- tracking_events PARTITION BY RANGE (ts monthly); move to TimescaleDB for telemetry

## Data Retention
- tracking_events raw retained 12 months; aggregate rollups beyond
- logs 30 days hot, 365 days cold storage

## Migrations
- Use schema migration tool (Flyway or Liquibase) per service
- Test migrations in CI; backward-compatible, additive-first

