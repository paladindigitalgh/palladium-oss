-- +goose Up
-- Lets an operator mark one DeviceManufacturer, and separately one
-- DeviceModel per DeviceManufacturer, as the default New Device pre-
-- selects in its cascading Manufacturer/Model pickers (frontend
-- DeviceFormDialog.vue) -- still fully overridable per-Device, this only
-- changes what the form starts on. False for every existing row: no
-- manufacturer or model is more "default" than another until an operator
-- says so via Administration's Hardware page.
ALTER TABLE device_manufacturers ADD COLUMN is_default BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE device_models ADD COLUMN is_default BOOLEAN NOT NULL DEFAULT false;

-- "At most one default" is enforced here, not just in application code,
-- the same defense-in-depth reasoning every other invariant in this
-- schema gets (foreign keys, unique indexes) rather than trusting every
-- caller to never race or bypass the service layer.
--
-- device_manufacturers: a plain partial unique index over the (constant,
-- for indexed rows) is_default column -- a second TRUE row collides with
-- the first as a duplicate key, so at most one can ever be TRUE
-- system-wide.
CREATE UNIQUE INDEX idx_device_manufacturers_one_default ON device_manufacturers (is_default) WHERE is_default;

-- device_models: deliberately scoped per manufacturer_id rather than
-- system-wide -- a Kontron default and an Iskratel default coexist, since
-- DeviceFormDialog.vue's Model picker is always filtered to whichever
-- Manufacturer is currently selected (see that component's own doc
-- comment on modelOptions), so the only default that could ever actually
-- apply is the one belonging to the selected Manufacturer.
CREATE UNIQUE INDEX idx_device_models_one_default_per_manufacturer ON device_models (manufacturer_id) WHERE is_default;

-- +goose Down
DROP INDEX idx_device_models_one_default_per_manufacturer;
DROP INDEX idx_device_manufacturers_one_default;
ALTER TABLE device_models DROP COLUMN is_default;
ALTER TABLE device_manufacturers DROP COLUMN is_default;
