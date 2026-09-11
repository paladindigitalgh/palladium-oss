-- +goose Up
-- Access Network is removed as a domain concept: Palladium only ever runs
-- against a single physical network per instance, so requiring every OLT
-- to belong to a named AccessNetwork grouping was an unused layer of
-- indirection with no other domain ever referencing it. OLT becomes the
-- new root of the Network hierarchy (OLT -> PON Port -> Access Interface
-- -> Access Attachment). Dropping this column also drops its dependent
-- foreign key constraint and idx_olts_access_network_id, since both
-- depend solely on it.
ALTER TABLE olts DROP COLUMN access_network_id;

DROP TABLE access_networks;

-- +goose Down
-- Best-effort only: the original per-OLT AccessNetwork assignments
-- cannot be restored, the same as every other DROP COLUMN/DROP TABLE in
-- this codebase's migration history.
CREATE TABLE access_networks (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    status TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_access_networks_name ON access_networks (name);
CREATE INDEX idx_access_networks_status ON access_networks (status);

ALTER TABLE olts ADD COLUMN access_network_id UUID REFERENCES access_networks (id) ON DELETE RESTRICT;

CREATE INDEX idx_olts_access_network_id ON olts (access_network_id);
