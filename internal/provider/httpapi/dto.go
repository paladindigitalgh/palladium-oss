// Package httpapi is the Provider domain's REST layer. It depends on
// internal/provider/service, never on a repository directly, and never
// exposes internal/provider's domain types over the wire — see the DTOs
// in this file. It mirrors internal/serviceprofile/httpapi exactly.
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/provider"
)

// providerRequest is the JSON body for POST /api/v1/providers and
// PUT /api/v1/providers/{id}.
//
// Status is a plain string here, not provider.Status, even though that
// type would marshal to the same JSON today — the same "DTOs only"
// separation internal/serviceprofile/httpapi.serviceProfileRequest
// documents. The conversion happens once, explicitly, in toProvider
// below; ProviderService.Create/Update reject an unrecognized value via
// Provider.Validate (see internal/provider/validate.go) exactly as they
// would for a request built any other way — this handler does not
// duplicate that check (the service is where validation lives).
//
// It intentionally has no ID or timestamp fields. Identity is either
// server-assigned (POST) or comes from the URL path (PUT); CreatedAt and
// UpdatedAt are metadata the repository owns and a caller cannot set.
type providerRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// toProvider converts a request into a domain provider.Provider. id is
// supplied by the caller: uuid.Nil for Create (the repository assigns a
// real one), or the URL path parameter's UUID for Update.
func (req providerRequest) toProvider(id uuid.UUID) provider.Provider {
	return provider.Provider{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Status:      provider.Status(req.Status),
	}
}

// providerResponse is the JSON representation of a Provider returned to
// clients. Decoupling the wire format from provider.Provider's Go field
// layout and types means a change to how the domain model is composed
// internally can never silently change the API's JSON shape.
type providerResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func newProviderResponse(p provider.Provider) providerResponse {
	return providerResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Status:      string(p.Status),
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// providerListResponse wraps a slice of providers in an object rather
// than returning a bare JSON array — the same reasoning as
// internal/serviceprofile/httpapi's serviceProfileListResponse.
type providerListResponse struct {
	Providers []providerResponse `json:"providers"`
}

func newProviderListResponse(providers []provider.Provider) providerListResponse {
	resp := providerListResponse{Providers: make([]providerResponse, len(providers))}
	for i, p := range providers {
		resp.Providers[i] = newProviderResponse(p)
	}
	return resp
}
