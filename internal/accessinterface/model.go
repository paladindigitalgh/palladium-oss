// Package accessinterface models Palladium's Access Interface domain
// (v1): a logical interface presented on a PON port (see
// internal/ponport) — the point at which subscriber equipment actually
// attaches (see internal/accessattachment, one layer up). This package
// holds only the domain model, field validation, and the repository
// interface — no SQL, no migrations, no HTTP CRUD — mirroring
// internal/ponport's own package exactly.
//
// This package does not import internal/ponport. PONPortID is a bare
// uuid.UUID, not a reference to ponport.PONPort: the foreign key to
// pon_ports(id) is a database concept, enforced by
// internal/accessinterface/postgres and its migration, not a Go package
// dependency — the same reasoning internal/ponport/model.go documents
// for why PONPort does not import internal/olt.
//
// An AccessInterface describes that a logical interface exists on a PON
// port and what technology it speaks, nothing about how it is configured
// or who is attached to it. Per this milestone's explicit scope, this
// package does not model:
//
//   - VLANs or DBA profiles: those are logical network configuration
//     layered on top of an interface, not part of its identity — the
//     same reasoning internal/ponport/model.go gives for excluding
//     VLANs from PONPort.
//   - Optics: a physical/signal-level concern about the PON port itself,
//     not the logical interface riding on it.
//   - Authentication or bandwidth: provisioning-time concerns (see
//     docs/ARCHITECTURE.md's Provisioning domain) that act on an
//     interface because it exists, not facts an interface record itself
//     holds.
//   - Utilization: a real-time/monitoring question explicitly out of
//     scope for Palladium entirely (see CLAUDE.md's "Palladium is NOT ...
//     a monitoring platform").
//
// Which subscriber equipment is attached to an interface, and when, is
// internal/accessattachment's responsibility, not this package's — an
// AccessInterface exists independently of anything being attached to it,
// the same "resources exist independently of customers" principle
// CLAUDE.md states for inventory generally.
package accessinterface

import (
	"time"

	"github.com/google/uuid"
)

// AccessInterface is a logical interface on a PON port.
type AccessInterface struct {
	ID          uuid.UUID
	PONPortID   uuid.UUID
	Technology  Technology
	Name        string
	Status      Status
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
