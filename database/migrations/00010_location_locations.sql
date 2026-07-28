-- +goose Up
CREATE TABLE locations (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customers (id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    -- Intentionally plain TEXT, not an enum type or CHECK constraint, for
    -- both type and status: the set of valid values already lives in
    -- exactly one place — location.LocationType and location.LocationStatus
    -- in internal/location — and callers are expected to validate against
    -- it before persisting (see Location.Validate). Mirrors the same
    -- decision already made for customers.customer_type/status in
    -- database/migrations/00009_customer_customers.sql.
    type TEXT NOT NULL,
    status TEXT NOT NULL,

    -- Optional per goal 1 ("address fields are optional for now"): plain
    -- TEXT with an empty-string default, the same convention as every
    -- other optional text field in this schema (e.g. description below).
    address1 TEXT NOT NULL DEFAULT '',
    address2 TEXT NOT NULL DEFAULT '',
    city TEXT NOT NULL DEFAULT '',
    state TEXT NOT NULL DEFAULT '',
    postal_code TEXT NOT NULL DEFAULT '',
    country TEXT NOT NULL DEFAULT '',

    -- Nullable, not defaulted to 0: see location.Location's doc comment
    -- in internal/location/model.go for why 0 cannot mean "not supplied"
    -- for a coordinate the way empty string can for text. Plain
    -- DOUBLE PRECISION, not a PostGIS geography/geometry column — this
    -- milestone is explicitly not a GIS system.
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,

    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Postgres does not automatically index foreign key columns. Without
-- this, both listing a Customer's Locations (a near-certain future need)
-- and the ON DELETE RESTRICT check itself would be sequential scans.
-- Mirrors idx_buildings_site_id in
-- database/migrations/00003_inventory_buildings.sql.
CREATE INDEX idx_locations_customer_id ON locations (customer_id);

-- Supports ORDER BY name in LocationRepository.List, the same reasoning
-- as idx_sites_name and idx_customers_name.
CREATE INDEX idx_locations_name ON locations (name);

-- Supports filtering by type or status — goal 3 asks for both explicitly,
-- mirroring idx_customers_customer_type/idx_customers_status.
CREATE INDEX idx_locations_type ON locations (type);
CREATE INDEX idx_locations_status ON locations (status);

-- +goose Down
DROP TABLE locations;
