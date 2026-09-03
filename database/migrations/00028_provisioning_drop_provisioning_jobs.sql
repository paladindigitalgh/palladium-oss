-- +goose Up
-- internal/provisioning has been replaced by internal/workflow +
-- internal/plugin (see database/migrations/00027_workflow_workflow_instances.sql).
-- Engine.Execute was never reachable from the API in the provisioning
-- design, so no ProvisioningJob row ever completed a real execution —
-- there is no historical data here worth preserving.
DROP TABLE provisioning_jobs;

-- +goose Down
CREATE TABLE provisioning_jobs (
    id UUID PRIMARY KEY,
    service_id UUID NOT NULL REFERENCES services (id) ON DELETE RESTRICT,
    requested_by_user_id UUID REFERENCES users (id) ON DELETE RESTRICT,
    operation TEXT NOT NULL,
    status TEXT NOT NULL,
    retry_count INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_provisioning_jobs_service_id ON provisioning_jobs (service_id);
CREATE INDEX idx_provisioning_jobs_requested_by_user_id ON provisioning_jobs (requested_by_user_id);
CREATE INDEX idx_provisioning_jobs_status ON provisioning_jobs (status);
CREATE INDEX idx_provisioning_jobs_operation ON provisioning_jobs (operation);
