package connectionprofile

import (
	"context"

	"github.com/google/uuid"
)

// ConnectionProfileRepository persists ConnectionProfiles. It follows
// the exact shape of every other repository in this codebase (see e.g.
// internal/catalog.CatalogRepository): Get, List, Create, Update,
// Delete, with Create and Update returning the persisted entity so a
// caller sees anything the store sets (e.g. timestamps) without a second
// read.
//
// Nothing in this package implements it — no SQL, no migrations — so the
// domain has zero dependency on any storage technology. A concrete
// implementation (internal/connectionprofile/postgres) satisfies it.
type ConnectionProfileRepository interface {
	Get(ctx context.Context, id uuid.UUID) (ConnectionProfile, error)
	List(ctx context.Context) ([]ConnectionProfile, error)
	Create(ctx context.Context, profile ConnectionProfile) (ConnectionProfile, error)
	Update(ctx context.Context, profile ConnectionProfile) (ConnectionProfile, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
