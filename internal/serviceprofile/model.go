// Package serviceprofile models Palladium's Service Profile domain
// (v1): the named, reusable description of a Service's operational
// intent — e.g. "Residential Internet" or "Business Ethernet" — that a
// Service (see internal/service) is required to reference. This package
// holds only the domain model, field validation, and the repository
// interface — no SQL, no migrations, no HTTP CRUD — mirroring
// internal/catalog's own package exactly.
//
// A ServiceProfile describes what kind of service is being delivered,
// never how it is configured on the network. Per this milestone's
// explicit scope, this package does not model:
//
//   - Bandwidth fields, QoS, or DBA profiles: those are provisioning-time
//     network configuration, layered on top of a profile, not folded
//     into it.
//   - VLANs: those belong to a future Network domain (see
//     docs/ARCHITECTURE.md's Network domain).
//   - Connector mappings or vendor identifiers: per CLAUDE.md's Plugin
//     Philosophy, anything vendor-specific belongs in a plugin (see
//     internal/provisioning/connectors), never in a core domain type
//     like this one.
//   - Provisioning logic: nothing here configures a device or a network.
//     A future Provisioning domain will act because a Service references
//     a ServiceProfile, not the other way around.
//
// Everything above is a real feature some future milestone will add. None
// of it is implied by what exists today.
package serviceprofile

import (
	"time"

	"github.com/google/uuid"
)

// ServiceProfile is a named, reusable description of a Service's
// operational intent.
type ServiceProfile struct {
	ID          uuid.UUID
	Name        string
	Status      Status
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
