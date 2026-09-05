// Package provider models Palladium's Provider domain: the retail ISP
// that owns and sells a set of Plans (see internal/product and
// internal/provisioning) — the company identity a Product belongs to,
// distinct from the network operator that owns the physical OLTs and
// PON ports those Plans are eventually delivered over.
//
// This distinction only matters in an open-access deployment, where one
// physical network carries multiple independent ISPs, each with their
// own Plans and their own OLT vendor profiles for otherwise-identical
// speed tiers (see internal/provisioning's package doc comment — a
// ProvisioningProfile is keyed by Product, and every Product now belongs
// to exactly one Provider, so per-Provider profile isolation falls out
// automatically with no change to that domain). In a single-ISP
// deployment, exactly one Provider exists and the frontend never asks an
// operator to think about it (see productRepository.ts's
// createProduct doc comment).
//
// This package holds only the domain model, field validation, and the
// repository interface — no SQL, no migrations, no HTTP CRUD —
// mirroring internal/serviceprofile's own package exactly. Per this
// milestone's explicit scope, this package does not model:
//
//   - Billing, contracts, or revenue share agreements between the
//     network operator and a Provider: those are a future commercial
//     domain, not implied by anything here.
//   - Which OLTs, PON ports, or VLANs a Provider is entitled to use:
//     that is a future Network/open-access domain. A Provider here is
//     purely a commercial identity Products belong to.
//   - Provider-facing accounts, logins, or a Provider self-service
//     portal: Palladium's users are always the network operator's own
//     staff (see CLAUDE.md — there is no customer- or partner-facing UI
//     anywhere in this system).
package provider

import (
	"time"

	"github.com/google/uuid"
)

// Provider is a retail ISP identity that Products (see internal/product)
// belong to.
type Provider struct {
	ID          uuid.UUID
	Name        string
	Status      Status
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
