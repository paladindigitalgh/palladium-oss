// Package httpapi is the Access Interface domain's REST layer. It
// depends on internal/accessinterface/service, never on a repository
// directly, and never exposes internal/accessinterface's domain types
// over the wire — see the DTOs in this file. It mirrors
// internal/olt/httpapi exactly.
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessinterface"
)

// accessInterfaceRequest is the JSON body for POST
// /api/v1/access-interfaces and PUT /api/v1/access-interfaces/{id}.
//
// Technology and Status are plain strings here, not
// accessinterface.Technology/accessinterface.Status, even though those
// types would marshal to the same JSON today — the same "DTOs only"
// separation internal/olt/httpapi.oltRequest documents. The conversion
// happens once, explicitly, in toAccessInterface below;
// AccessInterfaceService.Create/Update reject an unrecognized value via
// AccessInterface.Validate (see internal/accessinterface/validate.go)
// exactly as they would for a request built any other way — this
// handler does not duplicate that check (the service is where
// validation lives).
//
// PONPortID is left as its plain uuid.UUID rather than following that
// same string-everywhere rule: it carries no domain enum type to
// decouple from in the first place — the same reasoning
// internal/olt/httpapi.oltRequest gives for its own AccessNetworkID
// field.
//
// It intentionally has no ID or timestamp fields. Identity is either
// server-assigned (POST) or comes from the URL path (PUT); CreatedAt and
// UpdatedAt are metadata the repository owns and a caller cannot set.
type accessInterfaceRequest struct {
	PONPortID   uuid.UUID `json:"pon_port_id"`
	Technology  string    `json:"technology"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
}

// toAccessInterface converts a request into a domain AccessInterface. id
// is supplied by the caller: uuid.Nil for Create (the repository assigns
// a real one), or the URL path parameter's UUID for Update.
func (req accessInterfaceRequest) toAccessInterface(id uuid.UUID) accessinterface.AccessInterface {
	return accessinterface.AccessInterface{
		ID:          id,
		PONPortID:   req.PONPortID,
		Technology:  accessinterface.Technology(req.Technology),
		Name:        req.Name,
		Status:      accessinterface.Status(req.Status),
		Description: req.Description,
	}
}

// accessInterfaceResponse is the JSON representation of an
// AccessInterface returned to clients. Decoupling the wire format from
// accessinterface.AccessInterface's Go field layout and types means a
// change to how the domain model is composed internally can never
// silently change the API's JSON shape.
type accessInterfaceResponse struct {
	ID          uuid.UUID `json:"id"`
	PONPortID   uuid.UUID `json:"pon_port_id"`
	Technology  string    `json:"technology"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func newAccessInterfaceResponse(a accessinterface.AccessInterface) accessInterfaceResponse {
	return accessInterfaceResponse{
		ID:          a.ID,
		PONPortID:   a.PONPortID,
		Technology:  string(a.Technology),
		Name:        a.Name,
		Status:      string(a.Status),
		Description: a.Description,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}

// accessInterfaceListResponse wraps a slice of interfaces in an object
// rather than returning a bare JSON array — the same reasoning as
// internal/olt/httpapi's oltListResponse: a bare top-level array can
// never gain sibling fields (a total count, a pagination cursor, ...)
// without becoming a breaking change for existing clients, while adding
// a field next to "access_interfaces" is not.
type accessInterfaceListResponse struct {
	AccessInterfaces []accessInterfaceResponse `json:"access_interfaces"`
}

func newAccessInterfaceListResponse(interfaces []accessinterface.AccessInterface) accessInterfaceListResponse {
	resp := accessInterfaceListResponse{AccessInterfaces: make([]accessInterfaceResponse, len(interfaces))}
	for i, a := range interfaces {
		resp.AccessInterfaces[i] = newAccessInterfaceResponse(a)
	}
	return resp
}
