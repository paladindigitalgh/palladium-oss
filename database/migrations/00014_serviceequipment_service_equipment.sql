-- +goose Up
CREATE TABLE service_equipment (
    id UUID PRIMARY KEY,
    -- Both foreign keys are required (NOT NULL) and RESTRICT, not
    -- CASCADE: a ServiceEquipment record cannot exist without naming both
    -- the Service it is delivering and the Device delivering it, and
    -- deleting either while a ServiceEquipment record still references it
    -- must fail loudly rather than silently orphaning or cascading away
    -- an equipment assignment. Mirrors the same fail-secure reasoning
    -- already documented for services.location_id/product_id in
    -- database/migrations/00013_service_services.sql.
    service_id UUID NOT NULL REFERENCES services (id) ON DELETE RESTRICT,
    device_id UUID NOT NULL REFERENCES devices (id) ON DELETE RESTRICT,
    -- Intentionally plain TEXT, not an enum type or CHECK constraint: the
    -- set of valid values already lives in exactly one place —
    -- serviceequipment.EquipmentRole in internal/serviceequipment — and
    -- callers are expected to validate against it before persisting (see
    -- ServiceEquipment.Validate). Mirrors the same decision already made
    -- for services.status in database/migrations/00013_service_services.sql.
    role TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',

    -- Nullable, not defaulted: a newly created assignment may not yet
    -- record when it was installed, and removed_at in particular is this
    -- milestone's literal definition of "active" (removed_at IS NULL —
    -- see serviceequipment.ServiceEquipment.Active). 0001-01-01 is a real
    -- (if nonsensical) instant, so these cannot default to any non-null
    -- value without risking that being mistaken for a genuine timestamp.
    -- Mirrors the same reasoning already given for
    -- services.activated_at/suspended_at/disconnected_at.
    installed_at TIMESTAMPTZ,
    removed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Deliberately no partial unique index on (device_id) WHERE removed_at IS
-- NULL, even though Postgres could enforce "at most one active
-- assignment per device" that way. Goal 2 is explicit that this rule is
-- implemented in ServiceEquipmentService (see
-- internal/serviceequipment/service), backed by the
-- GetActiveByDeviceID repository query — a deliberate, single-source-of-
-- truth choice, not an oversight: duplicating the rule into a database
-- constraint would create a second place it could drift from, and this
-- codebase's convention throughout is that business rules beyond
-- "required fields are present and well-formed" live in the service
-- layer, not the schema (see e.g. this milestone's own
-- ServiceEquipment.Validate doc comment).

-- Postgres does not automatically index foreign key columns. Without
-- these, both listing a Service's or Device's equipment assignments (a
-- near-certain future need), the ON DELETE RESTRICT check itself, and
-- GetActiveByDeviceID would be sequential scans. Mirrors
-- idx_services_location_id/idx_services_product_id.
CREATE INDEX idx_service_equipment_service_id ON service_equipment (service_id);
CREATE INDEX idx_service_equipment_device_id ON service_equipment (device_id);

-- Supports filtering assignments by role — goal 3 asks for it explicitly,
-- mirroring idx_services_status.
CREATE INDEX idx_service_equipment_role ON service_equipment (role);

-- +goose Down
DROP TABLE service_equipment;
