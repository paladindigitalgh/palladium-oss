-- +goose Up
-- This is the first ALTER TABLE migration in this codebase — every prior
-- migration has been a CREATE TABLE for a brand-new domain. Adding
-- service_profile_id as NOT NULL to an existing table is safe here only
-- because Palladium OSS has no production deployment yet (this is still
-- pre-v1, greenfield development — see CLAUDE.md); there is no existing
-- services data anywhere that a NOT NULL ADD COLUMN with no default
-- could break. A future milestone altering a table that genuinely holds
-- production data would need a different, two-phase approach (add
-- nullable, backfill, then add the NOT NULL constraint) — this migration
-- deliberately does not build that machinery because nothing here needs
-- it yet.
--
-- REFERENCES ... ON DELETE RESTRICT and NOT NULL mirror this milestone's
-- explicit instruction and the same fail-secure reasoning already
-- documented for services.location_id/product_id in
-- database/migrations/00013_service_services.sql: a Service cannot exist
-- without naming the ServiceProfile describing its operational intent,
-- and deleting a ServiceProfile that still has Services referencing it
-- must fail loudly rather than silently orphaning them.
ALTER TABLE services
    ADD COLUMN service_profile_id UUID NOT NULL REFERENCES service_profiles (id) ON DELETE RESTRICT;

-- Postgres does not automatically index foreign key columns. Without
-- this, both listing a ServiceProfile's Services (a near-certain future
-- need) and the ON DELETE RESTRICT check itself would be sequential
-- scans. Mirrors idx_services_location_id/idx_services_product_id.
CREATE INDEX idx_services_service_profile_id ON services (service_profile_id);

-- +goose Down
DROP INDEX idx_services_service_profile_id;
ALTER TABLE services DROP COLUMN service_profile_id;
