-- +goose Up
CREATE TABLE customer_devices (
    id UUID PRIMARY KEY,
    -- Both foreign keys are required (NOT NULL) and RESTRICT, not
    -- CASCADE, mirroring service_equipment's own two foreign keys
    -- (database/migrations/00014_serviceequipment_service_equipment.sql):
    -- a CustomerDevice record cannot exist without naming both the
    -- Customer it is placed at and the Device placed there, and deleting
    -- either while a record still references it must fail loudly rather
    -- than silently orphaning or cascading away an attachment.
    customer_id UUID NOT NULL REFERENCES customers (id) ON DELETE RESTRICT,
    device_id UUID NOT NULL REFERENCES devices (id) ON DELETE RESTRICT,
    description TEXT NOT NULL DEFAULT '',

    -- Nullable, not defaulted -- the exact same reasoning as
    -- service_equipment.installed_at/removed_at: a newly created
    -- attachment may not yet record when it was attached, and
    -- detached_at in particular is this domain's literal definition of
    -- "active" (detached_at IS NULL -- see
    -- customerdevice.CustomerDevice.Active). 0001-01-01 is a real (if
    -- nonsensical) instant, so these cannot default to any non-null
    -- value without risking that being mistaken for a genuine timestamp.
    attached_at TIMESTAMPTZ,
    detached_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Deliberately no partial unique index on (device_id) WHERE detached_at
-- IS NULL, for the same single-source-of-truth reasoning
-- service_equipment's own migration documents for its identical choice:
-- "a Device may be attached to at most one Customer at a time" is
-- enforced in customerdevice/service.CustomerDeviceService, backed by
-- GetActiveByDeviceID, not a database constraint.

-- Postgres does not automatically index foreign key columns. Without
-- these, listing a Customer's or Device's attachment records, the ON
-- DELETE RESTRICT check itself, and GetActiveByDeviceID would be
-- sequential scans. Mirrors idx_service_equipment_service_id/
-- idx_service_equipment_device_id.
CREATE INDEX idx_customer_devices_customer_id ON customer_devices (customer_id);
CREATE INDEX idx_customer_devices_device_id ON customer_devices (device_id);

-- +goose Down
DROP TABLE customer_devices;
