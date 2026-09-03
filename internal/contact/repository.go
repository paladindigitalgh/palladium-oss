package contact

import (
	"context"

	"github.com/google/uuid"
)

// ContactRepository persists Contacts. It follows the exact shape of
// every other repository in this codebase: Get, List, Create, Update,
// Delete, with Create and Update returning the persisted entity so a
// caller sees anything the store sets (e.g. timestamps) without a second
// read.
//
// Nothing in this package implements it — no SQL, no migrations — so the
// domain has zero dependency on any storage technology. A concrete
// implementation (internal/contact/postgres) satisfies it.
type ContactRepository interface {
	Get(ctx context.Context, id uuid.UUID) (Contact, error)
	List(ctx context.Context) ([]Contact, error)
	Create(ctx context.Context, c Contact) (Contact, error)
	Update(ctx context.Context, c Contact) (Contact, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
