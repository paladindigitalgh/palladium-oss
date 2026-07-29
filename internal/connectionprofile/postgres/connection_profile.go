// Package postgres implements the Connection Profile domain's
// ConnectionProfileRepository against PostgreSQL using pgx directly —
// no ORM — following the exact pattern established by
// internal/catalog/postgres.CatalogRepository.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/paladindigitalgh/palladium-oss/internal/connectionprofile"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// ConnectionProfileRepository implements
// connectionprofile.ConnectionProfileRepository against PostgreSQL. See
// internal/inventory/postgres/site.go for the reasoning behind depending
// on database.Querier and injecting clock/ids, which is not repeated
// here.
type ConnectionProfileRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ connectionprofile.ConnectionProfileRepository = (*ConnectionProfileRepository)(nil)

// NewConnectionProfileRepository builds a ConnectionProfileRepository.
func NewConnectionProfileRepository(db database.Querier, clock clock.Clock, ids id.Generator) *ConnectionProfileRepository {
	return &ConnectionProfileRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves a ConnectionProfile by ID, or an apperror.KindNotFound
// error if none exists.
func (r *ConnectionProfileRepository) Get(ctx context.Context, profileID uuid.UUID) (connectionprofile.ConnectionProfile, error) {
	const query = `
		SELECT id, name, protocol, port, authentication_id, timeout_ns, host_key_policy, description, created_at, updated_at
		FROM connection_profiles
		WHERE id = $1
	`

	p, err := scanConnectionProfile(r.db.QueryRow(ctx, query, profileID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return connectionprofile.ConnectionProfile{}, profileNotFound(profileID)
		}
		return connectionprofile.ConnectionProfile{}, translateError("get connection profile", err)
	}
	return p, nil
}

// List returns every ConnectionProfile, ordered by name for stable,
// human-useful output.
func (r *ConnectionProfileRepository) List(ctx context.Context) ([]connectionprofile.ConnectionProfile, error) {
	const query = `
		SELECT id, name, protocol, port, authentication_id, timeout_ns, host_key_policy, description, created_at, updated_at
		FROM connection_profiles
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list connection profiles", err)
	}
	defer rows.Close()

	profiles := []connectionprofile.ConnectionProfile{}
	for rows.Next() {
		p, err := scanConnectionProfile(rows)
		if err != nil {
			return nil, translateError("scan connection profile row", err)
		}
		profiles = append(profiles, p)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list connection profiles", err)
	}

	return profiles, nil
}

// Create inserts p and returns the persisted record.
//
// As with CatalogRepository.Create, the repository assigns ID, CreatedAt,
// and UpdatedAt itself — any values already set on the input
// ConnectionProfile for those fields are ignored. A Name that collides
// with an existing ConnectionProfile, or an AuthenticationID that does
// not reference an existing Authentication, fails with an
// apperror.KindConflict error (see translateError).
func (r *ConnectionProfileRepository) Create(ctx context.Context, p connectionprofile.ConnectionProfile) (connectionprofile.ConnectionProfile, error) {
	const query = `
		INSERT INTO connection_profiles (id, name, protocol, port, authentication_id, timeout_ns, host_key_policy, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
		RETURNING id, name, protocol, port, authentication_id, timeout_ns, host_key_policy, description, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanConnectionProfile(r.db.QueryRow(ctx, query,
		r.ids.New(), p.Name, p.Protocol, p.Port, p.AuthenticationID, int64(p.Timeout), string(p.HostKeyPolicy), p.Description, now))
	if err != nil {
		return connectionprofile.ConnectionProfile{}, translateError("create connection profile", err)
	}
	return created, nil
}

// Update overwrites the mutable fields of the ConnectionProfile
// identified by p.ID and returns the persisted record, or an
// apperror.KindNotFound error if it does not exist.
//
// CreatedAt cannot be altered through this method: the UPDATE statement
// below never assigns that column, and the RETURNING clause reports its
// true stored value regardless of what the input ConnectionProfile
// contained.
func (r *ConnectionProfileRepository) Update(ctx context.Context, p connectionprofile.ConnectionProfile) (connectionprofile.ConnectionProfile, error) {
	const query = `
		UPDATE connection_profiles
		SET name = $1, protocol = $2, port = $3, authentication_id = $4, timeout_ns = $5,
		    host_key_policy = $6, description = $7, updated_at = $8
		WHERE id = $9
		RETURNING id, name, protocol, port, authentication_id, timeout_ns, host_key_policy, description, created_at, updated_at
	`

	updated, err := scanConnectionProfile(r.db.QueryRow(ctx, query,
		p.Name, p.Protocol, p.Port, p.AuthenticationID, int64(p.Timeout), string(p.HostKeyPolicy), p.Description, r.clock.Now(), p.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return connectionprofile.ConnectionProfile{}, profileNotFound(p.ID)
		}
		return connectionprofile.ConnectionProfile{}, translateError("update connection profile", err)
	}
	return updated, nil
}

// Delete removes the ConnectionProfile identified by id, or returns an
// apperror.KindNotFound error if it does not exist.
func (r *ConnectionProfileRepository) Delete(ctx context.Context, profileID uuid.UUID) error {
	const query = `DELETE FROM connection_profiles WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, profileID)
	if err != nil {
		return translateError("delete connection profile", err)
	}
	if tag.RowsAffected() == 0 {
		return profileNotFound(profileID)
	}
	return nil
}

func profileNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("connection profile %s not found", id))
}

// rowScanner is satisfied by both pgx.Row (QueryRow, a single row) and
// pgx.Rows (Query, iterated one row at a time via Next then Scan), so
// scanConnectionProfile backs Get/Create/Update and List alike.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanConnectionProfile(row rowScanner) (connectionprofile.ConnectionProfile, error) {
	var (
		p             connectionprofile.ConnectionProfile
		hostKeyPolicy string
		timeoutNS     int64
	)
	err := row.Scan(
		&p.ID, &p.Name, &p.Protocol, &p.Port, &p.AuthenticationID, &timeoutNS,
		&hostKeyPolicy, &p.Description, &p.CreatedAt, &p.UpdatedAt,
	)
	p.HostKeyPolicy = connectionprofile.HostKeyPolicy(hostKeyPolicy)
	p.Timeout = time.Duration(timeoutNS)
	return p, err
}
