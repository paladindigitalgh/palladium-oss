// Package product models Palladium's Product domain (v1): an individual
// offering within a ProductCatalog (see internal/catalog) — e.g.
// "Residential Internet 100/20". This package holds only the domain
// model, field validation, and the repository interface — no SQL, no
// migrations, no HTTP CRUD — mirroring internal/location's own package
// exactly.
//
// This package does not import internal/catalog. CatalogID is a bare
// uuid.UUID, not a catalog.ProductCatalog reference: the foreign key to
// catalogs(id) is a database concept, enforced by
// internal/product/postgres and its migration, not a Go package
// dependency — the same reasoning internal/location/model.go documents
// for why Location does not import internal/customer.
//
// A Product describes what the ISP offers, never a subscriber's actual
// service. Per this milestone's explicit scope:
//
//   - No pricing, no billing: a Product has no rate, no billing cycle, no
//     currency. Those belong to a future rate card, layered on top of
//     this catalog entry, not folded into it.
//   - No provisioning, no bandwidth profiles, no automation: nothing
//     here configures a device, a VLAN, or a queue. A Product is a
//     catalog entry a Service (a later phase; see
//     docs/ARCHITECTURE.md's Services domain) will eventually reference —
//     it does not itself cause anything to happen on the network.
//   - No service assignments: a Product never references
//     internal/customer.Customer. Per CLAUDE.md's Core Philosophy
//     ("Customers own Services. Services consume Resources.") a future
//     Service will reference both a Customer and a Product; neither of
//     those two references belongs on the Product itself.
//
// Everything above is a real feature some future milestone will add. None
// of it is implied by what exists today.
package product

import (
	"time"

	"github.com/google/uuid"
)

// Product is a single offering within a ProductCatalog — what the ISP
// sells, described independently of who buys it (see the package doc
// comment for what this deliberately excludes).
type Product struct {
	ID          uuid.UUID
	CatalogID   uuid.UUID
	Name        string
	Category    ProductCategory
	Status      ProductStatus
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
