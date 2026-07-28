-- +goose Up
CREATE TABLE access_attachments (
    id UUID PRIMARY KEY,
    -- Both foreign keys are required (NOT NULL) and RESTRICT, not
    -- CASCADE: an AccessAttachment record cannot exist without naming
    -- both the AccessInterface it is plugged into and the
    -- ServiceEquipment plugged in, and deleting either while an
    -- AccessAttachment record still references it must fail loudly
    -- rather than silently orphaning or cascading away an attachment
    -- record. Mirrors the same fail-secure reasoning already documented
    -- for service_equipment.service_id/device_id in
    -- database/migrations/00014_serviceequipment_service_equipment.sql.
    access_interface_id UUID NOT NULL REFERENCES access_interfaces (id) ON DELETE RESTRICT,
    service_equipment_id UUID NOT NULL REFERENCES service_equipment (id) ON DELETE RESTRICT,

    -- Nullable, not defaulted: a newly created attachment may not yet
    -- record when it was installed, and removed_at in particular is this
    -- milestone's literal definition of "active" (removed_at IS NULL —
    -- see accessattachment.AccessAttachment.Active). 0001-01-01 is a real
    -- (if nonsensical) instant, so these cannot default to any non-null
    -- value without risking that being mistaken for a genuine timestamp.
    -- Mirrors the same reasoning already given for
    -- service_equipment.installed_at/removed_at.
    installed_at TIMESTAMPTZ,
    removed_at TIMESTAMPTZ,
    removal_reason TEXT NOT NULL DEFAULT '',

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Deliberately no partial unique index on (service_equipment_id) WHERE
-- removed_at IS NULL, even though Postgres could enforce "at most one
-- active attachment per ServiceEquipment" that way. This milestone is
-- explicit that this rule is implemented in AccessAttachmentService (see
-- internal/accessattachment/service), backed by the
-- GetActiveByServiceEquipmentID repository query — a deliberate,
-- single-source-of-truth choice, not an oversight, mirroring
-- database/migrations/00014_serviceequipment_service_equipment.sql's own
-- identical decision and reasoning for the same rule one domain over.

-- Postgres does not automatically index foreign key columns. Without
-- these, both listing an interface's or equipment's attachments (a
-- near-certain future need), the ON DELETE RESTRICT check itself, and
-- GetActiveByServiceEquipmentID would be sequential scans. Mirrors
-- idx_service_equipment_service_id/idx_service_equipment_device_id.
CREATE INDEX idx_access_attachments_access_interface_id ON access_attachments (access_interface_id);
CREATE INDEX idx_access_attachments_service_equipment_id ON access_attachments (service_equipment_id);

-- +goose Down
DROP TABLE access_attachments;
