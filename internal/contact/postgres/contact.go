// Package postgres implements the Contact domain's ContactRepository
// against PostgreSQL using pgx directly — no ORM — following the exact
// pattern established by internal/location/postgres.LocationRepository.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/paladindigitalgh/palladium-oss/internal/contact"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// ContactRepository implements contact.ContactRepository against
// PostgreSQL. See internal/location/postgres/location.go for the
// reasoning behind depending on database.Querier and injecting
// clock/ids, which is not repeated here.
type ContactRepository struct {
	db    database.Querier
	clock clock.Clock
	ids   id.Generator
}

var _ contact.ContactRepository = (*ContactRepository)(nil)

// NewContactRepository builds a ContactRepository.
func NewContactRepository(db database.Querier, clock clock.Clock, ids id.Generator) *ContactRepository {
	return &ContactRepository{db: db, clock: clock, ids: ids}
}

// Get retrieves a Contact by ID, or an apperror.KindNotFound error if
// none exists.
func (r *ContactRepository) Get(ctx context.Context, contactID uuid.UUID) (contact.Contact, error) {
	const query = `
		SELECT id, customer_id, name, role, email, phone, status, description, created_at, updated_at
		FROM contacts
		WHERE id = $1
	`

	c, err := scanContact(r.db.QueryRow(ctx, query, contactID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return contact.Contact{}, contactNotFound(contactID)
		}
		return contact.Contact{}, translateError("get contact", err)
	}
	return c, nil
}

// List returns every Contact, ordered by name for stable, human-useful
// output (see the index added on that column in the migration).
func (r *ContactRepository) List(ctx context.Context) ([]contact.Contact, error) {
	const query = `
		SELECT id, customer_id, name, role, email, phone, status, description, created_at, updated_at
		FROM contacts
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list contacts", err)
	}
	defer rows.Close()

	contacts := []contact.Contact{}
	for rows.Next() {
		c, err := scanContact(rows)
		if err != nil {
			return nil, translateError("scan contact row", err)
		}
		contacts = append(contacts, c)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list contacts", err)
	}

	return contacts, nil
}

// Create inserts c and returns the persisted record.
//
// As with LocationRepository.Create, the repository assigns ID,
// CreatedAt, and UpdatedAt itself — any values already set on the input
// Contact for those fields are ignored. A CustomerID that does not
// reference an existing Customer fails with an apperror.KindConflict
// error (see translateError).
func (r *ContactRepository) Create(ctx context.Context, c contact.Contact) (contact.Contact, error) {
	const query = `
		INSERT INTO contacts (id, customer_id, name, role, email, phone, status, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
		RETURNING id, customer_id, name, role, email, phone, status, description, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := scanContact(r.db.QueryRow(ctx, query,
		r.ids.New(), c.CustomerID, c.Name, string(c.Role), c.Email, c.Phone, string(c.Status), c.Description, now))
	if err != nil {
		return contact.Contact{}, translateError("create contact", err)
	}
	return created, nil
}

// Update overwrites the mutable fields of the Contact identified by c.ID
// and returns the persisted record, or an apperror.KindNotFound error if
// it does not exist.
//
// CreatedAt cannot be altered through this method: the UPDATE statement
// below never assigns that column, and the RETURNING clause reports its
// true stored value regardless of what the input Contact contained.
func (r *ContactRepository) Update(ctx context.Context, c contact.Contact) (contact.Contact, error) {
	const query = `
		UPDATE contacts
		SET customer_id = $1, name = $2, role = $3, email = $4, phone = $5, status = $6, description = $7, updated_at = $8
		WHERE id = $9
		RETURNING id, customer_id, name, role, email, phone, status, description, created_at, updated_at
	`

	updated, err := scanContact(r.db.QueryRow(ctx, query,
		c.CustomerID, c.Name, string(c.Role), c.Email, c.Phone, string(c.Status), c.Description, r.clock.Now(), c.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return contact.Contact{}, contactNotFound(c.ID)
		}
		return contact.Contact{}, translateError("update contact", err)
	}
	return updated, nil
}

// Delete removes the Contact identified by id, or returns an
// apperror.KindNotFound error if it does not exist.
func (r *ContactRepository) Delete(ctx context.Context, contactID uuid.UUID) error {
	const query = `DELETE FROM contacts WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, contactID)
	if err != nil {
		return translateError("delete contact", err)
	}
	if tag.RowsAffected() == 0 {
		return contactNotFound(contactID)
	}
	return nil
}

func contactNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("contact %s not found", id))
}

// rowScanner is satisfied by both pgx.Row (QueryRow, a single row) and
// pgx.Rows (Query, iterated one row at a time via Next then Scan), so
// scanContact backs Get/Create/Update and List alike.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanContact(row rowScanner) (contact.Contact, error) {
	var (
		c      contact.Contact
		role   string
		status string
	)
	err := row.Scan(
		&c.ID, &c.CustomerID, &c.Name, &role, &c.Email, &c.Phone, &status, &c.Description, &c.CreatedAt, &c.UpdatedAt,
	)
	c.Role = contact.ContactRole(role)
	c.Status = contact.ContactStatus(status)
	return c, err
}
