CREATE TABLE IF NOT EXISTS realms (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    owner_user_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS realm_users (
    realm_id UUID NOT NULL REFERENCES realms(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL CHECK (role IN ('admin', 'member')),
    added_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (realm_id, user_id)
);

CREATE TABLE IF NOT EXISTS assets (
    id UUID PRIMARY KEY,
    realm_id UUID NOT NULL REFERENCES realms(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    asset_type VARCHAR(100) NOT NULL CHECK (asset_type IN ('spirits', 'wine', 'beer', 'mixers', 'garnishes')),
    description TEXT,
    owner_user_id VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_realms_owner ON realms(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_realm_users_user ON realm_users(user_id);
CREATE INDEX IF NOT EXISTS idx_realm_users_realm ON realm_users(realm_id);
CREATE INDEX IF NOT EXISTS idx_assets_realm ON assets(realm_id);
CREATE INDEX IF NOT EXISTS idx_assets_owner ON assets(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_assets_type ON assets(asset_type);