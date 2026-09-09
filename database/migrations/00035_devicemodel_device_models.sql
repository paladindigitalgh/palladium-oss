-- +goose Up
CREATE TABLE device_models (
    id UUID PRIMARY KEY,
    -- RESTRICT, not CASCADE, matching every other foreign key in this
    -- schema: deleting a DeviceManufacturer that still has DeviceModels
    -- referencing it must fail loudly rather than silently orphaning
    -- those models.
    manufacturer_id UUID NOT NULL REFERENCES device_manufacturers (id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- A given manufacturer's model name identifies at most one catalog
-- entry — two rows both naming manufacturer X's "G-140W-CT" would leave
-- a future Device's DeviceModelID ambiguous about which one is
-- authoritative. Mirrors idx_olt_models_vendor_name's reasoning in
-- database/migrations/00033_oltmodel_olt_models.sql.
CREATE UNIQUE INDEX idx_device_models_manufacturer_id_name ON device_models (manufacturer_id, name);

-- Every Device in an existing environment already has a free-text
-- manufacturer and model — this domain is being introduced after the
-- fact, the same situation
-- database/migrations/00034_devicemanufacturer_device_manufacturers.sql's
-- own backfill describes, one level down. One DeviceModel row is
-- created per distinct (manufacturer, model) pair already present in
-- devices, joined to the DeviceManufacturer row that migration just
-- backfilled, so no existing Device is left pointing at nothing once
-- devices.model is replaced by a device_model_id foreign key in
-- database/migrations/00036_inventory_devices_add_device_model_id.sql.
INSERT INTO device_models (id, manufacturer_id, name, description, created_at, updated_at)
SELECT gen_random_uuid(), device_manufacturers.id, existing.model,
       'Auto-created when Device Model was introduced, to hold existing Devices whose model was previously free text.',
       now(), now()
FROM (SELECT DISTINCT manufacturer, model FROM devices) AS existing
JOIN device_manufacturers ON device_manufacturers.name = existing.manufacturer;

-- +goose Down
DROP TABLE device_models;
