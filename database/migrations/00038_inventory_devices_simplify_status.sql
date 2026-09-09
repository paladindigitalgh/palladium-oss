-- +goose Up
-- Collapses inventory.DeviceStatus from its original 7-value shelf/
-- rack-inventory lifecycle (Ordered, Received, InStock, Installed,
-- Maintenance, Retired, Disposed) down to 3 values (see
-- internal/inventory/device_status.go's own doc comment): the Device
-- Collection View tracks CPE out in customer homes and businesses, not
-- shelf inventory, and the only fact that actually varies for that kind
-- of device is whether it is currently attached to a Customer's Service.
--
-- Mapping, applied to whatever rows already exist:
--   Ordered, Received, InStock -> Unused   (never yet attached to a Service)
--   Installed, Maintenance     -> Active   (in service / actively assigned)
--   Retired, Disposed          -> Retired  (fully pulled out of service)
--
-- This is a data remap, not a schema change: status was, and remains, a
-- plain TEXT column with no CHECK constraint (see
-- database/migrations/00006_inventory_devices.sql) — validity is
-- application-level (inventory.DeviceStatus.Valid()), the same choice
-- this codebase makes throughout rather than duplicating an enum in SQL.
UPDATE devices SET status = 'Unused' WHERE status IN ('Ordered', 'Received', 'InStock');
UPDATE devices SET status = 'Active' WHERE status IN ('Installed', 'Maintenance');
UPDATE devices SET status = 'Retired' WHERE status IN ('Retired', 'Disposed');

-- +goose Down
-- Lossy by necessity: the original 7-value scheme distinguished states
-- (e.g. Ordered vs Received vs InStock) this migration's Up direction
-- already collapsed into one value (Unused) — there is no way to recover
-- which one a given row used to be. Down restores a single representative
-- value per collapsed group instead: InStock for Unused, Installed for
-- Active, Retired for Retired (already exact).
UPDATE devices SET status = 'InStock' WHERE status = 'Unused';
UPDATE devices SET status = 'Installed' WHERE status = 'Active';
