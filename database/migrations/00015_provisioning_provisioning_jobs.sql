-- +goose Up
CREATE TABLE provisioning_jobs (
    id UUID PRIMARY KEY,
    -- Required and RESTRICT: a ProvisioningJob cannot exist without
    -- naming the Service it acts on, and deleting a Service that still
    -- has provisioning history must fail loudly rather than silently
    -- orphaning that history. Mirrors the same fail-secure reasoning
    -- already documented for service_equipment.service_id in
    -- database/migrations/00014_serviceequipment_service_equipment.sql.
    service_id UUID NOT NULL REFERENCES services (id) ON DELETE RESTRICT,
    -- Nullable and RESTRICT: not every ProvisioningJob has a human
    -- requester (see provisioning.ProvisioningJob's doc comment on why
    -- RequestedByUserID is *uuid.UUID), but when one is recorded, the
    -- user account it names must not be deletable out from under a real
    -- audit trail.
    --
    -- References users(id), not "auth_users(id)": this milestone's own
    -- spec names the target table "auth_users", but no such table exists
    -- in this schema — the auth domain's user table is named "users"
    -- (see database/migrations/00007_auth_users.sql, whose *migration
    -- file* is named auth_users while the table it creates is named
    -- users). Following the spec literally here would reference a
    -- nonexistent table and fail at migration time; this uses the real
    -- table name instead.
    requested_by_user_id UUID REFERENCES users (id) ON DELETE RESTRICT,
    -- Intentionally plain TEXT, not an enum type or CHECK constraint, for
    -- both operation and status: the set of valid values already lives in
    -- exactly one place — provisioning.ProvisioningOperation and
    -- provisioning.ProvisioningStatus in internal/provisioning — and
    -- callers are expected to validate against it before persisting (see
    -- ProvisioningJob.Validate). Mirrors the same decision already made
    -- for services.status in database/migrations/00013_service_services.sql.
    operation TEXT NOT NULL,
    status TEXT NOT NULL,
    -- Defaults to 0, the same as any freshly created job that has never
    -- been retried. Never defaulted to NULL: RetryCount is a count, not
    -- an optional fact — it always has a real value, even if that value
    -- is zero (unlike error_message/started_at/completed_at below, which
    -- are genuinely absent until a specific lifecycle event happens).
    retry_count INTEGER NOT NULL DEFAULT 0,

    -- Nullable, not defaulted: none of these apply to a job that has not
    -- yet reached the relevant point in its lifecycle, and none of their
    -- zero values (empty string, 0001-01-01) can stand in for "not yet
    -- set" without risking that being mistaken for real data. Mirrors the
    -- same reasoning already given for
    -- service_equipment.installed_at/removed_at.
    error_message TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Postgres does not automatically index foreign key columns. Without
-- these, both listing a Service's provisioning history
-- (ListByServiceID — a real, implemented need, not a hypothetical future
-- one) and each ON DELETE RESTRICT check itself would be sequential
-- scans. Mirrors idx_service_equipment_service_id/device_id.
CREATE INDEX idx_provisioning_jobs_service_id ON provisioning_jobs (service_id);
CREATE INDEX idx_provisioning_jobs_requested_by_user_id ON provisioning_jobs (requested_by_user_id);

-- Supports filtering jobs by status or operation — goal 5 asks for both
-- explicitly, mirroring idx_service_equipment_role.
CREATE INDEX idx_provisioning_jobs_status ON provisioning_jobs (status);
CREATE INDEX idx_provisioning_jobs_operation ON provisioning_jobs (operation);

-- +goose Down
DROP TABLE provisioning_jobs;
