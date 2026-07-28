// Package accessnetwork models Palladium's Access Network domain (v1):
// the physical PON access network as a named, independently-lifecycled
// grouping — e.g. "North Region GPON" versus "Downtown XGS-PON" — under
// which OLTs (see internal/olt) are organized. This package holds only
// the domain model, field validation, and the repository interface — no
// SQL, no migrations, no HTTP CRUD — mirroring internal/catalog's own
// package exactly.
//
// This is deliberately not provisioning, optics, bandwidth, or
// monitoring. Per this milestone's explicit scope:
//
//   - No ONU attachment, no splitters, no optics, no VLANs, no bandwidth
//     profiles: an AccessNetwork only groups OLTs. Everything below an
//     OLT's PON ports (see internal/ponport) — splitters, ONUs, the fiber
//     plant itself — is a real future domain, not implied by anything
//     here.
//   - No monitoring, no alarms: link state, optical power, and uptime
//     belong in a monitoring platform (see docs/ARCHITECTURE.md's "What
//     Palladium Is Not" — Zabbix, LibreNMS, Prometheus), never in this
//     package.
//   - No provisioning, no vendor APIs: nothing here talks to a real OLT.
//     A future Provisioning Engine connector (see
//     internal/provisioning/connectors) will act on an OLT, not the
//     other way around.
//
// Everything above is a real feature some future milestone will add. None
// of it is implied by what exists today.
package accessnetwork

import (
	"time"

	"github.com/google/uuid"
)

// AccessNetwork groups the OLTs (see internal/olt) that make up one
// physical PON access network.
type AccessNetwork struct {
	ID          uuid.UUID
	Name        string
	Status      AccessNetworkStatus
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
