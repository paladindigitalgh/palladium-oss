-- +goose Up
-- Changes workflow_instances.service_id from ON DELETE RESTRICT to ON
-- DELETE CASCADE, at the user's explicit request: "Remove Service" was
-- unconditionally blocked by any WorkflowInstance history at all, with
-- no UI or API path to clear it first -- unlike Service Equipment, which
-- already has its own real hard-delete action ("Remove" on the
-- Equipment section). Deleting a Service now takes its WorkflowInstance
-- history with it, rather than requiring an operator to reach for raw
-- SQL first (the workaround this gap forced during live testing). This
-- is a deliberate loss of that audit trail for a Service actually being
-- removed, not an oversight -- the user does not want it preserved once
-- the Service itself is gone.
ALTER TABLE workflow_instances DROP CONSTRAINT workflow_instances_service_id_fkey;
ALTER TABLE workflow_instances ADD CONSTRAINT workflow_instances_service_id_fkey
    FOREIGN KEY (service_id) REFERENCES services (id) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE workflow_instances DROP CONSTRAINT workflow_instances_service_id_fkey;
ALTER TABLE workflow_instances ADD CONSTRAINT workflow_instances_service_id_fkey
    FOREIGN KEY (service_id) REFERENCES services (id) ON DELETE RESTRICT;
