-- +goose Up
CREATE TABLE device_manufacturers (
    id UUID PRIMARY KEY,
    -- UNIQUE, not just indexed: two rows both named "Nokia" would leave
    -- a future DeviceModel's ManufacturerID ambiguous about which one is
    -- authoritative. Mirrors idx_olt_models_vendor_name's reasoning in
    -- database/migrations/00033_oltmodel_olt_models.sql, one level
    -- simpler here since there is no separate closed-enum column to pair
    -- it with (see internal/devicemanufacturer/model.go's package doc
    -- comment on why Name is free text, not a Vendor-style enum).
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Every Device in an existing environment already has a free-text
-- manufacturer — this domain is being introduced after the fact, the
-- same situation database/migrations/00033_oltmodel_olt_models.sql's
-- OLTModel backfill describes. One DeviceManufacturer row is created
-- per distinct manufacturer value already present in devices, so no
-- existing Device is left pointing at nothing once devices.manufacturer
-- is replaced by a device_model_id foreign key in
-- database/migrations/00036_inventory_devices_add_device_model_id.sql.
INSERT INTO device_manufacturers (id, name, description, created_at, updated_at)
SELECT gen_random_uuid(), manufacturer,
       'Auto-created when Device Manufacturer was introduced, to hold existing Devices whose manufacturer was previously free text.',
       now(), now()
FROM (SELECT DISTINCT manufacturer FROM devices) AS existing_device_manufacturers;

-- +goose Down
DROP TABLE device_manufacturers;
