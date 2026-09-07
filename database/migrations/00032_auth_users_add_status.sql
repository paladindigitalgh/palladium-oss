-- +goose Up
-- Intentionally plain TEXT, not an enum type or CHECK constraint: the set
-- of valid values already lives in exactly one place -- auth.UserStatus
-- in internal/auth -- and callers are expected to validate against it
-- before persisting (see User.Validate). Mirrors the same decision
-- already made for providers.status in
-- database/migrations/00031_provider_providers.sql.
--
-- NOT NULL with a DEFAULT, added in one step rather than the two-phase
-- nullable-then-backfill-then-NOT-NULL approach that same migration used
-- for products.provider_id: that column needed a real, meaningful value
-- per existing row (which Provider a Product belongs to), so a default
-- would have been a guess. Every existing User -- including the
-- bootstrapped Administrator -- should simply be Active, so the DEFAULT
-- itself is the correct backfill; no separate UPDATE is needed.
ALTER TABLE users ADD COLUMN status TEXT NOT NULL DEFAULT 'Active';

CREATE INDEX idx_users_status ON users (status);

-- +goose Down
DROP INDEX idx_users_status;
ALTER TABLE users DROP COLUMN status;
