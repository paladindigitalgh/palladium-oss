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
// OLTModelID is left as a plain uuid.UUID: it carries no domain enum type
// to decouple from in the first place — the same reasoning
// internal/product/httpapi.productRequest gives for its own CatalogID
// field. OLTModelID replaced this request's former Vendor and Model
// string fields when both moved to internal/oltmodel.OLTModel (see
// internal/olt/model.go's package doc comment) — OLTModelID is what a
// caller now supplies instead.
//
// It intentionally has no ID or timestamp fields. Identity is either
// server-assigned (POST) or comes from the URL path (PUT); CreatedAt and
// UpdatedAt are metadata the repository owns and a caller cannot set.
type oltRequest struct {
	Name                string     `json:"name"`
	OLTModelID          uuid.UUID  `json:"olt_model_id"`
	ManagementIPAddress string     `json:"management_ip_address"`
	ConnectionProfileID *uuid.UUID `json:"connection_profile_id"`
	Description         string     `json:"description"`
}

// toOLT converts a request into a domain olt.OLT. id is supplied by the
// caller: uuid.Nil for Create (the repository assigns a real one), or
// the URL path parameter's UUID for Update.
func (req oltRequest) toOLT(id uuid.UUID) olt.OLT {
	return olt.OLT{
		ID:                  id,
		Name:                req.Name,
		OLTModelID:          req.OLTModelID,
		ManagementIPAddress: req.ManagementIPAddress,
		ConnectionProfileID: req.ConnectionProfileID,
		Description:         req.Description,
	}
}

// oltResponse is the JSON representation of an OLT returned to clients.
// Decoupling the wire format from olt.OLT's Go field layout and types
// means a change to how the domain model is composed internally can
// never silently change the API's JSON shape.
type oltResponse struct {
	ID                  uuid.UUID  `json:"id"`
	Name                string     `json:"name"`
	OLTModelID          uuid.UUID  `json:"olt_model_id"`
	ManagementIPAddress string     `json:"management_ip_address"`
	ConnectionProfileID *uuid.UUID `json:"connection_profile_id"`
	Description         string     `json:"description"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func newOLTResponse(o olt.OLT) oltResponse {
	return oltResponse{
		ID:                  o.ID,
		Name:                o.Name,
		OLTModelID:          o.OLTModelID,
		ManagementIPAddress: o.ManagementIPAddress,
		ConnectionProfileID: o.ConnectionProfileID,
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
