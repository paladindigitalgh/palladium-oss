// Package ponport models Palladium's PON Port domain (v1): a single
// numbered PON port on an OLT (see internal/olt). This package holds
// only the domain model, field validation, and the repository interface
// — no SQL, no migrations, no HTTP CRUD — mirroring internal/olt's own
// package exactly.
//
// This package does not import internal/olt. OLTID is a bare uuid.UUID,
// not a reference to olt.OLT: the foreign key to olts(id) is a database
// concept, enforced by internal/ponport/postgres and its migration, not
// a Go package dependency — the same reasoning internal/olt/model.go
// documents for why OLT does not import internal/accessnetwork.
//
// A PONPort describes that a numbered port exists on an OLT, nothing
// about what is happening on it. Per this milestone's explicit scope,
// this package does not model:
//
//   - Split ratio, optics, or utilization: those describe the fiber
//     plant and the physical signal on it, not the port's identity.
//   - VLANs: those belong to a future Network domain (see
//     docs/ARCHITECTURE.md's Network domain), layered on top of a port,
//     not folded into it.
//   - Status: unlike accessnetwork.AccessNetwork, a PON port has no
//     lifecycle field here at all — "is this port up," "is it
//     provisioned," and "is it in alarm" are all real-time or
//     provisioning questions this milestone explicitly excludes (see
//     "No monitoring, no alarms, no provisioning" in
//     internal/olt/model.go's package doc comment, one level up), not
//     static facts about the port a record like this could hold.
//   - Subscribers: which ONUs or customers sit behind a splitter on this
//     port is a real future relationship (ONU attachment, explicitly out
//     of scope), not implied by a bare port number existing.
//
// Everything above is a real feature some future milestone will add. None
// of it is implied by what exists today.
package ponport

import (
	"time"

	"github.com/google/uuid"
)

// PONPort is a single numbered PON port on an OLT.
type PONPort struct {
	ID          uuid.UUID
	OLTID       uuid.UUID
	PortNumber  int
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
