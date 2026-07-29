package authentication

import (
	"context"

	"github.com/google/uuid"
)

// AuthenticationRepository persists Authentications. It follows the
// exact shape of every other repository in this codebase (see e.g.
// internal/catalog.CatalogRepository): Get, List, Create, Update,
// Delete, with Create and Update returning the persisted entity so a
// caller sees anything the store sets (e.g. timestamps) without a second
// read.
//
// Nothing in this package implements it — no SQL, no migrations, no
// encryption — so the domain has zero dependency on any storage
// technology or on internal/platform/encryption. A concrete
// implementation (internal/authentication/postgres) satisfies it, and is
// the one place encryption enters the picture at all (see that
// package's own doc comment).
type AuthenticationRepository interface {
	Get(ctx context.Context, id uuid.UUID) (Authentication, error)
	List(ctx context.Context) ([]Authentication, error)
	Create(ctx context.Context, auth Authentication) (Authentication, error)
	Update(ctx context.Context, auth Authentication) (Authentication, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
