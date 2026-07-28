// Package httpapi is the Access Network domain's REST layer. It depends
// on internal/accessnetwork/service, never on a repository directly, and
// never exposes internal/accessnetwork's domain types over the wire —
// see the DTOs in this file. It mirrors internal/catalog/httpapi
// exactly.
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessnetwork"
)

// accessNetworkRequest is the JSON body for POST /api/v1/access-networks
// and PUT /api/v1/access-networks/{id}.
//
// Status is a plain string here, not accessnetwork.AccessNetworkStatus,
// even though that type would marshal to the same JSON today — the same
// "DTOs only" separation internal/catalog/httpapi.catalogRequest
// documents. The conversion happens once, explicitly, in
// toAccessNetwork below; AccessNetworkService.Create/Update reject an
// unrecognized value via AccessNetwork.Validate (see
// internal/accessnetwork/validate.go) exactly as they would for a
// request built any other way — this handler does not duplicate that
// check (the service is where validation lives).
//
// It intentionally has no ID or timestamp fields. Identity is either
// server-assigned (POST) or comes from the URL path (PUT); CreatedAt and
// UpdatedAt are metadata the repository owns and a caller cannot set.
type accessNetworkRequest struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

// toAccessNetwork converts a request into a domain
// accessnetwork.AccessNetwork. id is supplied by the caller: uuid.Nil for
// Create (the repository assigns a real one), or the URL path
// parameter's UUID for Update.
func (req accessNetworkRequest) toAccessNetwork(id uuid.UUID) accessnetwork.AccessNetwork {
	return accessnetwork.AccessNetwork{
		ID:          id,
		Name:        req.Name,
		Status:      accessnetwork.AccessNetworkStatus(req.Status),
		Description: req.Description,
	}
}

// accessNetworkResponse is the JSON representation of an AccessNetwork
// returned to clients. Decoupling the wire format from
// accessnetwork.AccessNetwork's Go field layout and types means a change
// to how the domain model is composed internally can never silently
// change the API's JSON shape.
type accessNetworkResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func newAccessNetworkResponse(a accessnetwork.AccessNetwork) accessNetworkResponse {
	return accessNetworkResponse{
		ID:          a.ID,
		Name:        a.Name,
		Status:      string(a.Status),
		Description: a.Description,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}

// accessNetworkListResponse wraps a slice of access networks in an
// object rather than returning a bare JSON array — the same reasoning as
// internal/catalog/httpapi's catalogListResponse: a bare top-level array
// can never gain sibling fields (a total count, a pagination cursor, ...)
// without becoming a breaking change for existing clients, while adding
// a field next to "access_networks" is not.
type accessNetworkListResponse struct {
	AccessNetworks []accessNetworkResponse `json:"access_networks"`
}

func newAccessNetworkListResponse(networks []accessnetwork.AccessNetwork) accessNetworkListResponse {
	resp := accessNetworkListResponse{AccessNetworks: make([]accessNetworkResponse, len(networks))}
	for i, a := range networks {
		resp.AccessNetworks[i] = newAccessNetworkResponse(a)
	}
	return resp
}
