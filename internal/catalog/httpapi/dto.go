// Package httpapi is the Catalog domain's REST layer. It depends on
// internal/catalog/service, never on a repository directly, and never
// exposes internal/catalog's domain types over the wire — see the DTOs
// in this file. It mirrors internal/location/httpapi exactly.
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/catalog"
)

// catalogRequest is the JSON body for POST /api/v1/catalogs and
// PUT /api/v1/catalogs/{id}.
//
// Status is a plain string here, not catalog.CatalogStatus, even though
// that type would marshal to the same JSON today — the same "DTOs only"
// separation internal/location/httpapi.locationRequest documents. The
// conversion happens once, explicitly, in toCatalog below;
// CatalogService.Create/Update reject an unrecognized value via
// ProductCatalog.Validate (see internal/catalog/validate.go) exactly as
// they would for a request built any other way — this handler does not
// duplicate that check (the service is where validation lives).
//
// It intentionally has no ID or timestamp fields. Identity is either
// server-assigned (POST) or comes from the URL path (PUT); CreatedAt and
// UpdatedAt are metadata the repository owns and a caller cannot set.
type catalogRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// toCatalog converts a request into a domain catalog.ProductCatalog. id
// is supplied by the caller: uuid.Nil for Create (the repository assigns
// a real one), or the URL path parameter's UUID for Update.
func (req catalogRequest) toCatalog(id uuid.UUID) catalog.ProductCatalog {
	return catalog.ProductCatalog{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Status:      catalog.CatalogStatus(req.Status),
	}
}

// catalogResponse is the JSON representation of a ProductCatalog returned
// to clients. Decoupling the wire format from catalog.ProductCatalog's Go
// field layout and types means a change to how the domain model is
// composed internally can never silently change the API's JSON shape.
type catalogResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func newCatalogResponse(c catalog.ProductCatalog) catalogResponse {
	return catalogResponse{
		ID:          c.ID,
		Name:        c.Name,
		Description: c.Description,
		Status:      string(c.Status),
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

// catalogListResponse wraps a slice of catalogs in an object rather than
// returning a bare JSON array — the same reasoning as
// internal/location/httpapi's locationListResponse: a bare top-level
// array can never gain sibling fields (a total count, a pagination
// cursor, ...) without becoming a breaking change for existing clients,
// while adding a field next to "catalogs" is not.
type catalogListResponse struct {
	Catalogs []catalogResponse `json:"catalogs"`
}

func newCatalogListResponse(catalogs []catalog.ProductCatalog) catalogListResponse {
	resp := catalogListResponse{Catalogs: make([]catalogResponse, len(catalogs))}
	for i, c := range catalogs {
		resp.Catalogs[i] = newCatalogResponse(c)
	}
	return resp
}
