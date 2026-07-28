// Package catalog models Palladium's Product Catalog domain (v1): the
// grouping under which the ISP's Products are organized — e.g. a
// "Residential" catalog versus a "Business" catalog. This package holds
// only the domain model, field validation, and the repository interface —
// no SQL, no migrations, no HTTP CRUD — mirroring internal/customer's own
// package exactly.
//
// A ProductCatalog describes what the ISP offers, never who it is
// delivered to and never what a subscriber has. Per this milestone's
// explicit scope:
//
//   - No pricing, no billing: a catalog is a name and a lifecycle state,
//     not a price book. Rate cards are a real future feature, not implied
//     by anything here.
//   - No provisioning, no bandwidth profiles, no automation: nothing here
//     configures a network or a device. See internal/product's package
//     doc comment for the same boundary drawn one level down, at the
//     individual Product.
//   - No service assignments: a ProductCatalog never references
//     internal/customer.Customer or any other customer-facing type. Per
//     CLAUDE.md's Core Philosophy ("Customers own Services. Services
//     consume Resources.") a future Service will reference a Product, not
//     the other way around, and nothing in the Catalog domain reaches
//     toward Customers at all.
//
// Everything above is a real feature some future milestone will add. None
// of it is implied by what exists today.
package catalog

import (
	"time"

	"github.com/google/uuid"
)

// ProductCatalog groups the ISP's Products (see internal/product) under a
// single named, independently-lifecycled collection.
type ProductCatalog struct {
	ID          uuid.UUID
	Name        string
	Description string
	Status      CatalogStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
