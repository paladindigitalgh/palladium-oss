// Package customer models Palladium's Customer domain (v1): who a service
// could be delivered to, and nothing about what is actually being
// delivered. This package holds only the domain model, field validation,
// and the repository interface — no SQL, no migrations, no HTTP CRUD —
// mirroring internal/inventory's own package exactly.
//
// This is deliberately not a CRM. Per CLAUDE.md's Core Philosophy
// ("Customers own Services. Services consume Resources. Resources exist
// independently of Customers.") and this milestone's explicit scope, a
// Customer here is an identity record only:
//
//   - No contacts, no addresses: a Customer is not yet "who do we call"
//     or "where do we ship equipment" — those are their own future
//     domains, not fields bolted onto this one.
//   - No services, no billing, no provisioning: this package cannot
//     answer "what does this Customer have" at all. See
//     docs/ARCHITECTURE.md's Customer Philosophy — Services (a later
//     phase) will reference a Customer, not the other way around.
//   - No equipment relationships: a Customer never references
//     inventory.Device or any other Inventory type directly, for the same
//     reason auth.User does not embed inventory.Metadata — these are
//     separate domains, and CLAUDE.md's Core Philosophy is explicit that
//     inventory must never be coupled directly to customers.
//   - No notes, no documents: free-form record-keeping is a feature, not
//     a byproduct of naming a field "notes"; it isn't asked for here.
//
// Everything above is a real feature some future milestone will add. None
// of it is implied by what exists today.
package customer

import (
	"time"

	"github.com/google/uuid"
)

// Customer is a billing/service identity — who a Service could be
// delivered to. It is not a physical or logical resource (see
// internal/inventory) and it is not yet reachable at any address or
// through any contact (see the package doc comment).
type Customer struct {
	ID           uuid.UUID
	Name         string
	CustomerType CustomerType
	Status       CustomerStatus
	Description  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
