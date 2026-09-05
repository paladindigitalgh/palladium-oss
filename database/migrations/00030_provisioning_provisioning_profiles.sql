-- +goose Up
CREATE TABLE provisioning_profiles (
    id UUID PRIMARY KEY,
    product_id UUID NOT NULL REFERENCES products (id) ON DELETE RESTRICT,
    -- Plain TEXT, not an enum, mirroring olts.vendor exactly: a second
    -- vendor is a new row here, never a schema change.
    vendor TEXT NOT NULL,
    profile_name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    -- A given Product maps to at most one profile per vendor.
    UNIQUE (product_id, vendor),
    -- A given vendor's profile name identifies exactly one Product --
    -- two Products both claiming the same OLT profile is a data-entry
    -- mistake this constraint catches at write time.
    UNIQUE (vendor, profile_name)
);

-- Postgres does not automatically index foreign key columns. Without
-- this, the ON DELETE RESTRICT check on products would be a sequential
-- scan. Mirrors idx_products_catalog_id in
-- database/migrations/00012_product_products.sql.
CREATE INDEX idx_provisioning_profiles_product_id ON provisioning_profiles (product_id);

-- +goose Down
DROP TABLE provisioning_profiles;
