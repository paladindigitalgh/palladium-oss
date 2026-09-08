-- +goose Up
CREATE TABLE olt_models (
    id UUID PRIMARY KEY,
    -- Intentionally plain TEXT, not an enum type or CHECK constraint: the
    -- set of valid values already lives in exactly one place —
    -- oltmodel.Vendor in internal/oltmodel — and callers are expected to
    -- validate against it before persisting (see OLTModel.Validate).
    -- Mirrors the same decision already made for olts.vendor in
    -- database/migrations/00017_olt_olts.sql, before Vendor moved to
    -- this domain.
    vendor TEXT NOT NULL,
    name TEXT NOT NULL,
    -- No CHECK (pon_port_count > 0): that rule already lives in
    -- OLTModel.Validate, and every caller of this table goes through
    -- OLTModelService, the same "validate in Go, not in the schema"
    -- choice already made throughout this codebase.
    pon_port_count INTEGER NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- A given vendor's model name identifies at most one catalog entry — two
-- rows both named "Kontron C16" would leave a future OLT's OLTModelID
-- ambiguous about which one is authoritative. Mirrors the reasoning
-- behind idx_products_provider_id being a plain (non-unique) index while
-- this is deliberately UNIQUE, the same distinction
-- database/migrations/00030_provisioning_provisioning_profiles.sql draws
-- between its own product_id index and its (product_id, vendor)
-- uniqueness constraint.
CREATE UNIQUE INDEX idx_olt_models_vendor_name ON olt_models (vendor, name);

-- Every OLT in an existing environment already has a free-text vendor
-- and model — this domain is being introduced after the fact, the same
-- situation database/migrations/00031_provider_providers.sql's Products
-- backfill describes. One OLTModel row is created per distinct
-- (vendor, model) pair already present in olts, so no existing OLT is
-- left pointing at nothing once olt_model_id becomes NOT NULL below.
--
-- pon_port_count is seeded at 1, a deliberately conservative placeholder
-- — this migration has no way to know how many PON ports a given
-- pre-existing OLT's chassis actually has, only that it exists. An
-- operator corrects the real count via the new OLT Models Administration
-- page; internal/olt/service.OLTService's auto-port-creation only runs
-- on OLT *creation*, so no port records are silently invented for these
-- already-existing OLTs by this migration.
INSERT INTO olt_models (id, vendor, name, pon_port_count, description, created_at, updated_at)
SELECT gen_random_uuid(), vendor, model, 1,
       'Auto-created when OLT Model was introduced, to hold existing OLTs whose vendor/model was previously free text. Correct the PON port count via Administration -> OLT Models.',
       now(), now()
FROM (SELECT DISTINCT vendor, model FROM olts) AS existing_olt_vendor_models;

ALTER TABLE olts ADD COLUMN olt_model_id UUID REFERENCES olt_models (id) ON DELETE RESTRICT;

UPDATE olts
SET olt_model_id = olt_models.id
FROM olt_models
WHERE olts.vendor = olt_models.vendor AND olts.model = olt_models.name;

ALTER TABLE olts ALTER COLUMN olt_model_id SET NOT NULL;

-- Postgres does not automatically index foreign key columns. Without
-- this, both listing an OLTModel's OLTs (a near-certain future need)
-- and the ON DELETE RESTRICT check itself would be sequential scans.
-- Mirrors idx_olts_access_network_id.
CREATE INDEX idx_olts_olt_model_id ON olts (olt_model_id);

-- Vendor and Model both moved from OLT to OLTModel (see
-- internal/olt/model.go and internal/oltmodel/model.go's package doc
-- comments for why): storing vendor in two places risked the two
-- disagreeing, exactly what this catalog exists to prevent.
ALTER TABLE olts DROP COLUMN vendor;
ALTER TABLE olts DROP COLUMN model;

-- +goose Down
ALTER TABLE olts ADD COLUMN vendor TEXT NOT NULL DEFAULT 'Other';
ALTER TABLE olts ADD COLUMN model TEXT NOT NULL DEFAULT '';

UPDATE olts
SET vendor = olt_models.vendor, model = olt_models.name
FROM olt_models
WHERE olts.olt_model_id = olt_models.id;

ALTER TABLE olts ALTER COLUMN vendor DROP DEFAULT;
ALTER TABLE olts ALTER COLUMN model DROP DEFAULT;

ALTER TABLE olts DROP COLUMN olt_model_id;
DROP TABLE olt_models;
