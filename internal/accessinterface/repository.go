package accessinterface

import (
	"context"

	"github.com/google/uuid"
)

// AccessInterfaceRepository persists AccessInterfaces. It follows the
// exact shape of every other repository in this codebase (see e.g.
// ponport.PONPortRepository): Get, List, Create, Update, Delete, with
// Create and Update returning the persisted entity so a caller sees
// anything the store sets (e.g. timestamps) without a second read.
//
// Nothing in this package implements it — no SQL, no migrations — so the
// domain has zero dependency on any storage technology. A concrete
// implementation (internal/accessinterface/postgres) satisfies it.
type AccessInterfaceRepository interface {
	Get(ctx context.Context, id uuid.UUID) (AccessInterface, error)
	List(ctx context.Context) ([]AccessInterface, error)
	Create(ctx context.Context, iface AccessInterface) (AccessInterface, error)
	Update(ctx context.Context, iface AccessInterface) (AccessInterface, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
