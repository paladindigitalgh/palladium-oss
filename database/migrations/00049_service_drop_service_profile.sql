-- +goose Up
-- Service Profile is removed as a domain concept: Service Type (see
-- 00048) now lives on Product, reached from Service by joining through
-- ProductID, never as Service's own duplicate field -- the same
-- "obtained through a join, not a redundant field" reasoning
-- internal/service/model.go already documents for why Service has no
-- CustomerID (obtained through Location instead).
DROP INDEX idx_services_service_profile_id;
ALTER TABLE services DROP COLUMN service_profile_id;
DROP TABLE service_profiles;

-- +goose Down
-- Best-effort only, the same as every other DROP COLUMN/DROP TABLE in
-- this codebase's migration history: original per-Service ServiceProfile
-- assignments cannot be restored. Recreates service_profiles matching
-- its true final shape (post-00044, which already dropped its
-- description column) -- not its original 00021 shape.
CREATE TABLE service_profiles (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_service_profiles_name ON service_profiles (name);
CREATE INDEX idx_service_profiles_status ON service_profiles (status);

ALTER TABLE services ADD COLUMN service_profile_id UUID REFERENCES service_profiles (id) ON DELETE RESTRICT;

CREATE INDEX idx_services_service_profile_id ON services (service_profile_id);
