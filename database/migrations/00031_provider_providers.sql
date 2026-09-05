-- +goose Up
CREATE TABLE providers (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    -- Intentionally plain TEXT, not an enum type or CHECK constraint, for
    -- status: the set of valid values already lives in exactly one place
    -- — provider.Status in internal/provider — and callers are expected
    -- to validate against it before persisting (see Provider.Validate).
    -- Mirrors the same decision already made for service_profiles.status
    -- in database/migrations/00021_serviceprofile_service_profiles.sql.
    status TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_providers_name ON providers (name);
CREATE INDEX idx_providers_status ON providers (status);

-- Unlike database/migrations/00022_service_add_service_profile_id.sql
-- (which could add its NOT NULL foreign key in one step because nothing
-- had ever written a row to the pre-v1, empty services table), real
-- Products already exist in every environment that has used this OSS at
-- all -- Provider is being introduced after the fact. So this uses the
-- two-phase approach that migration's own comment named as what a table
-- with genuine existing data would require: add the column nullable,
-- backfill every existing Product onto one auto-created Provider, then
-- make it NOT NULL. An operator running a single-ISP deployment can
-- rename this row (or leave it) via the new Providers panel; one running
-- open-access can split existing Products across additional Providers
-- they create the same way.
INSERT INTO providers (id, name, status, description, created_at, updated_at)
SELECT gen_random_uuid(), 'Default Provider', 'Active',
       'Auto-created when Provider was introduced, to hold Products that already existed. Rename or replace via Administration -> Plans.',
       now(), now()
WHERE EXISTS (SELECT 1 FROM products);

ALTER TABLE products ADD COLUMN provider_id UUID REFERENCES providers (id) ON DELETE RESTRICT;

UPDATE products
SET provider_id = (SELECT id FROM providers WHERE name = 'Default Provider' LIMIT 1)
WHERE provider_id IS NULL;

ALTER TABLE products ALTER COLUMN provider_id SET NOT NULL;

-- Postgres does not automatically index foreign key columns. Without
-- this, both listing a Provider's Products (a near-certain future need)
-- and the ON DELETE RESTRICT check itself would be sequential scans.
CREATE INDEX idx_products_provider_id ON products (provider_id);

-- +goose Down
ALTER TABLE products DROP COLUMN provider_id;
DROP TABLE providers;
