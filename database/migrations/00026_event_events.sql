-- +goose Up
CREATE TABLE events (
    id UUID PRIMARY KEY,
    -- entity_type/entity_id are a loose reference, not a foreign key:
    -- events describe entities across every domain (customer, service,
    -- device, ...), and this package deliberately has no dependency on
    -- any of them (see internal/event's package doc comment).
    entity_type TEXT NOT NULL,
    entity_id UUID NOT NULL,
    type TEXT NOT NULL,
    message TEXT NOT NULL,
    metadata JSONB,
    -- Nullable: system-generated events (e.g. a workflow transition with
    -- no human operator behind it) have no actor to record.
    actor_user_id UUID REFERENCES users (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL
);

-- The one real query this table serves: "every event for this entity, in
-- order" (see EventRepository.ListByEntity).
CREATE INDEX idx_events_entity ON events (entity_type, entity_id, created_at);
CREATE INDEX idx_events_actor_user_id ON events (actor_user_id);

-- +goose Down
DROP TABLE events;
