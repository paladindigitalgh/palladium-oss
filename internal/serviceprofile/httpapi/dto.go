// Package httpapi is the Service Profile domain's REST layer. It
// depends on internal/serviceprofile/service, never on a repository
// directly, and never exposes internal/serviceprofile's domain types
// over the wire — see the DTOs in this file. It mirrors
// internal/catalog/httpapi exactly.
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/serviceprofile"
)

// serviceProfileRequest is the JSON body for POST
// /api/v1/service-profiles and PUT /api/v1/service-profiles/{id}.
//
// Status is a plain string here, not serviceprofile.Status, even though
// that type would marshal to the same JSON today — the same "DTOs only"
// separation internal/catalog/httpapi.catalogRequest documents. The
// conversion happens once, explicitly, in toServiceProfile below;
// ServiceProfileService.Create/Update reject an unrecognized value via
// ServiceProfile.Validate (see internal/serviceprofile/validate.go)
// exactly as they would for a request built any other way — this
// handler does not duplicate that check (the service is where
// validation lives).
//
// It intentionally has no ID or timestamp fields. Identity is either
// server-assigned (POST) or comes from the URL path (PUT); CreatedAt and
// UpdatedAt are metadata the repository owns and a caller cannot set.
type serviceProfileRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// toServiceProfile converts a request into a domain
// serviceprofile.ServiceProfile. id is supplied by the caller: uuid.Nil
// for Create (the repository assigns a real one), or the URL path
// parameter's UUID for Update.
func (req serviceProfileRequest) toServiceProfile(id uuid.UUID) serviceprofile.ServiceProfile {
	return serviceprofile.ServiceProfile{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Status:      serviceprofile.Status(req.Status),
	}
}

// serviceProfileResponse is the JSON representation of a ServiceProfile
// returned to clients. Decoupling the wire format from
// serviceprofile.ServiceProfile's Go field layout and types means a
// change to how the domain model is composed internally can never
// silently change the API's JSON shape.
type serviceProfileResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func newServiceProfileResponse(p serviceprofile.ServiceProfile) serviceProfileResponse {
	return serviceProfileResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Status:      string(p.Status),
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// serviceProfileListResponse wraps a slice of service profiles in an
// object rather than returning a bare JSON array — the same reasoning
// as internal/catalog/httpapi's catalogListResponse: a bare top-level
// array can never gain sibling fields (a total count, a pagination
// cursor, ...) without becoming a breaking change for existing clients,
// while adding a field next to "service_profiles" is not.
type serviceProfileListResponse struct {
	ServiceProfiles []serviceProfileResponse `json:"service_profiles"`
}

func newServiceProfileListResponse(profiles []serviceprofile.ServiceProfile) serviceProfileListResponse {
	resp := serviceProfileListResponse{ServiceProfiles: make([]serviceProfileResponse, len(profiles))}
	for i, p := range profiles {
		resp.ServiceProfiles[i] = newServiceProfileResponse(p)
	}
	return resp
}
