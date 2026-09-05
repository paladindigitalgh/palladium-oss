// Package httpapi is the ProvisioningProfile domain's REST layer. It
// depends on internal/provisioning/service, never on a repository
// directly, and never exposes internal/provisioning's domain types over
// the wire — see the DTOs in this file. It mirrors internal/product/httpapi
// exactly.
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/provisioning"
)

// provisioningProfileRequest is the JSON body for
// POST /api/v1/provisioning-profiles and
// PUT /api/v1/provisioning-profiles/{id}.
//
// It intentionally has no ID or timestamp fields. Identity is either
// server-assigned (POST) or comes from the URL path (PUT); CreatedAt and
// UpdatedAt are metadata the repository owns and a caller cannot set.
type provisioningProfileRequest struct {
	ProductID   uuid.UUID `json:"product_id"`
	Vendor      string    `json:"vendor"`
	ProfileName string    `json:"profile_name"`
	Description string    `json:"description"`
}

// toProvisioningProfile converts a request into a domain
// provisioning.ProvisioningProfile. id is supplied by the caller:
// uuid.Nil for Create (the repository assigns a real one), or the URL
// path parameter's UUID for Update.
func (req provisioningProfileRequest) toProvisioningProfile(id uuid.UUID) provisioning.ProvisioningProfile {
	return provisioning.ProvisioningProfile{
		ID:          id,
		ProductID:   req.ProductID,
		Vendor:      req.Vendor,
		ProfileName: req.ProfileName,
		Description: req.Description,
	}
}

// provisioningProfileResponse is the JSON representation of a
// ProvisioningProfile returned to clients. Decoupling the wire format
// from provisioning.ProvisioningProfile's Go field layout means a change
// to how the domain model is composed internally can never silently
// change the API's JSON shape.
type provisioningProfileResponse struct {
	ID          uuid.UUID `json:"id"`
	ProductID   uuid.UUID `json:"product_id"`
	Vendor      string    `json:"vendor"`
	ProfileName string    `json:"profile_name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func newProvisioningProfileResponse(p provisioning.ProvisioningProfile) provisioningProfileResponse {
	return provisioningProfileResponse{
		ID:          p.ID,
		ProductID:   p.ProductID,
		Vendor:      p.Vendor,
		ProfileName: p.ProfileName,
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// provisioningProfileListResponse wraps a slice of profiles in an object
// rather than returning a bare JSON array — the same reasoning as
// internal/product/httpapi's productListResponse.
type provisioningProfileListResponse struct {
	ProvisioningProfiles []provisioningProfileResponse `json:"provisioning_profiles"`
}

func newProvisioningProfileListResponse(profiles []provisioning.ProvisioningProfile) provisioningProfileListResponse {
	resp := provisioningProfileListResponse{ProvisioningProfiles: make([]provisioningProfileResponse, len(profiles))}
	for i, p := range profiles {
		resp.ProvisioningProfiles[i] = newProvisioningProfileResponse(p)
	}
	return resp
}
