-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY,
    -- Case-sensitive, exactly as specified for this milestone. Real-world
    -- email matching is usually case-insensitive, but that is a schema
    -- decision (e.g. a functional unique index on lower(email), or the
    -- citext extension) this milestone does not make; adding it here would
    -- be schema design beyond what was asked. Revisit when a registration
    -- flow exists to actually exercise the collision case.
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- No separate index on email: UNIQUE already creates one, and
-- UserRepository.GetByEmail's lookup is served by it directly.

-- +goose Down
DROP TABLE users;
