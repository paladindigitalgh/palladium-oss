-- +goose Up
CREATE TABLE onu_authorizations (
    id UUID PRIMARY KEY,
    -- Both foreign keys are required (NOT NULL) and RESTRICT, not
    -- CASCADE: an OnuAuthorization record cannot exist without naming
    -- both the Device it authorizes and the OLT it is authorized on, and
    -- deleting either while an OnuAuthorization record still references
    -- it must fail loudly rather than silently orphaning or cascading
    -- away an authorization record -- the same fail-secure reasoning
    -- database/migrations/00020_accessattachment_access_attachments.sql
    -- gives for its own two foreign keys.
    device_id UUID NOT NULL REFERENCES devices (id) ON DELETE RESTRICT,
    olt_id UUID NOT NULL REFERENCES olts (id) ON DELETE RESTRICT,

    -- The raw interface path a vendor plugin's SSH client expects
    -- verbatim (e.g. "xgs/6/2"), not a foreign key to
    -- access_interfaces(id) -- see internal/onuauthorization/model.go's
    -- own doc comment for why.
    interface TEXT NOT NULL,

    -- authorized_at is always set (the moment the OLT-side authorize
    -- command succeeded); deauthorized_at is nullable and NULL means
    -- "still active" -- the same removed_at pattern
    -- access_attachments/service_equipment already establish, for the
    -- identical reason: 0001-01-01 is a real instant, so it cannot
    -- double as "not yet deauthorized."
    authorized_at TIMESTAMPTZ NOT NULL,
    deauthorized_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Deliberately no partial unique index on (device_id) WHERE
-- deauthorized_at IS NULL, mirroring
-- access_attachments'/service_equipment's identical choice and reasoning
-- in their own migrations: "at most one active authorization per
-- device" is application logic (see
-- internal/onuauthorization/postgres/onu_authorization.go's
-- GetActiveByDeviceID), not a database constraint.

-- Postgres does not automatically index foreign key columns. Without
-- these, both GetActiveByDeviceID and the ON DELETE RESTRICT checks
-- themselves would be sequential scans. Mirrors
-- idx_access_attachments_access_interface_id/idx_access_attachments_service_equipment_id.
CREATE INDEX idx_onu_authorizations_device_id ON onu_authorizations (device_id);
CREATE INDEX idx_onu_authorizations_olt_id ON onu_authorizations (olt_id);

-- +goose Down
DROP TABLE onu_authorizations;
