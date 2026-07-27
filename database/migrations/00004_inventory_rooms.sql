-- +goose Up
CREATE TABLE rooms (
    id UUID PRIMARY KEY,
    building_id UUID NOT NULL REFERENCES buildings (id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- See idx_buildings_site_id: same reasoning, one level down the hierarchy.
CREATE INDEX idx_rooms_building_id ON rooms (building_id);

-- Mirrors idx_sites_name: supports ORDER BY name in List.
CREATE INDEX idx_rooms_name ON rooms (name);

-- +goose Down
DROP TABLE rooms;
