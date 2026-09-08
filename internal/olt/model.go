// Package olt models Palladium's OLT domain (v1): a physical Optical
// Line Terminal, identified by its chassis type and management address,
// belonging to one AccessNetwork (see internal/accessnetwork). This
// package holds only the domain model, field validation, and the
// repository interface — no SQL, no migrations, no HTTP CRUD —
// mirroring internal/product's own package exactly.
//
// This package does not import internal/accessnetwork or
// internal/oltmodel. AccessNetworkID and OLTModelID are bare uuid.UUID
// values, not references to accessnetwork.AccessNetwork or
// oltmodel.OLTModel: the foreign keys to access_networks(id) and
// olt_models(id) are database concepts, enforced by internal/olt/postgres
// and its migrations, not a Go package dependency — the same reasoning
// internal/product/model.go documents for why Product does not import
// internal/catalog.
//
// # Vendor moved to OLTModel
//
// OLT used to carry its own Vendor (a closed enum) and free-text Model
// fields directly. Both moved to internal/oltmodel.OLTModel, referenced
// here as OLTModelID: a model name like "C16" already implies its
// vendor, and storing Vendor in two places (here and on OLTModel) risked
// the two disagreeing. OLTModelID is required, not nullable — unlike
// ConnectionProfileID below — because internal/olt/service.OLTService's
// auto-port-creation on Create depends on it always resolving to a real
// OLTModel with a known PONPortCount; there is no legitimate "OLT
// recorded before its chassis type is known" state the way there is for
// connection credentials.
//
// This package also does not reference internal/inventory.Device, per
// this milestone's explicit instruction. A real OLT is, physically, a
// piece of inventory — but wiring that relationship in now would be
// guessing at a shape (does an OLT have exactly one Device? Can a
// chassis span several?) this milestone was explicitly told to leave for
// later. An OLT record here stands alone.
//
// An OLT describes that a physical Optical Line Terminal exists and
// where it can be managed, never how it is configured or what it is
// currently doing. Per this milestone's explicit scope:
//
//   - No ONU attachment, no splitters, no optics, no VLANs, no bandwidth
//     profiles: those describe what an OLT's PON ports (see
//     internal/ponport) are actually doing, not the OLT itself.
//   - No monitoring, no alarms, no provisioning, no vendor APIs: nothing
//     here talks to a real OLT or reports its live state. See
//     internal/accessnetwork/model.go's package doc comment for the same
//     boundary drawn one level up.
//
// Everything above is a real feature some future milestone will add. None
// of it is implied by what exists today.
//
// # Connection Profile
//
// A later milestone (Authentication and Connection Profile
// infrastructure) added ConnectionProfileID, replacing what would
// otherwise have been direct authentication fields on OLT itself
// (username, password, and the like) with a reference to a reusable
// internal/connectionprofile.ConnectionProfile — which in turn
// references a reusable internal/authentication.Authentication. That
// milestone's own instruction was explicit: "Remove any direct
// authentication fields if present." None existed on OLT before this —
// ManagementIPAddress records where to reach an OLT, never how to log
// in to it — so there was nothing to remove; ConnectionProfileID is a
// pure addition.
package olt

import (
	"time"

	"github.com/google/uuid"
)

// OLT is a single Optical Line Terminal within an AccessNetwork.
//
// ConnectionProfileID is nullable (*uuid.UUID), for the same reason
// connectionprofile.ConnectionProfile.AuthenticationID is: this
// milestone does not require every OLT to already have a connection
// profile bound to it — an OLT recorded before its management
// credentials are configured is a legitimate, ordinary state, the same
// way a Rack can exist before it is installed in a Room (see
// inventory.Rack.RoomID). The foreign key to connection_profiles(id) is
// a database concept, enforced by internal/olt/postgres and its
// migration, not a Go package dependency — this package does not import
// internal/connectionprofile, the same reasoning it already documents
// above for not importing internal/accessnetwork.
type OLT struct {
	ID                  uuid.UUID
	AccessNetworkID     uuid.UUID
	Name                string
	OLTModelID          uuid.UUID
	ManagementIPAddress string
	ConnectionProfileID *uuid.UUID
	Description         string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
