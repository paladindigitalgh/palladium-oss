-- +goose Up
CREATE TABLE customers (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    -- Intentionally plain TEXT, not an enum type or CHECK constraint, for
    -- both customer_type and status: the set of valid values already
    -- lives in exactly one place — customer.CustomerType and
    -- customer.CustomerStatus in internal/customer — and callers are
    -- expected to validate against it before persisting (see
    -- Customer.Validate). Duplicating that set into the schema would
    -- create a second copy that could silently drift from the Go source
    -- of truth. Mirrors the same decision already made for
    -- devices.status in database/migrations/00006_inventory_devices.sql.
    customer_type TEXT NOT NULL,
    status TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Supports ORDER BY name in CustomerRepository.List, the same reasoning as
-- idx_sites_name in database/migrations/00002_inventory_sites.sql. Not
-- UNIQUE: this milestone does not introduce a business rule that customer
-- names must be unique (a household and a small business next door could
-- plausibly share a name), so the schema does not invent one.
CREATE INDEX idx_customers_name ON customers (name);

-- Supports filtering customers by type or status — the two dimensions a
-- Customer list is realistically narrowed by (goal 3 asks for both
-- explicitly). Neither column is highly selective on its own (four and
-- three possible values respectively), but an index still lets Postgres
-- avoid a sequential scan once the table is large, and there is no
-- meaningful downside to having them at this table's write volume
-- (customers are created and edited far less often than, say, inventory
-- events would be).
CREATE INDEX idx_customers_customer_type ON customers (customer_type);
CREATE INDEX idx_customers_status ON customers (status);

-- +goose Down
DROP TABLE customers;
