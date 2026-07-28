// Package postgres implements the Service Profile domain's
// ServiceProfileRepository against PostgreSQL using pgx directly — no
// ORM — following the exact pattern established by
// internal/catalog/postgres.CatalogRepository.
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
	"github.com/paladindigitalgh/palladium-oss/internal/serviceprofile"
)

// ServiceProfileRepository implements
// serviceprofile.ServiceProfileRepository against PostgreSQL. See
// internal/inventory/postgres/site.go for the reasoning behind depending
// on database.Querier and injecting clock/ids, which is not repeated
// here.
type ServiceProfileRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ serviceprofile.ServiceProfileRepository = (*ServiceProfileRepository)(nil)

// NewServiceProfileRepository builds a ServiceProfileRepository.
func NewServiceProfileRepository(db database.Querier, clock clock.Clock, ids id.Generator) *ServiceProfileRepository {
	return &ServiceProfileRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves a ServiceProfile by ID, or an apperror.KindNotFound
// error if none exists.
func (r *ServiceProfileRepository) Get(ctx context.Context, profileID uuid.UUID) (serviceprofile.ServiceProfile, error) {
	const query = `
		SELECT id, name, description, status, created_at, updated_at
		FROM service_profiles
		WHERE id = $1
	`

	p, err := scanServiceProfile(r.db.QueryRow(ctx, query, profileID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return serviceprofile.ServiceProfile{}, profileNotFound(profileID)
		}
		return serviceprofile.ServiceProfile{}, translateError("get service profile", err)
	}
	return p, nil
}

// List returns every ServiceProfile, ordered by name for stable,
// human-useful output (see the index added on that column in the
// migration).
func (r *ServiceProfileRepository) List(ctx context.Context) ([]serviceprofile.ServiceProfile, error) {
	const query = `
		SELECT id, name, description, status, created_at, updated_at
		FROM service_profiles
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list service profiles", err)
	}
	defer rows.Close()

	profiles := []serviceprofile.ServiceProfile{}
	for rows.Next() {
		p, err := scanServiceProfile(rows)
		if err != nil {
			return nil, translateError("scan service profile row", err)
		}
		profiles = append(profiles, p)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list service profiles", err)
	}

	return profiles, nil
}

// Create inserts p and returns the persisted record.
//
// As with CatalogRepository.Create, the repository assigns ID, CreatedAt,
// and UpdatedAt itself — any values already set on the input
// ServiceProfile for those fields are ignored. Status is taken from the
// input exactly as given; the repository has no business logic and does
// not decide it.
func (r *ServiceProfileRepository) Create(ctx context.Context, p serviceprofile.ServiceProfile) (serviceprofile.ServiceProfile, error) {
	const query = `
		INSERT INTO service_profiles (id, name, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING id, name, description, status, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanServiceProfile(r.db.QueryRow(ctx, query,
		r.ids.New(), p.Name, p.Description, string(p.Status), now))
	if err != nil {
		return serviceprofile.ServiceProfile{}, translateError("create service profile", err)
	}
	return created, nil
}

// Update overwrites the mutable fields (Name, Description, Status) of
// the ServiceProfile identified by p.ID and returns the persisted
// record, or an apperror.KindNotFound error if it does not exist.
//
// CreatedAt cannot be altered through this method: the UPDATE statement
// below never assigns that column, and the RETURNING clause reports its
// true stored value regardless of what the input ServiceProfile
// contained.
func (r *ServiceProfileRepository) Update(ctx context.Context, p serviceprofile.ServiceProfile) (serviceprofile.ServiceProfile, error) {
	const query = `
		UPDATE service_profiles
		SET name = $1, description = $2, status = $3, updated_at = $4
		WHERE id = $5
		RETURNING id, name, description, status, created_at, updated_at
	`

	updated, err := scanServiceProfile(r.db.QueryRow(ctx, query,
		p.Name, p.Description, string(p.Status), r.clock.Now(), p.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return serviceprofile.ServiceProfile{}, profileNotFound(p.ID)
		}
		return serviceprofile.ServiceProfile{}, translateError("update service profile", err)
	}
	return updated, nil
}

// Delete removes the ServiceProfile identified by id, or returns an
// apperror.KindNotFound error if it does not exist. If a Service still
// references this profile, the delete fails with apperror.KindConflict
// instead (see errors.go's translateError and the ON DELETE RESTRICT
// foreign key in database/migrations/00022_service_add_service_profile_id.sql).
func (r *ServiceProfileRepository) Delete(ctx context.Context, profileID uuid.UUID) error {
	const query = `DELETE FROM service_profiles WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, profileID)
	if err != nil {
		return translateError("delete service profile", err)
	}
	if tag.RowsAffected() == 0 {
		return profileNotFound(profileID)
	}
	return nil
}

func profileNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("service profile %s not found", id))
}

// rowScanner is satisfied by both pgx.Row (QueryRow, a single row) and
// pgx.Rows (Query, iterated one row at a time via Next then Scan), so
// scanServiceProfile backs Get/Create/Update and List alike.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanServiceProfile(row rowScanner) (serviceprofile.ServiceProfile, error) {
	var (
		p      serviceprofile.ServiceProfile
		status string
	)
	err := row.Scan(&p.ID, &p.Name, &p.Description, &status, &p.CreatedAt, &p.UpdatedAt)
	p.Status = serviceprofile.Status(status)
	return p, err
}
