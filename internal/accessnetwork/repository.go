package accessnetwork

import (
	"context"

	"github.com/google/uuid"
)

// AccessNetworkRepository persists AccessNetworks. It follows the exact
// shape of every other repository in this codebase (see e.g.
// internal/catalog.CatalogRepository): Get, List, Create, Update,
// Delete, with Create and Update returning the persisted entity so a
// caller sees anything the store sets (e.g. timestamps) without a second
// read.
//
// Nothing in this package implements it — no SQL, no migrations — so the
// domain has zero dependency on any storage technology. A concrete
// implementation (internal/accessnetwork/postgres) satisfies it.
type AccessNetworkRepository interface {
	Get(ctx context.Context, id uuid.UUID) (AccessNetwork, error)
	List(ctx context.Context) ([]AccessNetwork, error)
	Create(ctx context.Context, network AccessNetwork) (AccessNetwork, error)
	Update(ctx context.Context, network AccessNetwork) (AccessNetwork, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
