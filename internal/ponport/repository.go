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
// GetByOLTIDAndPortNumber exists specifically to support
// internal/provisioning/kontron/service.AuthorizeAndCreateDeviceService's
// own "find or create the PON port a freshly authorized ONU's interface
// belongs to" step: a real OLT authorization already names an OLT and a
// port number (see kontron.ParsePortNumber), and this is the query that
// answers "does Palladium already have a PONPort record for that pair"
// directly, in one round trip, rather than every caller fetching List
// and filtering client-side. Like Get, it returns an apperror.KindNotFound
// error when no PONPort exists for (oltID, portNumber) — the expected,
// common case the first time a given port is ever authorized on.
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
	GetByOLTIDAndPortNumber(ctx context.Context, oltID uuid.UUID, portNumber int) (PONPort, error)
}
