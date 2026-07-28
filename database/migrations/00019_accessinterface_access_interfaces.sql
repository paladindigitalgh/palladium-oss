-- +goose Up
CREATE TABLE access_interfaces (
    id UUID PRIMARY KEY,
    -- Required and RESTRICT: an AccessInterface cannot exist without
    -- naming the PON port it is on, and deleting a PON port that still
    -- has interfaces must fail loudly rather than silently orphaning
    -- them. Mirrors the same fail-secure reasoning already documented
    -- for pon_ports.olt_id in
    -- database/migrations/00018_ponport_pon_ports.sql.
    pon_port_id UUID NOT NULL REFERENCES pon_ports (id) ON DELETE RESTRICT,
    -- Intentionally plain TEXT, not an enum type or CHECK constraint: the
    -- set of valid values already lives in exactly one place —
    -- accessinterface.Technology in internal/accessinterface — and
    -- callers are expected to validate against it before persisting (see
    -- AccessInterface.Validate). Mirrors the same decision already made
    -- for olts.vendor in database/migrations/00017_olt_olts.sql.
    technology TEXT NOT NULL,
    name TEXT NOT NULL,
    -- Same TEXT-not-enum reasoning as technology above, backed by
    -- accessinterface.Status.
    status TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Postgres does not automatically index foreign key columns. Without
-- this, both listing a PON port's interfaces (a near-certain future
-- need) and the ON DELETE RESTRICT check itself would be sequential
-- scans. Mirrors idx_pon_ports_olt_id.
CREATE INDEX idx_access_interfaces_pon_port_id ON access_interfaces (pon_port_id);

-- Supports filtering interfaces by technology or status, mirroring
-- idx_olts_vendor and idx_access_networks_status.
CREATE INDEX idx_access_interfaces_technology ON access_interfaces (technology);
CREATE INDEX idx_access_interfaces_status ON access_interfaces (status);

-- +goose Down
DROP TABLE access_interfaces;
