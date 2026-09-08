// Package provisioning models the mapping between a commercial Product
// (see internal/product — e.g. "Residential Internet 500/500") and the
// named configuration profile a specific OLT vendor uses to actually
// deliver it — e.g. a Kontron/Iskratel C16 "service profile" that
// bundles rate limiting and VLAN assignment under one name.
//
// This is deliberately NOT the internal/provisioning that existed
// earlier in this codebase's history (see
// database/migrations/00028_provisioning_drop_provisioning_jobs.sql): that
// package modeled ProvisioningJob, an execution/job-tracking concept
// superseded by internal/workflow + internal/plugin. Nothing here tracks
// a job, runs a command, or opens a connection to a device — it is pure
// reference data, the same kind of thing internal/connectionprofile
// already is. Reusing this package path is deliberate too: see
// internal/serviceprofile's own package doc comment, which already names
// internal/provisioning/kontron as where vendor-specific command
// execution (not vendor-specific reference data, which is what this file
// holds) belongs — see that package's own doc comment for why it is a
// sibling of this one, not a competing implementation of the same
// responsibility.
//
// Per this milestone's explicit scope:
//
//   - No command execution: applying a ProvisioningProfile's rate/VLAN
//     configuration to a live ONU is still future work, layered on top
//     of this lookup, not folded into it — internal/provisioning/kontron
//     exists now, but only for ONU authorization (a fixed, always-applied
//     management profile), not yet for a Product-specific
//     ProvisioningProfile. See that package for the precedent this future
//     work will follow: raw vendor commands, no parsing, isolated from
//     core.
//   - Vendor is a plain string here, not oltmodel.Vendor's closed enum
//     (see internal/oltmodel/vendor.go) -- a second vendor tomorrow is a
//     new row, not a schema change, and this package has no reason to
//     reject a vendor OLTModel has not yet been taught about.
//   - One profile per (Product, Vendor): a given commercial offering
//     maps to exactly one named profile on a given vendor's equipment.
package provisioning

import (
	"time"

	"github.com/google/uuid"
)

// ProvisioningProfile maps one Product to the named profile a specific
// OLT vendor already has configured for it -- built by an operator
// directly on the OLT (see the package doc comment), never generated or
// modified by Palladium itself.
type ProvisioningProfile struct {
	ID          uuid.UUID
	ProductID   uuid.UUID
	Vendor      string
	ProfileName string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
