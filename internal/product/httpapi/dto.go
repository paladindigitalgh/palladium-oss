// Package httpapi is the Product domain's REST layer. It depends on
// internal/product/service, never on a repository directly, and never
// exposes internal/product's domain types over the wire — see the DTOs
// in this file. It mirrors internal/catalog/httpapi exactly.
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/product"
)

// productRequest is the JSON body for POST /api/v1/products and
// PUT /api/v1/products/{id}.
//
// Category and Status are plain strings here, not product.ProductCategory
// / product.ProductStatus, even though those types would marshal to the
// same JSON today — the same "DTOs only" separation
// internal/catalog/httpapi.catalogRequest documents, applied to
// individual fields, not just whole structs. The conversion happens once,
// explicitly, in toProduct below; ProductService.Create/Update reject an
// unrecognized value via Product.Validate (see internal/product/validate.go)
// exactly as they would for a request built any other way — this handler
// does not duplicate that check (the service is where validation lives).
//
// CatalogID and ProviderID are left as plain uuid.UUID rather than
// following that same string-everywhere rule: neither carries a domain
// enum type to decouple from in the first place — the same reasoning
// internal/location/httpapi.locationRequest gives for its own CustomerID
// field.
//
// It intentionally has no ID or timestamp fields. Identity is either
// server-assigned (POST) or comes from the URL path (PUT); CreatedAt and
// UpdatedAt are metadata the repository owns and a caller cannot set.
type productRequest struct {
	CatalogID   uuid.UUID `json:"catalog_id"`
	ProviderID  uuid.UUID `json:"provider_id"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
}

// toProduct converts a request into a domain product.Product. id is
// supplied by the caller: uuid.Nil for Create (the repository assigns a
// real one), or the URL path parameter's UUID for Update.
func (req productRequest) toProduct(id uuid.UUID) product.Product {
	return product.Product{
		ID:          id,
		CatalogID:   req.CatalogID,
		ProviderID:  req.ProviderID,
		Name:        req.Name,
		Category:    product.ProductCategory(req.Category),
		Status:      product.ProductStatus(req.Status),
		Description: req.Description,
	}
}

// productResponse is the JSON representation of a Product returned to
// clients. Decoupling the wire format from product.Product's Go field
// layout and types means a change to how the domain model is composed
// internally can never silently change the API's JSON shape.
type productResponse struct {
	ID          uuid.UUID `json:"id"`
	CatalogID   uuid.UUID `json:"catalog_id"`
	ProviderID  uuid.UUID `json:"provider_id"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func newProductResponse(p product.Product) productResponse {
	return productResponse{
		ID:          p.ID,
		CatalogID:   p.CatalogID,
		ProviderID:  p.ProviderID,
		Name:        p.Name,
		Category:    string(p.Category),
		Status:      string(p.Status),
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// productListResponse wraps a slice of products in an object rather than
// returning a bare JSON array — the same reasoning as
// internal/catalog/httpapi's catalogListResponse: a bare top-level array
// can never gain sibling fields (a total count, a pagination cursor, ...)
// without becoming a breaking change for existing clients, while adding a
// field next to "products" is not.
type productListResponse struct {
	Products []productResponse `json:"products"`
}

func newProductListResponse(products []product.Product) productListResponse {
	resp := productListResponse{Products: make([]productResponse, len(products))}
	for i, p := range products {
		resp.Products[i] = newProductResponse(p)
	}
	return resp
}
