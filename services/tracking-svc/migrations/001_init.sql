-- tracking events hypertable (TimescaleDB)
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS tracking_events (
  id UUID NOT NULL DEFAULT gen_random_uuid(),
  shipment_id UUID NOT NULL,
  type TEXT NOT NULL,
  ts TIMESTAMPTZ NOT NULL DEFAULT now(),
  lat DOUBLE PRECISION,
  lon DOUBLE PRECISION,
  hub_id TEXT,
  metadata JSONB,
  PRIMARY KEY (id, ts)
);

SELECT create_hypertable('tracking_events', 'ts', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS ix_tracking_shipment_ts ON tracking_events(shipment_id, ts DESC);
CREATE INDEX IF NOT EXISTS ix_tracking_ts ON tracking_events(ts DESC);

