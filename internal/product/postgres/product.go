// Package postgres implements the Product domain's ProductRepository
// against PostgreSQL using pgx directly — no ORM — following the exact
// pattern established by internal/location/postgres.LocationRepository
// (the closest precedent: a required foreign key to a sibling domain).
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
	"github.com/paladindigitalgh/palladium-oss/internal/product"
)

// ProductRepository implements product.ProductRepository against
// PostgreSQL. See internal/inventory/postgres/site.go for the reasoning
// behind depending on database.Querier and injecting clock/ids, which is
// not repeated here.
type ProductRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ product.ProductRepository = (*ProductRepository)(nil)

// NewProductRepository builds a ProductRepository.
func NewProductRepository(db database.Querier, clock clock.Clock, ids id.Generator) *ProductRepository {
	return &ProductRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves a Product by ID, or an apperror.KindNotFound error if
// none exists.
func (r *ProductRepository) Get(ctx context.Context, productID uuid.UUID) (product.Product, error) {
	const query = `
		SELECT id, catalog_id, name, category, status, description, created_at, updated_at
		FROM products
		WHERE id = $1
	`

	p, err := scanProduct(r.db.QueryRow(ctx, query, productID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return product.Product{}, productNotFound(productID)
		}
		return product.Product{}, translateError("get product", err)
	}
	return p, nil
}

// List returns every Product, ordered by name for stable, human-useful
// output (see the index added on that column in the migration).
func (r *ProductRepository) List(ctx context.Context) ([]product.Product, error) {
	const query = `
		SELECT id, catalog_id, name, category, status, description, created_at, updated_at
		FROM products
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list products", err)
	}
	defer rows.Close()

	products := []product.Product{}
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, translateError("scan product row", err)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list products", err)
	}

	return products, nil
}

// Create inserts p and returns the persisted record.
//
// As with LocationRepository.Create, the repository assigns ID, CreatedAt,
// and UpdatedAt itself — any values already set on the input Product for
// those fields are ignored. A CatalogID that does not reference an
// existing ProductCatalog fails with an apperror.KindConflict error (see
// translateError).
func (r *ProductRepository) Create(ctx context.Context, p product.Product) (product.Product, error) {
	const query = `
		INSERT INTO products (id, catalog_id, name, category, status, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		RETURNING id, catalog_id, name, category, status, description, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanProduct(r.db.QueryRow(ctx, query,
		r.ids.New(), p.CatalogID, p.Name, string(p.Category), string(p.Status), p.Description, now))
	if err != nil {
		return product.Product{}, translateError("create product", err)
	}
	return created, nil
}

// Update overwrites the mutable fields of the Product identified by p.ID
// and returns the persisted record, or an apperror.KindNotFound error if
// it does not exist.
//
// CreatedAt cannot be altered through this method: the UPDATE statement
// below never assigns that column, and the RETURNING clause reports its
// true stored value regardless of what the input Product contained.
func (r *ProductRepository) Update(ctx context.Context, p product.Product) (product.Product, error) {
	const query = `
		UPDATE products
		SET catalog_id = $1, name = $2, category = $3, status = $4, description = $5, updated_at = $6
		WHERE id = $7
		RETURNING id, catalog_id, name, category, status, description, created_at, updated_at
	`

	updated, err := scanProduct(r.db.QueryRow(ctx, query,
		p.CatalogID, p.Name, string(p.Category), string(p.Status), p.Description, r.clock.Now(), p.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return product.Product{}, productNotFound(p.ID)
		}
		return product.Product{}, translateError("update product", err)
	}
	return updated, nil
}

// Delete removes the Product identified by id, or returns an
// apperror.KindNotFound error if it does not exist.
func (r *ProductRepository) Delete(ctx context.Context, productID uuid.UUID) error {
	const query = `DELETE FROM products WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, productID)
	if err != nil {
		return translateError("delete product", err)
	}
	if tag.RowsAffected() == 0 {
		return productNotFound(productID)
	}
	return nil
}

func productNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("product %s not found", id))
}

// rowScanner is satisfied by both pgx.Row (QueryRow, a single row) and
// pgx.Rows (Query, iterated one row at a time via Next then Scan), so
// scanProduct backs Get/Create/Update and List alike.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanProduct(row rowScanner) (product.Product, error) {
	var (
		p        product.Product
		category string
		status   string
	)
	err := row.Scan(&p.ID, &p.CatalogID, &p.Name, &category, &status, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	p.Category = product.ProductCategory(category)
	p.Status = product.ProductStatus(status)
	return p, err
}
