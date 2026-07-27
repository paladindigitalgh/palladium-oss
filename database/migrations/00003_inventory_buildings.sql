-- +goose Up
CREATE TABLE buildings (
    id UUID PRIMARY KEY,
    site_id UUID NOT NULL REFERENCES sites (id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Postgres does not automatically index foreign key columns. Without this,
-- both BuildingRepository.List-by-site (future) and the referential check
-- ON DELETE RESTRICT performs against sites would be sequential scans.
CREATE INDEX idx_buildings_site_id ON buildings (site_id);

-- Mirrors idx_sites_name: supports ORDER BY name in List.
CREATE INDEX idx_buildings_name ON buildings (name);

-- +goose Down
DROP TABLE buildings;
