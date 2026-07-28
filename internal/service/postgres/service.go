// Package postgres implements the Service domain's ServiceRepository
// against PostgreSQL using pgx directly — no ORM — following the exact
// pattern established by internal/location/postgres.LocationRepository
// (the closest precedent: required foreign keys to sibling domains).
// Service was originally the first entity in this codebase with two
// required foreign keys rather than one; the Service Profile milestone
// added a third (ServiceProfileID), handled identically to LocationID
// and ProductID.
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
	"github.com/paladindigitalgh/palladium-oss/internal/service"
)

// ServiceRepository implements service.ServiceRepository against
// PostgreSQL. See internal/inventory/postgres/site.go for the reasoning
// behind depending on database.Querier and injecting clock/ids, which is
// not repeated here.
type ServiceRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ service.ServiceRepository = (*ServiceRepository)(nil)

// NewServiceRepository builds a ServiceRepository.
func NewServiceRepository(db database.Querier, clock clock.Clock, ids id.Generator) *ServiceRepository {
	return &ServiceRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves a Service by ID, or an apperror.KindNotFound error if
// none exists.
func (r *ServiceRepository) Get(ctx context.Context, serviceID uuid.UUID) (service.Service, error) {
	const query = `
		SELECT id, location_id, product_id, service_profile_id, status, description,
		       activated_at, suspended_at, disconnected_at, created_at, updated_at
		FROM services
		WHERE id = $1
	`

	s, err := scanService(r.db.QueryRow(ctx, query, serviceID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return service.Service{}, serviceNotFound(serviceID)
		}
		return service.Service{}, translateError("get service", err)
	}
	return s, nil
}

// List returns every Service, ordered by created_at for stable,
// human-useful output. Unlike Location or Product, Service has no name
// column to order by — a subscriber purchase is not user-named — so the
// next most natural ordering is creation order, oldest first, matching
// how such lists are typically reviewed operationally.
func (r *ServiceRepository) List(ctx context.Context) ([]service.Service, error) {
	const query = `
		SELECT id, location_id, product_id, service_profile_id, status, description,
		       activated_at, suspended_at, disconnected_at, created_at, updated_at
		FROM services
		ORDER BY created_at
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list services", err)
	}
	defer rows.Close()

	services := []service.Service{}
	for rows.Next() {
		s, err := scanService(rows)
		if err != nil {
			return nil, translateError("scan service row", err)
		}
		services = append(services, s)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list services", err)
	}

	return services, nil
}

// Create inserts s and returns the persisted record.
//
// As with LocationRepository.Create, the repository assigns ID, CreatedAt,
// and UpdatedAt itself — any values already set on the input Service for
// those fields are ignored. A LocationID, ProductID, or ServiceProfileID
// that does not reference an existing row fails with an
// apperror.KindConflict error (see translateError).
func (r *ServiceRepository) Create(ctx context.Context, s service.Service) (service.Service, error) {
	const query = `
		INSERT INTO services (
			id, location_id, product_id, service_profile_id, status, description,
			activated_at, suspended_at, disconnected_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
		RETURNING id, location_id, product_id, service_profile_id, status, description,
		          activated_at, suspended_at, disconnected_at, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanService(r.db.QueryRow(ctx, query,
		r.ids.New(), s.LocationID, s.ProductID, s.ServiceProfileID, string(s.Status), s.Description,
		s.ActivatedAt, s.SuspendedAt, s.DisconnectedAt, now))
	if err != nil {
		return service.Service{}, translateError("create service", err)
	}
	return created, nil
}

// Update overwrites the mutable fields of the Service identified by s.ID
// and returns the persisted record, or an apperror.KindNotFound error if
// it does not exist.
//
// CreatedAt cannot be altered through this method: the UPDATE statement
// below never assigns that column, and the RETURNING clause reports its
// true stored value regardless of what the input Service contained.
func (r *ServiceRepository) Update(ctx context.Context, s service.Service) (service.Service, error) {
	const query = `
		UPDATE services
		SET location_id = $1, product_id = $2, service_profile_id = $3, status = $4, description = $5,
		    activated_at = $6, suspended_at = $7, disconnected_at = $8, updated_at = $9
		WHERE id = $10
		RETURNING id, location_id, product_id, service_profile_id, status, description,
		          activated_at, suspended_at, disconnected_at, created_at, updated_at
	`

	updated, err := scanService(r.db.QueryRow(ctx, query,
		s.LocationID, s.ProductID, s.ServiceProfileID, string(s.Status), s.Description,
		s.ActivatedAt, s.SuspendedAt, s.DisconnectedAt, r.clock.Now(), s.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return service.Service{}, serviceNotFound(s.ID)
		}
		return service.Service{}, translateError("update service", err)
	}
	return updated, nil
}

// Delete removes the Service identified by id, or returns an
// apperror.KindNotFound error if it does not exist.
func (r *ServiceRepository) Delete(ctx context.Context, serviceID uuid.UUID) error {
	const query = `DELETE FROM services WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, serviceID)
	if err != nil {
		return translateError("delete service", err)
	}
	if tag.RowsAffected() == 0 {
		return serviceNotFound(serviceID)
	}
	return nil
}

func serviceNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("service %s not found", id))
}

// rowScanner is satisfied by both pgx.Row (QueryRow, a single row) and
// pgx.Rows (Query, iterated one row at a time via Next then Scan), so
// scanService backs Get/Create/Update and List alike.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanService(row rowScanner) (service.Service, error) {
	var (
		s      service.Service
		status string
	)
	err := row.Scan(
		&s.ID, &s.LocationID, &s.ProductID, &s.ServiceProfileID, &status, &s.Description,
		&s.ActivatedAt, &s.SuspendedAt, &s.DisconnectedAt, &s.CreatedAt, &s.UpdatedAt,
	)
	s.Status = service.ServiceStatus(status)
	return s, err
}
