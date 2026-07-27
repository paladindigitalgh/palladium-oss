-- +goose Up
CREATE TABLE devices (
    id UUID PRIMARY KEY,
    -- Nullable for the same reason as racks.room_id: a device can be
    -- ordered, received, and stored before it is ever racked.
    rack_id UUID REFERENCES racks (id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    manufacturer TEXT NOT NULL,
    model TEXT NOT NULL,
    serial_number TEXT NOT NULL,
    -- Optional, like description: empty means "not set".
    asset_tag TEXT NOT NULL DEFAULT '',
    -- Intentionally plain TEXT, not an enum type or CHECK constraint. The
    -- set of valid values already lives in exactly one place —
    -- inventory.DeviceStatus in internal/inventory/device_status.go — and
    -- callers are expected to validate against it before persisting (see
    -- Device.Validate). Duplicating that set into a CHECK constraint would
    -- create a second copy that could silently drift from the Go source of
    -- truth; this migration does not invent that duplication.
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_devices_rack_id ON devices (rack_id);

-- Mirrors idx_sites_name: supports ORDER BY name in List.
CREATE INDEX idx_devices_name ON devices (name);

-- +goose Down
DROP TABLE devices;
