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
// GetByOLTIDAndName exists specifically to support
// internal/serviceequipment/service.ServiceEquipmentService's own
// "does a newly-active ServiceEquipment's Device have a known Access
// Interface to auto-attach to" step: internal/onuauthorization records a
// Device's real OLT and (Kontron-shaped) interface string, and this is
// the query that answers "does Palladium already have an AccessInterface
// for that pair" directly — joining through PON Port, since an
// AccessInterface only ever names its PONPortID, not an OLTID directly
// — rather than every caller resolving the PON port itself first. Like
// Get, it returns an apperror.KindNotFound error when none exists.
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
	GetByOLTIDAndName(ctx context.Context, oltID uuid.UUID, name string) (AccessInterface, error)
}
