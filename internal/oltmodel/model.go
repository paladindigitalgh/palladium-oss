// Package oltmodel models Palladium's OLT Model domain (v1): an
// Administration-managed catalog entry naming a physical OLT chassis
// type — e.g. "Kontron C16" — and, critically, how many PON ports that
// chassis type has. This package holds only the domain model, field
// validation, and the repository interface — no SQL, no migrations, no
// HTTP CRUD — mirroring internal/provider's own package exactly.
//
// This package exists to answer one question without hardcoding a
// vendor's product line into Go code: "how many PON ports does this OLT
// have?" Per CLAUDE.md's Plugin Philosophy ("The core system must never
// contain Kontron-, Nokia-, Calix-, Adtran-, MikroTik-, or vendor-specific
// logic"), that answer cannot live in a switch statement keyed on
// Vendor + model name — it has to be data an operator enters once, the
// same way internal/provider and internal/catalog are Administration-
// managed reference data other domains point to rather than code. An
// OLTModel is that data: a single row saying "a Kontron C16 has 16 PON
// ports," created once by an operator and referenced by every OLT of
// that type from then on (see internal/olt.OLT.OLTModelID).
//
// Vendor (see vendor.go) lives here, not on internal/olt.OLT itself: it
// moved from OLT to this package because a model name like "C16"
// already implies its vendor, and storing Vendor in two places (OLT and
// OLTModel) risked the two disagreeing — exactly the kind of divergence
// this catalog exists to prevent. See internal/olt/model.go's package
// doc comment for the OLT side of that change.
//
// Per this milestone's explicit scope, this package does not model:
//
//   - Optics, uplinks, chassis slot layout, or any other physical
//     characteristic beyond a PON port count: those are real future
//     attributes some future milestone may add, not implied by what
//     exists today.
//   - Firmware versions or vendor-specific configuration: those belong
//     to a future Firmware Catalog (see docs/03-DOMAIN-MODEL.md section
//     18's Future Domain Expansion list), not this catalog.
//   - Status (Active/Inactive): unlike Provider or Catalog, an OLTModel
//     has no lifecycle field in this milestone — a catalog entry
//     referenced by a real OLT (ON DELETE RESTRICT, see this domain's
//     migration) cannot be deleted, and there is no requirement yet to
//     mark one "no longer purchased" while leaving existing OLTs alone.
package oltmodel

import (
	"time"

	"github.com/google/uuid"
)

// OLTModel is a catalog entry describing one physical OLT chassis type:
// its Vendor, its model Name (e.g. "C16"), and how many PON ports it
// has.
type OLTModel struct {
	ID           uuid.UUID
	Vendor       Vendor
	Name         string
	PONPortCount int
	Description  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
