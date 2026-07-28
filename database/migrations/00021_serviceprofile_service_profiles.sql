-- +goose Up
CREATE TABLE service_profiles (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    -- Intentionally plain TEXT, not an enum type or CHECK constraint: the
    -- set of valid values already lives in exactly one place —
    -- serviceprofile.Status in internal/serviceprofile — and callers are
    -- expected to validate against it before persisting (see
    -- ServiceProfile.Validate). Mirrors the same decision already made
    -- for catalogs.status in database/migrations/00011_catalog_catalogs.sql.
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Supports ORDER BY name in ServiceProfileRepository.List, the same
-- reasoning as idx_catalogs_name.
CREATE INDEX idx_service_profiles_name ON service_profiles (name);

-- Supports filtering service profiles by status, mirroring
-- idx_catalogs_status.
CREATE INDEX idx_service_profiles_status ON service_profiles (status);

-- +goose Down
DROP TABLE service_profiles;
