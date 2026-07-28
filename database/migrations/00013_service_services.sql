-- +goose Up
CREATE TABLE services (
    id UUID PRIMARY KEY,
    -- Both foreign keys are required (NOT NULL) and RESTRICT, not
    -- CASCADE: a Service cannot exist without naming both the Location it
    -- is delivered to and the Product it is a purchase of, and deleting
    -- either while a Service still references it must fail loudly rather
    -- than silently orphaning or cascading away a subscriber's purchase
    -- record. Mirrors the same fail-secure reasoning already documented
    -- for locations.customer_id in
    -- database/migrations/00010_location_locations.sql.
    --
    -- There is intentionally no customer_id column: the customer
    -- relationship is obtained by joining through location_id ->
    -- locations.customer_id, per this milestone's explicit instruction.
    -- Storing it again here would be a second, redundant path to the same
    -- fact that could drift if a Location were ever reassigned.
    location_id UUID NOT NULL REFERENCES locations (id) ON DELETE RESTRICT,
    product_id UUID NOT NULL REFERENCES products (id) ON DELETE RESTRICT,
    -- Intentionally plain TEXT, not an enum type or CHECK constraint: the
    -- set of valid values already lives in exactly one place —
    -- service.ServiceStatus in internal/service — and callers are
    -- expected to validate against it before persisting (see
    -- Service.Validate). Mirrors the same decision already made for
    -- products.status in database/migrations/00012_product_products.sql.
    status TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',

    -- Nullable, not defaulted: a newly created Service (Pending) has none
    -- of these set, and 0001-01-01 is a real (if nonsensical) instant, so
    -- these cannot default to any non-null value without risking that
    -- being mistaken for a genuine timestamp. Mirrors the same reasoning
    -- already given for locations.latitude/longitude in
    -- database/migrations/00010_location_locations.sql.
    activated_at TIMESTAMPTZ,
    suspended_at TIMESTAMPTZ,
    disconnected_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Postgres does not automatically index foreign key columns. Without
-- these, both listing a Location's or Product's Services (a near-certain
-- future need) and each ON DELETE RESTRICT check itself would be
-- sequential scans. Mirrors idx_locations_customer_id and
-- idx_products_catalog_id.
CREATE INDEX idx_services_location_id ON services (location_id);
CREATE INDEX idx_services_product_id ON services (product_id);

-- Supports filtering services by status — goal 2 asks for it explicitly,
-- mirroring idx_products_status.
CREATE INDEX idx_services_status ON services (status);

-- +goose Down
DROP TABLE services;
