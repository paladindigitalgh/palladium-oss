-- +goose Up
CREATE TABLE connection_profiles (
    id UUID PRIMARY KEY,
    -- UNIQUE, not just indexed: this milestone's explicit rule is "Name
    -- unique." Mirrors authentication_methods.name UNIQUE in
    -- database/migrations/00023_authentication_authentication_methods.sql.
    name TEXT NOT NULL UNIQUE,
    -- Plain, unconstrained TEXT: unlike host_key_policy below, this
    -- milestone's spec gives Protocol no closed set of valid values (see
    -- internal/connectionprofile/validate.go's own doc comment), so
    -- there is nothing to enforce here beyond "it's a string."
    protocol TEXT NOT NULL DEFAULT '',
    -- Plain INTEGER, not validated as a real port range: this
    -- milestone's Rules section for ConnectionProfile does not require
    -- Port at all (see internal/connectionprofile/validate.go), so 0
    -- ("not yet configured") is a legitimate stored value, not an
    -- error condition a CHECK constraint should reject.
    port INTEGER NOT NULL DEFAULT 0,
    -- Nullable: this milestone's Rules section does not require
    -- AuthenticationID either — a ConnectionProfile can exist as a
    -- template before any specific Authentication is bound to it (see
    -- connectionprofile.ConnectionProfile's own doc comment).
    -- RESTRICT, not CASCADE: deleting an Authentication that a
    -- ConnectionProfile still references must fail loudly rather than
    -- silently orphaning the profile's credential reference.
    authentication_id UUID REFERENCES authentication_methods (id) ON DELETE RESTRICT,
    -- Stores a time.Duration's underlying int64 nanosecond count
    -- directly — the same representation
    -- internal/platform/ssh.Config.Timeout already uses in Go — so
    -- converting a stored ConnectionProfile.Timeout into an
    -- ssh.Config.Timeout (a future milestone's job) is a straight
    -- assignment, never a lossy unit conversion. BIGINT, not INTEGER:
    -- a plain 32-bit INTEGER overflows around 2.1 seconds' worth of
    -- nanoseconds, nowhere near enough headroom for a real timeout
    -- value.
    timeout_ns BIGINT NOT NULL DEFAULT 0,
    -- Intentionally plain TEXT, not an enum type or CHECK constraint: the
    -- set of valid values already lives in exactly one place —
    -- connectionprofile.HostKeyPolicy in internal/connectionprofile —
    -- and callers are expected to validate against it before persisting
    -- (see ConnectionProfile.Validate). Mirrors the same decision
    -- already made for every other enum-backed column in this schema.
    -- Unlike protocol/port/authentication_id/timeout_ns above,
    -- host_key_policy IS required by ConnectionProfile.Validate (it is
    -- a closed enum this package itself defines), hence NOT NULL with
    -- no DEFAULT.
    host_key_policy TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- UNIQUE (above) already creates an index usable for ORDER BY name in
-- ConnectionProfileRepository.List — no separate CREATE INDEX needed,
-- the same reasoning database/migrations/00023's own comment documents
-- for authentication_methods.name.

-- Postgres does not automatically index foreign key columns. Without
-- this, both listing an Authentication's ConnectionProfiles (a
-- near-certain future need) and the ON DELETE RESTRICT check itself
-- would be sequential scans. Mirrors every other FK index in this
-- schema.
CREATE INDEX idx_connection_profiles_authentication_id ON connection_profiles (authentication_id);

-- Supports filtering connection profiles by host key policy, mirroring
-- idx_authentication_methods_authentication_type.
CREATE INDEX idx_connection_profiles_host_key_policy ON connection_profiles (host_key_policy);

-- +goose Down
DROP TABLE connection_profiles;
