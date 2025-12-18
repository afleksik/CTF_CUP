ALTER TABLE realms ADD COLUMN IF NOT EXISTS gateway_vs_id TEXT;
ALTER TABLE realms ADD COLUMN IF NOT EXISTS gateway_vs_slug TEXT;
ALTER TABLE realms ADD COLUMN IF NOT EXISTS gateway_protected BOOLEAN DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_realms_gateway_vs_id ON realms(gateway_vs_id);
