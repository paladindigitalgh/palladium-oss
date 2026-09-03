-- +goose Up
CREATE TABLE contacts (
    id UUID PRIMARY KEY,
    -- ON DELETE CASCADE, not RESTRICT like locations.customer_id
    -- (database/migrations/00010_location_locations.sql) or
    -- services.location_id: a Contact has no downstream foreign key
    -- dependents and no operational significance of its own (nothing
    -- else in the system acts because a Contact exists) — it is a phone
    -- book entry, not a durable record worth blocking a Customer
    -- deletion over. This is a deliberate, reviewed difference from
    -- every other Customer sub-resource in this schema, not an
    -- oversight.
    customer_id UUID NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    -- Intentionally plain TEXT, not an enum type or CHECK constraint, for
    -- both role and status: the set of valid values already lives in
    -- exactly one place — contact.ContactRole and contact.ContactStatus
    -- in internal/contact — and callers are expected to validate against
    -- it before persisting (see Contact.Validate). Mirrors the same
    -- decision already made for locations.type/status in
    -- database/migrations/00010_location_locations.sql.
    role TEXT NOT NULL,
    email TEXT NOT NULL DEFAULT '',
    phone TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,

    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Postgres does not automatically index foreign key columns. Without
-- this, listing a Customer's Contacts (the only way this table is ever
-- queried from the frontend) would be a sequential scan. Mirrors
-- idx_locations_customer_id in
-- database/migrations/00010_location_locations.sql.
CREATE INDEX idx_contacts_customer_id ON contacts (customer_id);

-- Supports ORDER BY name in ContactRepository.List, the same reasoning
-- as idx_locations_name.
CREATE INDEX idx_contacts_name ON contacts (name);

-- Supports filtering by role or status, the same reasoning as
-- idx_locations_type/idx_locations_status.
CREATE INDEX idx_contacts_role ON contacts (role);
CREATE INDEX idx_contacts_status ON contacts (status);

-- +goose Down
DROP TABLE contacts;
