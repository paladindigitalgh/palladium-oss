// Package postgres implements the Location domain's LocationRepository
// against PostgreSQL using pgx directly — no ORM — following the exact
// pattern established by internal/inventory/postgres.SiteRepository.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/location"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// LocationRepository implements location.LocationRepository against
// PostgreSQL. See internal/inventory/postgres/site.go for the reasoning
// behind depending on database.Querier and injecting clock/ids, which is
// not repeated here.
type LocationRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ location.LocationRepository = (*LocationRepository)(nil)

// NewLocationRepository builds a LocationRepository.
func NewLocationRepository(db database.Querier, clock clock.Clock, ids id.Generator) *LocationRepository {
	return &LocationRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves a Location by ID, or an apperror.KindNotFound error if
// none exists.
func (r *LocationRepository) Get(ctx context.Context, locationID uuid.UUID) (location.Location, error) {
	const query = `
		SELECT id, customer_id, name, type, status,
		       address1, address2, city, state, postal_code, country,
		       latitude, longitude, description, created_at, updated_at
		FROM locations
		WHERE id = $1
	`

	l, err := scanLocation(r.db.QueryRow(ctx, query, locationID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return location.Location{}, locationNotFound(locationID)
		}
		return location.Location{}, translateError("get location", err)
	}
	return l, nil
}

// List returns every Location, ordered by name for stable, human-useful
// output (see the index added on that column in the migration).
func (r *LocationRepository) List(ctx context.Context) ([]location.Location, error) {
	const query = `
		SELECT id, customer_id, name, type, status,
		       address1, address2, city, state, postal_code, country,
		       latitude, longitude, description, created_at, updated_at
		FROM locations
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list locations", err)
	}
	defer rows.Close()

	locations := []location.Location{}
	for rows.Next() {
		l, err := scanLocation(rows)
		if err != nil {
			return nil, translateError("scan location row", err)
		}
		locations = append(locations, l)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list locations", err)
	}

	return locations, nil
}

// ListByCustomerID returns every Location belonging to customerID,
// ordered by name for the same stable, human-useful reason List is.
func (r *LocationRepository) ListByCustomerID(ctx context.Context, customerID uuid.UUID) ([]location.Location, error) {
	const query = `
		SELECT id, customer_id, name, type, status,
		       address1, address2, city, state, postal_code, country,
		       latitude, longitude, description, created_at, updated_at
		FROM locations
		WHERE customer_id = $1
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query, customerID)
	if err != nil {
		return nil, translateError("list locations by customer", err)
	}
	defer rows.Close()

	locations := []location.Location{}
	for rows.Next() {
		l, err := scanLocation(rows)
		if err != nil {
			return nil, translateError("scan location row", err)
		}
		locations = append(locations, l)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list locations by customer", err)
	}

	return locations, nil
}

// Create inserts l and returns the persisted record.
//
// As with SiteRepository.Create, the repository assigns ID, CreatedAt,
// and UpdatedAt itself — any values already set on the input Location for
// those fields are ignored. A CustomerID that does not reference an
// existing Customer fails with an apperror.KindConflict error (see
// translateError).
func (r *LocationRepository) Create(ctx context.Context, l location.Location) (location.Location, error) {
	const query = `
		INSERT INTO locations (
			id, customer_id, name, type, status,
			address1, address2, city, state, postal_code, country,
			latitude, longitude, description, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $15)
		RETURNING id, customer_id, name, type, status,
		          address1, address2, city, state, postal_code, country,
		          latitude, longitude, description, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanLocation(r.db.QueryRow(ctx, query,
		r.ids.New(), l.CustomerID, l.Name, string(l.Type), string(l.Status),
		l.Address1, l.Address2, l.City, l.State, l.PostalCode, l.Country,
		l.Latitude, l.Longitude, l.Description, now))
	if err != nil {
		return location.Location{}, translateError("create location", err)
	}
	return created, nil
}

// Update overwrites the mutable fields of the Location identified by l.ID
// and returns the persisted record, or an apperror.KindNotFound error if
// it does not exist.
//
// CreatedAt cannot be altered through this method: the UPDATE statement
// below never assigns that column, and the RETURNING clause reports its
// true stored value regardless of what the input Location contained.
func (r *LocationRepository) Update(ctx context.Context, l location.Location) (location.Location, error) {
	const query = `
		UPDATE locations
		SET customer_id = $1, name = $2, type = $3, status = $4,
		    address1 = $5, address2 = $6, city = $7, state = $8, postal_code = $9, country = $10,
		    latitude = $11, longitude = $12, description = $13, updated_at = $14
		WHERE id = $15
		RETURNING id, customer_id, name, type, status,
		          address1, address2, city, state, postal_code, country,
		          latitude, longitude, description, created_at, updated_at
	`

	updated, err := scanLocation(r.db.QueryRow(ctx, query,
		l.CustomerID, l.Name, string(l.Type), string(l.Status),
		l.Address1, l.Address2, l.City, l.State, l.PostalCode, l.Country,
		l.Latitude, l.Longitude, l.Description, r.clock.Now(), l.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return location.Location{}, locationNotFound(l.ID)
		}
		return location.Location{}, translateError("update location", err)
	}
	return updated, nil
}

// Delete removes the Location identified by id, or returns an
// apperror.KindNotFound error if it does not exist.
func (r *LocationRepository) Delete(ctx context.Context, locationID uuid.UUID) error {
	const query = `DELETE FROM locations WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, locationID)
	if err != nil {
		return translateError("delete location", err)
	}
	if tag.RowsAffected() == 0 {
		return locationNotFound(locationID)
	}
	return nil
}

func locationNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("location %s not found", id))
}

// rowScanner is satisfied by both pgx.Row (QueryRow, a single row) and
// pgx.Rows (Query, iterated one row at a time via Next then Scan), so
// scanLocation backs Get/Create/Update and List alike.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanLocation(row rowScanner) (location.Location, error) {
	var (
		l       location.Location
		locType string
		status  string
	)
	err := row.Scan(
		&l.ID, &l.CustomerID, &l.Name, &locType, &status,
		&l.Address1, &l.Address2, &l.City, &l.State, &l.PostalCode, &l.Country,
		&l.Latitude, &l.Longitude, &l.Description, &l.CreatedAt, &l.UpdatedAt,
	)
	l.Type = location.LocationType(locType)
	l.Status = location.LocationStatus(status)
	return l, err
}
