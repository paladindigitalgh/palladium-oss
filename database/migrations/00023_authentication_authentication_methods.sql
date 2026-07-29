-- +goose Up
CREATE TABLE authentication_methods (
    id UUID PRIMARY KEY,
    -- UNIQUE, not just indexed: this milestone's explicit rule is "Name
    -- unique." Mirrors users.email UNIQUE in
    -- database/migrations/00007_auth_users.sql — the closest existing
    -- precedent for a column this codebase enforces uniqueness on at the
    -- database level, not just in Go validation (uniqueness is a
    -- cross-row check no single record's Validate() can make on its
    -- own; see authentication.Authentication.Validate's own doc comment
    -- on why it does not attempt this).
    name TEXT NOT NULL UNIQUE,
    -- Intentionally plain TEXT, not an enum type or CHECK constraint: the
    -- set of valid values already lives in exactly one place —
    -- authentication.AuthenticationType in internal/authentication — and
    -- callers are expected to validate against it before persisting (see
    -- Authentication.Validate). Mirrors the same decision already made
    -- for every other enum-backed column in this schema.
    authentication_type TEXT NOT NULL,
    username TEXT NOT NULL,
    -- password and private_key store AES-256-GCM ciphertext (base64), not
    -- plaintext — see internal/platform/encryption and
    -- internal/authentication/postgres's own doc comment for exactly
    -- where encryption happens. Plain TEXT, not BYTEA: the encrypted
    -- output is already base64 (safe, printable ASCII), and this
    -- codebase's established convention throughout is TEXT for anything
    -- string-shaped, reserving specialized column types for cases that
    -- actually need them.
    --
    -- Both are NOT NULL DEFAULT '', never nullable: exactly one of the
    -- two is populated depending on authentication_type (see
    -- Authentication.Validate's conditional requirement), and the other
    -- is simply empty — the same "optional string, empty means not set"
    -- convention every other optional TEXT column in this schema
    -- follows (e.g. description columns throughout), not NULL, which
    -- would be a second, redundant way to express the identical fact.
    password TEXT NOT NULL DEFAULT '',
    private_key TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- UNIQUE (above) already creates an index usable for ORDER BY name in
-- AuthenticationRepository.List and for uniqueness-conflict detection —
-- no separate CREATE INDEX needed, the same reasoning
-- database/migrations/00007_auth_users.sql documents for users.email.

-- Supports filtering authentication methods by type, mirroring
-- idx_olts_vendor and every other enum-column index in this schema.
CREATE INDEX idx_authentication_methods_authentication_type ON authentication_methods (authentication_type);

-- +goose Down
DROP TABLE authentication_methods;
