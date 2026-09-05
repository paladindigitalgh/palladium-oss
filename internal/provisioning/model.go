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
// "internal/provisioning/connectors" as where vendor-specific
// connector logic (not vendor-specific reference data, which is what
// this file holds) belongs, once that work starts.
//
// Per this milestone's explicit scope:
//
//   - No command execution: applying a ProvisioningProfile to a live ONU
//     (the two or so CLI lines an operator described running by hand
//     today) is future work, layered on top of this lookup, not folded
//     into it. See internal/diagnostics/kontron for the precedent this
//     future work will follow: raw vendor commands, no parsing, isolated
//     from core.
//   - Vendor is a plain string, not an enum, mirroring olt.OLT's own
//     Vendor field exactly -- a second vendor tomorrow is a new row, not
//     a schema change.
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
