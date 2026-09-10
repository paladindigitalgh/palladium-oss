-- +goose Up
CREATE TABLE notes (
    id UUID PRIMARY KEY,
    -- entity_type/entity_id are a loose reference, not a foreign key:
    -- Notes describe entities across every domain (customer, device,
    -- service, ...), and this package deliberately has no dependency on
    -- any of them (see internal/note's package doc comment; mirrors
    -- events.entity_type/entity_id exactly).
    entity_type TEXT NOT NULL,
    entity_id UUID NOT NULL,
    -- Unlike events.actor_user_id, never nullable: every Note is written
    -- by a human operator, never generated internally (see
    -- internal/note's package doc comment).
    author_user_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    -- A snapshot of the author's email at write time, not resolved via a
    -- join against users.email at read time -- see internal/note.Note's
    -- own doc comment on AuthorEmail for why.
    author_email TEXT NOT NULL,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

-- The one real query this table serves: "every note for this entity,
-- newest first" (see NoteRepository.ListByEntity).
CREATE INDEX idx_notes_entity ON notes (entity_type, entity_id, created_at);
CREATE INDEX idx_notes_author_user_id ON notes (author_user_id);

-- +goose Down
DROP TABLE notes;
