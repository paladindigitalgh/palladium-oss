-- +goose Up
CREATE TABLE olts (
    id UUID PRIMARY KEY,
    -- Required and RESTRICT: an OLT cannot exist without naming the
    -- AccessNetwork it belongs to, and deleting an AccessNetwork that
    -- still has OLTs must fail loudly rather than silently orphaning
    -- them. Mirrors the same fail-secure reasoning already documented
    -- for products.catalog_id in
    -- database/migrations/00012_product_products.sql.
    access_network_id UUID NOT NULL REFERENCES access_networks (id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    -- Intentionally plain TEXT, not an enum type or CHECK constraint: the
    -- set of valid values already lives in exactly one place —
    -- olt.Vendor in internal/olt — and callers are expected to validate
    -- against it before persisting (see OLT.Validate). Mirrors the same
    -- decision already made for products.category in
    -- database/migrations/00012_product_products.sql.
    vendor TEXT NOT NULL,
    -- Optional, like description below: empty means "not set." Unlike
    -- vendor, model is free text (there is no closed set of valid model
    -- names to enumerate), so it is never a candidate for the
    -- TEXT-backed-enum treatment vendor gets.
    model TEXT NOT NULL DEFAULT '',
    -- Plain TEXT, not the Postgres inet type: this milestone does not
    -- validate that this is a well-formed address (see OLT.Validate's
    -- doc comment for why), so there is no reason for the schema to
    -- enforce a stricter type than the Go domain layer does — that would
    -- just move the same unimplemented validation into the database
    -- instead of adding it.
    management_ip_address TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Postgres does not automatically index foreign key columns. Without
-- this, both listing an AccessNetwork's OLTs (a near-certain future
-- need) and the ON DELETE RESTRICT check itself would be sequential
-- scans. Mirrors idx_products_catalog_id.
CREATE INDEX idx_olts_access_network_id ON olts (access_network_id);

-- Supports ORDER BY name in OLTRepository.List, the same reasoning as
-- idx_access_networks_name.
CREATE INDEX idx_olts_name ON olts (name);

-- Supports filtering OLTs by vendor.
CREATE INDEX idx_olts_vendor ON olts (vendor);

-- +goose Down
DROP TABLE olts;
