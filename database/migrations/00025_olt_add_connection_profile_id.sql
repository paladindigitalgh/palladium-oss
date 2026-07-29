-- +goose Up
-- The second ALTER TABLE migration in this codebase (the first was
-- database/migrations/00022_service_add_service_profile_id.sql, which
-- documents at length why adding a column to an existing table is safe
-- here: Palladium OSS has no production deployment yet, so there is no
-- existing olts data anywhere this could break). Unlike that migration,
-- this column is nullable, not NOT NULL — this milestone does not
-- require every OLT to already have a ConnectionProfileID (see
-- olt.OLT's own doc comment, "Connection Profile"), so there is no
-- backfill concern here even in principle.
--
-- RESTRICT, not CASCADE, matching every other foreign key in this
-- schema: deleting a ConnectionProfile that an OLT still references
-- must fail loudly rather than silently orphaning the OLT's connection
-- reference.
ALTER TABLE olts
    ADD COLUMN connection_profile_id UUID REFERENCES connection_profiles (id) ON DELETE RESTRICT;

-- Postgres does not automatically index foreign key columns. Without
-- this, both listing a ConnectionProfile's OLTs (a near-certain future
-- need) and the ON DELETE RESTRICT check itself would be sequential
-- scans. Mirrors every other FK index in this schema.
CREATE INDEX idx_olts_connection_profile_id ON olts (connection_profile_id);

-- +goose Down
DROP INDEX idx_olts_connection_profile_id;
ALTER TABLE olts DROP COLUMN connection_profile_id;
