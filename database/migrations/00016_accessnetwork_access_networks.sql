-- +goose Up
CREATE TABLE access_networks (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    -- Intentionally plain TEXT, not an enum type or CHECK constraint: the
    -- set of valid values already lives in exactly one place —
    -- accessnetwork.AccessNetworkStatus in internal/accessnetwork — and
    -- callers are expected to validate against it before persisting (see
    -- AccessNetwork.Validate). Mirrors the same decision already made
    -- for catalogs.status in database/migrations/00011_catalog_catalogs.sql.
    status TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Supports ORDER BY name in AccessNetworkRepository.List, the same
-- reasoning as idx_catalogs_name.
CREATE INDEX idx_access_networks_name ON access_networks (name);

-- Supports filtering access networks by status.
CREATE INDEX idx_access_networks_status ON access_networks (status);

-- +goose Down
DROP TABLE access_networks;
