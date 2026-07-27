-- +goose Up
CREATE TABLE sites (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Supports ORDER BY name in SiteRepository.List (letting Postgres satisfy
-- the sort from the index instead of a sort step) and the name-based
-- lookups a "find site by name" feature will need next. Not UNIQUE: this
-- milestone does not introduce a business rule that site names must be
-- unique, so the schema does not invent one either.
CREATE INDEX idx_sites_name ON sites (name);

-- +goose Down
DROP TABLE sites;
