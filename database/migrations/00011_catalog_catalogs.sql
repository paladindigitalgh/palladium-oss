-- +goose Up
CREATE TABLE catalogs (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    -- Intentionally plain TEXT, not an enum type or CHECK constraint: the
    -- set of valid values already lives in exactly one place —
    -- catalog.CatalogStatus in internal/catalog — and callers are
    -- expected to validate against it before persisting (see
    -- ProductCatalog.Validate). Mirrors the same decision already made
    -- for locations.status in database/migrations/00010_location_locations.sql.
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Supports ORDER BY name in CatalogRepository.List, the same reasoning as
-- idx_customers_name in database/migrations/00009_customer_customers.sql.
CREATE INDEX idx_catalogs_name ON catalogs (name);

-- Supports filtering catalogs by status — goal 3 asks for it explicitly.
CREATE INDEX idx_catalogs_status ON catalogs (status);

-- +goose Down
DROP TABLE catalogs;
