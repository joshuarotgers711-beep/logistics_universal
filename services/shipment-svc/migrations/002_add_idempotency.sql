-- Add idempotency key to shipments (unique, nullable)
ALTER TABLE shipments ADD COLUMN IF NOT EXISTS idempotency_key TEXT UNIQUE;

