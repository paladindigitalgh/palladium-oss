package ponport

import (
	"context"

	"github.com/google/uuid"
)

// PONPortRepository persists PONPorts. It follows the exact shape of
// every other repository in this codebase (see e.g. olt.OLTRepository):
// Get, List, Create, Update, Delete, with Create and Update returning
// the persisted entity so a caller sees anything the store sets (e.g.
// timestamps) without a second read.
//
// Nothing in this package implements it — no SQL, no migrations — so the
// domain has zero dependency on any storage technology. A concrete
// implementation (internal/ponport/postgres) satisfies it.
type PONPortRepository interface {
	Get(ctx context.Context, id uuid.UUID) (PONPort, error)
	List(ctx context.Context) ([]PONPort, error)
	Create(ctx context.Context, port PONPort) (PONPort, error)
	Update(ctx context.Context, port PONPort) (PONPort, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
