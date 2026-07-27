-- +goose Up

-- The DEFAULT here is deliberately temporary: adding it lets Postgres
-- backfill every existing row to 'Administrator' in the same statement
-- (goal 1: "existing users should default to Administrator during
-- migration so upgrades remain usable"), but it is dropped immediately
-- after so the default does not linger for new rows. The application
-- (internal/auth/postgres.UserRepository.Create) always supplies a Role
-- explicitly; if it ever had a bug that omitted one, a lingering DEFAULT
-- would silently grant that new user Administrator, a privilege
-- escalation bug hiding in the schema. Without a default, the same bug
-- fails loudly instead — a NOT NULL constraint violation — because
-- auth.Role.Valid() also rejects the empty string, so there is no
-- meaningful "safe" default value to fall back to for a brand new user
-- the way there is for a pre-existing one being upgraded.
ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'Administrator';
ALTER TABLE users ALTER COLUMN role DROP DEFAULT;

-- +goose Down
ALTER TABLE users DROP COLUMN role;
