ALTER TABLE virtual_service_ti_feeds
ADD COLUMN api_key TEXT;

CREATE INDEX idx_vstf_api_key ON virtual_service_ti_feeds(api_key) WHERE api_key IS NOT NULL;