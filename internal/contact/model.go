// Package contact models Palladium's Contact domain (v1): a person to
// reach about a Customer's account — a billing contact, a technical
// contact, an emergency contact. This package holds only the domain
// model, field validation, and the repository interface — no SQL, no
// migrations, no HTTP CRUD — mirroring internal/location exactly, the
// closest structural precedent: a Contact belongs to one Customer, has no
// Detail page of its own, and is managed as a nested list inside the
// Customer Detail Workspace, exactly as Location already is.
//
// This is the domain internal/location's own package doc comment
// deferred: "who do we call at this address" was explicitly out of that
// package's v1 scope, named as a future domain rather than a field
// bolted onto Location. This package is that future domain, arriving
// once the rest of the Customer sub-resource pattern (Location, Service)
// was already proven out this session.
//
// A Contact references a Customer by ID only (CustomerID uuid.UUID) —
// this package does not import internal/customer at all, the same
// deliberate decoupling internal/location's own model.go documents: a
// foreign key is a database constraint, not a Go dependency.
package contact

import (
	"time"

	"github.com/google/uuid"
)

// Contact is a person to reach about a Customer's account.
type Contact struct {
	ID         uuid.UUID
	CustomerID uuid.UUID
	Name       string
	Role       ContactRole
	Email      string
	Phone      string
	Status     ContactStatus

	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
