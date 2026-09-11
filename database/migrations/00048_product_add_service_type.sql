-- +goose Up
-- Service Type replaces the standalone Service Profile domain (internal/
-- serviceprofile, dropped in 00049): a fixed 3-value enum lives directly
-- on Product instead of a free-CRUD entity Service referenced separately.
-- Backfilling every existing Product to 'Residential' here, as the ADD
-- COLUMN's own default, is this migration's entire backfill step --
-- safe because this is a dev-only instance with no real production data
-- (see CLAUDE.md). The DEFAULT is then dropped so every future
-- application-level INSERT must supply service_type explicitly --
-- mirrors this codebase's "no DB-level default relied on for new
-- inserts" convention (see product.Product.Validate's own required-field
-- check, added alongside this column).
ALTER TABLE products ADD COLUMN service_type TEXT NOT NULL DEFAULT 'Residential';
ALTER TABLE products ALTER COLUMN service_type DROP DEFAULT;

-- +goose Down
ALTER TABLE products DROP COLUMN service_type;
