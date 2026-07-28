// Package postgres implements the Catalog domain's CatalogRepository
// against PostgreSQL using pgx directly — no ORM — following the exact
// pattern established by internal/customer/postgres.CustomerRepository.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/paladindigitalgh/palladium-oss/internal/catalog"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// CatalogRepository implements catalog.CatalogRepository against
// PostgreSQL. See internal/inventory/postgres/site.go for the reasoning
// behind depending on database.Querier and injecting clock/ids, which is
// not repeated here.
type CatalogRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ catalog.CatalogRepository = (*CatalogRepository)(nil)

// NewCatalogRepository builds a CatalogRepository.
func NewCatalogRepository(db database.Querier, clock clock.Clock, ids id.Generator) *CatalogRepository {
	return &CatalogRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves a ProductCatalog by ID, or an apperror.KindNotFound error
// if none exists.
func (r *CatalogRepository) Get(ctx context.Context, catalogID uuid.UUID) (catalog.ProductCatalog, error) {
	const query = `
		SELECT id, name, description, status, created_at, updated_at
		FROM catalogs
		WHERE id = $1
	`

	c, err := scanCatalog(r.db.QueryRow(ctx, query, catalogID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return catalog.ProductCatalog{}, catalogNotFound(catalogID)
		}
		return catalog.ProductCatalog{}, translateError("get catalog", err)
	}
	return c, nil
}

// List returns every ProductCatalog, ordered by name for stable,
// human-useful output (see the index added on that column in the
// migration).
func (r *CatalogRepository) List(ctx context.Context) ([]catalog.ProductCatalog, error) {
	const query = `
		SELECT id, name, description, status, created_at, updated_at
		FROM catalogs
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list catalogs", err)
	}
	defer rows.Close()

	catalogs := []catalog.ProductCatalog{}
	for rows.Next() {
		c, err := scanCatalog(rows)
		if err != nil {
			return nil, translateError("scan catalog row", err)
		}
		catalogs = append(catalogs, c)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list catalogs", err)
	}

	return catalogs, nil
}

// Create inserts c and returns the persisted record.
//
// As with CustomerRepository.Create, the repository assigns ID, CreatedAt,
// and UpdatedAt itself — any values already set on the input ProductCatalog
// for those fields are ignored. Status is taken from the input exactly as
// given; the repository has no business logic and does not decide it.
func (r *CatalogRepository) Create(ctx context.Context, c catalog.ProductCatalog) (catalog.ProductCatalog, error) {
	const query = `
		INSERT INTO catalogs (id, name, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING id, name, description, status, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanCatalog(r.db.QueryRow(ctx, query,
		r.ids.New(), c.Name, c.Description, string(c.Status), now))
	if err != nil {
		return catalog.ProductCatalog{}, translateError("create catalog", err)
	}
	return created, nil
}

// Update overwrites the mutable fields (Name, Description, Status) of the
// ProductCatalog identified by c.ID and returns the persisted record, or
// an apperror.KindNotFound error if it does not exist.
//
// CreatedAt cannot be altered through this method: the UPDATE statement
// below never assigns that column, and the RETURNING clause reports its
// true stored value regardless of what the input ProductCatalog contained.
func (r *CatalogRepository) Update(ctx context.Context, c catalog.ProductCatalog) (catalog.ProductCatalog, error) {
	const query = `
		UPDATE catalogs
		SET name = $1, description = $2, status = $3, updated_at = $4
		WHERE id = $5
		RETURNING id, name, description, status, created_at, updated_at
	`

	updated, err := scanCatalog(r.db.QueryRow(ctx, query,
		c.Name, c.Description, string(c.Status), r.clock.Now(), c.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return catalog.ProductCatalog{}, catalogNotFound(c.ID)
		}
		return catalog.ProductCatalog{}, translateError("update catalog", err)
	}
	return updated, nil
}

// Delete removes the ProductCatalog identified by id, or returns an
// apperror.KindNotFound error if it does not exist. If a Product still
// references this catalog, the delete fails with apperror.KindConflict
// instead (see errors.go's translateError and the ON DELETE RESTRICT
// foreign key in database/migrations/00012_product_products.sql).
func (r *CatalogRepository) Delete(ctx context.Context, catalogID uuid.UUID) error {
	const query = `DELETE FROM catalogs WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, catalogID)
	if err != nil {
		return translateError("delete catalog", err)
	}
	if tag.RowsAffected() == 0 {
		return catalogNotFound(catalogID)
	}
	return nil
}

func catalogNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("catalog %s not found", id))
}

// rowScanner is satisfied by both pgx.Row (QueryRow, a single row) and
// pgx.Rows (Query, iterated one row at a time via Next then Scan), so
// scanCatalog backs Get/Create/Update and List alike.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanCatalog(row rowScanner) (catalog.ProductCatalog, error) {
	var (
		c      catalog.ProductCatalog
		status string
	)
	err := row.Scan(&c.ID, &c.Name, &c.Description, &status, &c.CreatedAt, &c.UpdatedAt)
	c.Status = catalog.CatalogStatus(status)
	return c, err
}
