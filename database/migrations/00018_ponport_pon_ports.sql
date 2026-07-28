-- +goose Up
CREATE TABLE pon_ports (
    id UUID PRIMARY KEY,
    -- Required and RESTRICT: a PONPort cannot exist without naming the
    -- OLT it is on, and deleting an OLT that still has PON ports must
    -- fail loudly rather than silently orphaning them. Mirrors the same
    -- fail-secure reasoning already documented for olts.access_network_id
    -- in database/migrations/00017_olt_olts.sql.
    olt_id UUID NOT NULL REFERENCES olts (id) ON DELETE RESTRICT,
    -- Plain INTEGER, not TEXT: a port number is a number, not a
    -- vendor-specific label (see ponport.PONPort's doc comment on what
    -- this package deliberately does not model — no shelf/slot/port
    -- addressing scheme). Validation that it is positive lives in
    -- PONPort.Validate, not a CHECK constraint, for the same reason
    -- every other enum/business-rule validation in this schema is kept
    -- in Go rather than duplicated into the database.
    port_number INTEGER NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Postgres does not automatically index foreign key columns. Without
-- this, both listing an OLT's PON ports (a near-certain future need)
-- and the ON DELETE RESTRICT check itself would be sequential scans.
-- Mirrors idx_olts_access_network_id.
CREATE INDEX idx_pon_ports_olt_id ON pon_ports (olt_id);

-- +goose Down
DROP TABLE pon_ports;
