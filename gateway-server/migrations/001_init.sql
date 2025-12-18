CREATE TABLE IF NOT EXISTS virtual_services (
    id UUID PRIMARY KEY,
    owner_user_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    backend_url TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    require_auth BOOLEAN NOT NULL DEFAULT false,
    ti_mode VARCHAR(20) NOT NULL DEFAULT 'disabled' CHECK (ti_mode IN ('disabled', 'monitor', 'block')),
    rate_limit_enabled BOOLEAN NOT NULL DEFAULT false,
    rate_limit_requests INTEGER NOT NULL DEFAULT 100,
    rate_limit_window_sec INTEGER NOT NULL DEFAULT 60,
    log_retention_minutes INTEGER NOT NULL DEFAULT 30 CHECK (log_retention_minutes >= 1 AND log_retention_minutes <= 30),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vs_slug ON virtual_services(slug);
CREATE INDEX idx_vs_owner ON virtual_services(owner_user_id);
CREATE INDEX idx_vs_active ON virtual_services(is_active);

CREATE TABLE IF NOT EXISTS virtual_service_users (
    vs_id UUID NOT NULL REFERENCES virtual_services(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    granted_by VARCHAR(255) NOT NULL,
    granted_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (vs_id, user_id)
);

CREATE INDEX idx_vsu_user ON virtual_service_users(user_id);

CREATE TABLE IF NOT EXISTS virtual_service_ti_feeds (
    vs_id UUID NOT NULL REFERENCES virtual_services(id) ON DELETE CASCADE,
    feed_id UUID NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    added_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (vs_id, feed_id)
);

CREATE INDEX idx_vstf_vs ON virtual_service_ti_feeds(vs_id);
CREATE INDEX idx_vstf_feed ON virtual_service_ti_feeds(feed_id);
CREATE INDEX idx_vstf_active ON virtual_service_ti_feeds(is_active);

CREATE TABLE IF NOT EXISTS traffic_logs (
    id UUID PRIMARY KEY,
    vs_id UUID NOT NULL REFERENCES virtual_services(id) ON DELETE CASCADE,
    user_id VARCHAR(255),
    client_ip VARCHAR(45) NOT NULL,
    method VARCHAR(10) NOT NULL,
    path TEXT NOT NULL,
    request_headers JSONB,
    request_body TEXT,
    status_code INTEGER NOT NULL,
    response_headers JSONB,
    response_body TEXT,
    ioc_matches JSONB NOT NULL DEFAULT '[]',
    blocked BOOLEAN NOT NULL DEFAULT false,
    response_time_ms INTEGER NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tl_vs ON traffic_logs(vs_id);
CREATE INDEX idx_tl_timestamp ON traffic_logs(timestamp);
CREATE INDEX idx_tl_blocked ON traffic_logs(blocked);
CREATE INDEX idx_tl_user ON traffic_logs(user_id);