-- +goose Up
CREATE TABLE workflow_instances (
    id UUID PRIMARY KEY,
    -- Not a foreign key into an in-database Definitions table: Workflow
    -- Definitions are an in-code, versioned constant set (see
    -- internal/workflow/definition.go), not a database-editable resource
    -- in v1.
    definition_name TEXT NOT NULL,
    service_id UUID NOT NULL REFERENCES services (id) ON DELETE RESTRICT,
    requested_by_user_id UUID REFERENCES users (id) ON DELETE RESTRICT,
    status TEXT NOT NULL,
    retry_count INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_workflow_instances_service_id ON workflow_instances (service_id);
CREATE INDEX idx_workflow_instances_requested_by_user_id ON workflow_instances (requested_by_user_id);
CREATE INDEX idx_workflow_instances_status ON workflow_instances (status);
CREATE INDEX idx_workflow_instances_definition_name ON workflow_instances (definition_name);

-- +goose Down
DROP TABLE workflow_instances;
