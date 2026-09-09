-- +goose Up
-- Every existing Device's free-text manufacturer/model already has a
-- matching DeviceManufacturer/DeviceModel row, backfilled by
-- database/migrations/00034_devicemanufacturer_device_manufacturers.sql
-- and database/migrations/00035_devicemodel_device_models.sql
-- respectively, so this column can go straight to NOT NULL with no
-- Device left pointing at nothing — the same shape
-- database/migrations/00033_oltmodel_olt_models.sql's olt_model_id
-- column takes.
ALTER TABLE devices ADD COLUMN device_model_id UUID REFERENCES device_models (id) ON DELETE RESTRICT;

UPDATE devices
SET device_model_id = device_models.id
FROM device_models
JOIN device_manufacturers ON device_manufacturers.id = device_models.manufacturer_id
WHERE devices.manufacturer = device_manufacturers.name AND devices.model = device_models.name;

ALTER TABLE devices ALTER COLUMN device_model_id SET NOT NULL;

-- Postgres does not automatically index foreign key columns. Without
-- this, both listing a DeviceModel's Devices (a near-certain future
-- need) and the ON DELETE RESTRICT check itself would be sequential
-- scans. Mirrors idx_olts_olt_model_id.
CREATE INDEX idx_devices_device_model_id ON devices (device_model_id);

-- Manufacturer and Model both moved from Device to DeviceModel (see
-- internal/inventory/model.go and internal/devicemodel/model.go's
-- package doc comments for why): storing them as free text on Device
-- risked drift and typos across otherwise-identical hardware, exactly
-- what this catalog exists to prevent.
ALTER TABLE devices DROP COLUMN manufacturer;
ALTER TABLE devices DROP COLUMN model;

-- +goose Down
ALTER TABLE devices ADD COLUMN manufacturer TEXT NOT NULL DEFAULT '';
ALTER TABLE devices ADD COLUMN model TEXT NOT NULL DEFAULT '';

UPDATE devices
SET manufacturer = device_manufacturers.name, model = device_models.name
FROM device_models
JOIN device_manufacturers ON device_manufacturers.id = device_models.manufacturer_id
WHERE devices.device_model_id = device_models.id;

ALTER TABLE devices ALTER COLUMN manufacturer DROP DEFAULT;
ALTER TABLE devices ALTER COLUMN model DROP DEFAULT;

ALTER TABLE devices DROP COLUMN device_model_id;
