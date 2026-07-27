-- +goose Up
CREATE TABLE racks (
    id UUID PRIMARY KEY,
    -- Nullable: per the Inventory Philosophy lifecycle (Ordered -> Received
    -- -> Stored -> Installed -> ...), a rack can exist before it is
    -- installed in a specific room — see inventory.Rack in
    -- internal/inventory/model.go.
    room_id UUID REFERENCES rooms (id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- See idx_buildings_site_id: same reasoning. A NULL room_id is simply never
-- matched by the equality lookups this index serves (the referential check
-- and future "list racks in this room" queries), so nullability doesn't
-- weaken the justification.
CREATE INDEX idx_racks_room_id ON racks (room_id);

-- Mirrors idx_sites_name: supports ORDER BY name in List.
CREATE INDEX idx_racks_name ON racks (name);

-- +goose Down
DROP TABLE racks;
