// Package httpapi is the OLT domain's REST layer. It depends on
// internal/olt/service, never on a repository directly, and never
// exposes internal/olt's domain types over the wire — see the DTOs in
// this file. It mirrors internal/product/httpapi exactly.
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/olt"
)

// oltRequest is the JSON body for POST /api/v1/olts and PUT
// /api/v1/olts/{id}.
//
// Vendor is a plain string here, not olt.Vendor, even though that type
// would marshal to the same JSON today — the same "DTOs only" separation
// internal/product/httpapi.productRequest documents. The conversion
// happens once, explicitly, in toOLT below; OLTService.Create/Update
// reject an unrecognized value via OLT.Validate (see
// internal/olt/validate.go) exactly as they would for a request built
// any other way — this handler does not duplicate that check (the
// service is where validation lives).
//
// AccessNetworkID is left as its plain uuid.UUID rather than following
// that same string-everywhere rule: it carries no domain enum type to
// decouple from in the first place — the same reasoning
// internal/product/httpapi.productRequest gives for its own CatalogID
// field.
//
// It intentionally has no ID or timestamp fields. Identity is either
// server-assigned (POST) or comes from the URL path (PUT); CreatedAt and
// UpdatedAt are metadata the repository owns and a caller cannot set.
type oltRequest struct {
	AccessNetworkID     uuid.UUID `json:"access_network_id"`
	Name                string    `json:"name"`
	Vendor              string    `json:"vendor"`
	Model               string    `json:"model"`
	ManagementIPAddress string    `json:"management_ip_address"`
	Description         string    `json:"description"`
}

// toOLT converts a request into a domain olt.OLT. id is supplied by the
// caller: uuid.Nil for Create (the repository assigns a real one), or
// the URL path parameter's UUID for Update.
func (req oltRequest) toOLT(id uuid.UUID) olt.OLT {
	return olt.OLT{
		ID:                  id,
		AccessNetworkID:     req.AccessNetworkID,
		Name:                req.Name,
		Vendor:              olt.Vendor(req.Vendor),
		Model:               req.Model,
		ManagementIPAddress: req.ManagementIPAddress,
		Description:         req.Description,
	}
}

// oltResponse is the JSON representation of an OLT returned to clients.
// Decoupling the wire format from olt.OLT's Go field layout and types
// means a change to how the domain model is composed internally can
// never silently change the API's JSON shape.
type oltResponse struct {
	ID                  uuid.UUID `json:"id"`
	AccessNetworkID     uuid.UUID `json:"access_network_id"`
	Name                string    `json:"name"`
	Vendor              string    `json:"vendor"`
	Model               string    `json:"model"`
	ManagementIPAddress string    `json:"management_ip_address"`
	Description         string    `json:"description"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func newOLTResponse(o olt.OLT) oltResponse {
	return oltResponse{
		ID:                  o.ID,
		AccessNetworkID:     o.AccessNetworkID,
		Name:                o.Name,
		Vendor:              string(o.Vendor),
		Model:               o.Model,
		ManagementIPAddress: o.ManagementIPAddress,
		Description:         o.Description,
		CreatedAt:           o.CreatedAt,
		UpdatedAt:           o.UpdatedAt,
	}
}

// oltListResponse wraps a slice of OLTs in an object rather than
// returning a bare JSON array — the same reasoning as
// internal/product/httpapi's productListResponse: a bare top-level array
// can never gain sibling fields (a total count, a pagination cursor, ...)
// without becoming a breaking change for existing clients, while adding
// a field next to "olts" is not.
type oltListResponse struct {
	OLTs []oltResponse `json:"olts"`
}

func newOLTListResponse(olts []olt.OLT) oltListResponse {
	resp := oltListResponse{OLTs: make([]oltResponse, len(olts))}
	for i, o := range olts {
		resp.OLTs[i] = newOLTResponse(o)
	}
	return resp
}
