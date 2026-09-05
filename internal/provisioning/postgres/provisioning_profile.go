// Package postgres implements the ProvisioningProfile domain's
// ProvisioningProfileRepository against PostgreSQL using pgx directly —
// no ORM — following the exact pattern established by
// internal/product/postgres.ProductRepository (the closest precedent: a
// required foreign key to a sibling domain).
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning"
)

// ProvisioningProfileRepository implements
// provisioning.ProvisioningProfileRepository against PostgreSQL.
type ProvisioningProfileRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ provisioning.ProvisioningProfileRepository = (*ProvisioningProfileRepository)(nil)

// NewProvisioningProfileRepository builds a ProvisioningProfileRepository.
func NewProvisioningProfileRepository(db database.Querier, clock clock.Clock, ids id.Generator) *ProvisioningProfileRepository {
	return &ProvisioningProfileRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves a ProvisioningProfile by ID, or an apperror.KindNotFound
// error if none exists.
func (r *ProvisioningProfileRepository) Get(ctx context.Context, profileID uuid.UUID) (provisioning.ProvisioningProfile, error) {
	const query = `
		SELECT id, product_id, vendor, profile_name, description, created_at, updated_at
		FROM provisioning_profiles
		WHERE id = $1
	`

	p, err := scanProvisioningProfile(r.db.QueryRow(ctx, query, profileID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return provisioning.ProvisioningProfile{}, profileNotFound(profileID)
		}
		return provisioning.ProvisioningProfile{}, translateError("get provisioning profile", err)
	}
	return p, nil
}

// List returns every ProvisioningProfile, ordered by vendor then profile
// name for stable, human-useful output.
func (r *ProvisioningProfileRepository) List(ctx context.Context) ([]provisioning.ProvisioningProfile, error) {
	const query = `
		SELECT id, product_id, vendor, profile_name, description, created_at, updated_at
		FROM provisioning_profiles
		ORDER BY vendor, profile_name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list provisioning profiles", err)
	}
	defer rows.Close()

	profiles := []provisioning.ProvisioningProfile{}
	for rows.Next() {
		p, err := scanProvisioningProfile(rows)
		if err != nil {
			return nil, translateError("scan provisioning profile row", err)
		}
		profiles = append(profiles, p)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list provisioning profiles", err)
	}

	return profiles, nil
}

// Create inserts p and returns the persisted record.
//
// The repository assigns ID, CreatedAt, and UpdatedAt itself — any values
// already set on the input for those fields are ignored. A ProductID
// that does not reference an existing Product, or a (product_id, vendor)
// or (vendor, profile_name) pair that already exists, fails with an
// apperror.KindConflict error (see translateError).
func (r *ProvisioningProfileRepository) Create(ctx context.Context, p provisioning.ProvisioningProfile) (provisioning.ProvisioningProfile, error) {
	const query = `
		INSERT INTO provisioning_profiles (id, product_id, vendor, profile_name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		RETURNING id, product_id, vendor, profile_name, description, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanProvisioningProfile(r.db.QueryRow(ctx, query,
		r.ids.New(), p.ProductID, p.Vendor, p.ProfileName, p.Description, now))
	if err != nil {
		return provisioning.ProvisioningProfile{}, translateError("create provisioning profile", err)
	}
	return created, nil
}

// Update overwrites the mutable fields of the ProvisioningProfile
// identified by p.ID and returns the persisted record, or an
// apperror.KindNotFound error if it does not exist.
func (r *ProvisioningProfileRepository) Update(ctx context.Context, p provisioning.ProvisioningProfile) (provisioning.ProvisioningProfile, error) {
	const query = `
		UPDATE provisioning_profiles
		SET product_id = $1, vendor = $2, profile_name = $3, description = $4, updated_at = $5
		WHERE id = $6
		RETURNING id, product_id, vendor, profile_name, description, created_at, updated_at
	`

	updated, err := scanProvisioningProfile(r.db.QueryRow(ctx, query,
		p.ProductID, p.Vendor, p.ProfileName, p.Description, r.clock.Now(), p.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return provisioning.ProvisioningProfile{}, profileNotFound(p.ID)
		}
		return provisioning.ProvisioningProfile{}, translateError("update provisioning profile", err)
	}
	return updated, nil
}

// Delete removes the ProvisioningProfile identified by id, or returns an
// apperror.KindNotFound error if it does not exist.
func (r *ProvisioningProfileRepository) Delete(ctx context.Context, profileID uuid.UUID) error {
	const query = `DELETE FROM provisioning_profiles WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, profileID)
	if err != nil {
		return translateError("delete provisioning profile", err)
	}
	if tag.RowsAffected() == 0 {
		return profileNotFound(profileID)
	}
	return nil
}

func profileNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("provisioning profile %s not found", id))
}

// rowScanner is satisfied by both pgx.Row (QueryRow, a single row) and
// pgx.Rows (Query, iterated one row at a time via Next then Scan), so
// scanProvisioningProfile backs Get/Create/Update and List alike.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanProvisioningProfile(row rowScanner) (provisioning.ProvisioningProfile, error) {
	var p provisioning.ProvisioningProfile
	err := row.Scan(&p.ID, &p.ProductID, &p.Vendor, &p.ProfileName, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}
