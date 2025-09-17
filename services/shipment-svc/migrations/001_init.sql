-- shipments core tables
CREATE TABLE IF NOT EXISTS shipments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  shipper_id TEXT NOT NULL,
  service_level TEXT NOT NULL,
  from_addr JSONB NOT NULL,
  to_addr JSONB NOT NULL,
  value NUMERIC(12,2) DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'CREATED',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS ix_shipments_status_created_at ON shipments(status, created_at DESC);

CREATE TABLE IF NOT EXISTS packages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  shipment_id UUID NOT NULL REFERENCES shipments(id) ON DELETE CASCADE,
  weight_kg NUMERIC(10,3) NOT NULL,
  length_cm NUMERIC(10,2) NOT NULL,
  width_cm NUMERIC(10,2) NOT NULL,
  height_cm NUMERIC(10,2) NOT NULL,
  barcode TEXT,
  hazmat BOOLEAN DEFAULT FALSE
);
CREATE INDEX IF NOT EXISTS ix_packages_shipment ON packages(shipment_id);

CREATE TABLE IF NOT EXISTS labels (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  shipment_id UUID NOT NULL REFERENCES shipments(id) ON DELETE CASCADE,
  format TEXT NOT NULL,
  storage_url TEXT,
  checksum TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS ix_labels_shipment ON labels(shipment_id);

CREATE TABLE IF NOT EXISTS audit_log (
  id BIGSERIAL PRIMARY KEY,
  entity TEXT NOT NULL,
  entity_id UUID,
  action TEXT NOT NULL,
  user_id TEXT,
  payload JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- required extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto;

