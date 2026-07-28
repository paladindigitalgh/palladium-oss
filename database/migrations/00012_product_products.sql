-- +goose Up
CREATE TABLE products (
    id UUID PRIMARY KEY,
    catalog_id UUID NOT NULL REFERENCES catalogs (id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    -- Intentionally plain TEXT, not an enum type or CHECK constraint, for
    -- both category and status: the set of valid values already lives in
    -- exactly one place — product.ProductCategory and product.ProductStatus
    -- in internal/product — and callers are expected to validate against
    -- it before persisting (see Product.Validate). Mirrors the same
    -- decision already made for locations.type/status in
    -- database/migrations/00010_location_locations.sql.
    category TEXT NOT NULL,
    status TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Postgres does not automatically index foreign key columns. Without
-- this, both listing a Catalog's Products (a near-certain future need)
-- and the ON DELETE RESTRICT check itself would be sequential scans.
-- Mirrors idx_locations_customer_id in
-- database/migrations/00010_location_locations.sql.
CREATE INDEX idx_products_catalog_id ON products (catalog_id);

-- Supports ORDER BY name in ProductRepository.List, the same reasoning as
-- idx_catalogs_name.
CREATE INDEX idx_products_name ON products (name);

-- Supports filtering products by category or status — goal 3 asks for
-- both explicitly, mirroring idx_locations_type/idx_locations_status.
CREATE INDEX idx_products_category ON products (category);
CREATE INDEX idx_products_status ON products (status);

-- +goose Down
DROP TABLE products;
