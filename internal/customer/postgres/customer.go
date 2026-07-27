// Package postgres implements the Customer domain's CustomerRepository
// against PostgreSQL using pgx directly — no ORM — following the exact
// pattern established by internal/inventory/postgres.SiteRepository.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/paladindigitalgh/palladium-oss/internal/customer"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// CustomerRepository implements customer.CustomerRepository against
// PostgreSQL. See internal/inventory/postgres/site.go for the reasoning
// behind depending on database.Querier and injecting clock/ids, which is
// not repeated here.
type CustomerRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ customer.CustomerRepository = (*CustomerRepository)(nil)

// NewCustomerRepository builds a CustomerRepository.
func NewCustomerRepository(db database.Querier, clock clock.Clock, ids id.Generator) *CustomerRepository {
	return &CustomerRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves a Customer by ID, or an apperror.KindNotFound error if
// none exists.
func (r *CustomerRepository) Get(ctx context.Context, customerID uuid.UUID) (customer.Customer, error) {
	const query = `
		SELECT id, name, customer_type, status, description, created_at, updated_at
		FROM customers
		WHERE id = $1
	`

	c, err := scanCustomer(r.db.QueryRow(ctx, query, customerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return customer.Customer{}, customerNotFound(customerID)
		}
		return customer.Customer{}, translateError("get customer", err)
	}
	return c, nil
}

// List returns every Customer, ordered by name for stable, human-useful
// output (see the index added on that column in the migration).
func (r *CustomerRepository) List(ctx context.Context) ([]customer.Customer, error) {
	const query = `
		SELECT id, name, customer_type, status, description, created_at, updated_at
		FROM customers
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list customers", err)
	}
	defer rows.Close()

	customers := []customer.Customer{}
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, translateError("scan customer row", err)
		}
		customers = append(customers, c)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list customers", err)
	}

	return customers, nil
}

// Create inserts c and returns the persisted record.
//
// As with SiteRepository.Create, the repository assigns ID, CreatedAt,
// and UpdatedAt itself — any values already set on the input Customer for
// those fields are ignored. CustomerType and Status are taken from the
// input exactly as given; the repository has no business logic and does
// not decide either.
func (r *CustomerRepository) Create(ctx context.Context, c customer.Customer) (customer.Customer, error) {
	const query = `
		INSERT INTO customers (id, name, customer_type, status, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		RETURNING id, name, customer_type, status, description, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanCustomer(r.db.QueryRow(ctx, query,
		r.ids.New(), c.Name, string(c.CustomerType), string(c.Status), c.Description, now))
	if err != nil {
		return customer.Customer{}, translateError("create customer", err)
	}
	return created, nil
}

// Update overwrites the mutable fields (Name, CustomerType, Status,
// Description) of the Customer identified by c.ID and returns the
// persisted record, or an apperror.KindNotFound error if it does not
// exist.
//
// CreatedAt cannot be altered through this method: the UPDATE statement
// below never assigns that column, and the RETURNING clause reports its
// true stored value regardless of what the input Customer contained.
func (r *CustomerRepository) Update(ctx context.Context, c customer.Customer) (customer.Customer, error) {
	const query = `
		UPDATE customers
		SET name = $1, customer_type = $2, status = $3, description = $4, updated_at = $5
		WHERE id = $6
		RETURNING id, name, customer_type, status, description, created_at, updated_at
	`

	updated, err := scanCustomer(r.db.QueryRow(ctx, query,
		c.Name, string(c.CustomerType), string(c.Status), c.Description, r.clock.Now(), c.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return customer.Customer{}, customerNotFound(c.ID)
		}
		return customer.Customer{}, translateError("update customer", err)
	}
	return updated, nil
}

// Delete removes the Customer identified by id, or returns an
// apperror.KindNotFound error if it does not exist.
func (r *CustomerRepository) Delete(ctx context.Context, customerID uuid.UUID) error {
	const query = `DELETE FROM customers WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, customerID)
	if err != nil {
		return translateError("delete customer", err)
	}
	if tag.RowsAffected() == 0 {
		return customerNotFound(customerID)
	}
	return nil
}

func customerNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("customer %s not found", id))
}

// rowScanner is satisfied by both pgx.Row (QueryRow, a single row) and
// pgx.Rows (Query, iterated one row at a time via Next then Scan), so
// scanCustomer backs Get/Create/Update and List alike.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanCustomer(row rowScanner) (customer.Customer, error) {
	var (
		c            customer.Customer
		customerType string
		status       string
	)
	err := row.Scan(&c.ID, &c.Name, &customerType, &status, &c.Description, &c.CreatedAt, &c.UpdatedAt)
	c.CustomerType = customer.CustomerType(customerType)
	c.Status = customer.CustomerStatus(status)
	return c, err
}
