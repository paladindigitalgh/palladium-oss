// Package postgres implements the Provider domain's ProviderRepository
// against PostgreSQL using pgx directly — no ORM — following the exact
// pattern established by internal/serviceprofile/postgres.ServiceProfileRepository.
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
	"github.com/paladindigitalgh/palladium-oss/internal/provider"
)

// ProviderRepository implements provider.ProviderRepository against
// PostgreSQL. See internal/inventory/postgres/site.go for the reasoning
// behind depending on database.Querier and injecting clock/ids, which is
// not repeated here.
type ProviderRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ provider.ProviderRepository = (*ProviderRepository)(nil)

// NewProviderRepository builds a ProviderRepository.
func NewProviderRepository(db database.Querier, clock clock.Clock, ids id.Generator) *ProviderRepository {
	return &ProviderRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves a Provider by ID, or an apperror.KindNotFound error if
// none exists.
func (r *ProviderRepository) Get(ctx context.Context, providerID uuid.UUID) (provider.Provider, error) {
	const query = `
		SELECT id, name, description, status, created_at, updated_at
		FROM providers
		WHERE id = $1
	`

	p, err := scanProvider(r.db.QueryRow(ctx, query, providerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return provider.Provider{}, providerNotFound(providerID)
		}
		return provider.Provider{}, translateError("get provider", err)
	}
	return p, nil
}

// List returns every Provider, ordered by name for stable, human-useful
// output (see the index added on that column in the migration).
func (r *ProviderRepository) List(ctx context.Context) ([]provider.Provider, error) {
	const query = `
		SELECT id, name, description, status, created_at, updated_at
		FROM providers
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list providers", err)
	}
	defer rows.Close()

	providers := []provider.Provider{}
	for rows.Next() {
		p, err := scanProvider(rows)
		if err != nil {
			return nil, translateError("scan provider row", err)
		}
		providers = append(providers, p)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list providers", err)
	}

	return providers, nil
}

// Create inserts p and returns the persisted record.
//
// The repository assigns ID, CreatedAt, and UpdatedAt itself — any
// values already set on the input Provider for those fields are
// ignored. Status is taken from the input exactly as given; the
// repository has no business logic and does not decide it.
func (r *ProviderRepository) Create(ctx context.Context, p provider.Provider) (provider.Provider, error) {
	const query = `
		INSERT INTO providers (id, name, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING id, name, description, status, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanProvider(r.db.QueryRow(ctx, query,
		r.ids.New(), p.Name, p.Description, string(p.Status), now))
	if err != nil {
		return provider.Provider{}, translateError("create provider", err)
	}
	return created, nil
}

// Update overwrites the mutable fields (Name, Description, Status) of
// the Provider identified by p.ID and returns the persisted record, or
// an apperror.KindNotFound error if it does not exist.
//
// CreatedAt cannot be altered through this method: the UPDATE statement
// below never assigns that column, and the RETURNING clause reports its
// true stored value regardless of what the input Provider contained.
func (r *ProviderRepository) Update(ctx context.Context, p provider.Provider) (provider.Provider, error) {
	const query = `
		UPDATE providers
		SET name = $1, description = $2, status = $3, updated_at = $4
		WHERE id = $5
		RETURNING id, name, description, status, created_at, updated_at
	`

	updated, err := scanProvider(r.db.QueryRow(ctx, query,
		p.Name, p.Description, string(p.Status), r.clock.Now(), p.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return provider.Provider{}, providerNotFound(p.ID)
		}
		return provider.Provider{}, translateError("update provider", err)
	}
	return updated, nil
}

// Delete removes the Provider identified by id, or returns an
// apperror.KindNotFound error if it does not exist. If a Product still
// references this Provider, the delete fails with apperror.KindConflict
// instead (see errors.go's translateError and the ON DELETE RESTRICT
// foreign key added to products by this domain's migration).
func (r *ProviderRepository) Delete(ctx context.Context, providerID uuid.UUID) error {
	const query = `DELETE FROM providers WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, providerID)
	if err != nil {
		return translateError("delete provider", err)
	}
	if tag.RowsAffected() == 0 {
		return providerNotFound(providerID)
	}
	return nil
}

func providerNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("provider %s not found", id))
}

// rowScanner is satisfied by both pgx.Row (QueryRow, a single row) and
// pgx.Rows (Query, iterated one row at a time via Next then Scan), so
// scanProvider backs Get/Create/Update and List alike.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanProvider(row rowScanner) (provider.Provider, error) {
	var (
		p      provider.Provider
		status string
	)
	err := row.Scan(&p.ID, &p.Name, &p.Description, &status, &p.CreatedAt, &p.UpdatedAt)
	p.Status = provider.Status(status)
	return p, err
}
