-- +goose Up
-- Tracks which of a Customer's Locations a placed Device physically sits
-- at -- purely informational (internal/customerdevice's own doc comment:
-- this domain answers "which Device is at this Customer's premises at
-- all," never how it is configured or what it delivers), so an operator
-- with several Locations for one Customer can tell which device is
-- where before any Service, and its own Location, exists to answer that
-- indirectly.
--
-- Nullable, not defaulted -- a placement recorded before the Location is
-- known (or before the Customer has any Location at all) is still a
-- legitimate CustomerDevice row, the same reasoning attached_at/
-- detached_at already establish on this table. RESTRICT, not CASCADE or
-- SET NULL, matching every other foreign key in this schema: removing a
-- Location that a CustomerDevice still names must fail loudly rather
-- than silently clearing the field out from under an operator.
ALTER TABLE customer_devices ADD COLUMN location_id UUID REFERENCES locations (id) ON DELETE RESTRICT;

-- Postgres does not automatically index foreign key columns. Mirrors
-- idx_customer_devices_customer_id/idx_customer_devices_device_id from
-- this table's own creating migration.
CREATE INDEX idx_customer_devices_location_id ON customer_devices (location_id);

-- +goose Down
DROP INDEX idx_customer_devices_location_id;
ALTER TABLE customer_devices DROP COLUMN location_id;
