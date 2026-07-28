// Package httpapi is the PON Port domain's REST layer. It depends on
// internal/ponport/service, never on a repository directly, and never
// exposes internal/ponport's domain types over the wire — see the DTOs
// in this file. It mirrors internal/olt/httpapi exactly.
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/ponport"
)

// ponPortRequest is the JSON body for POST /api/v1/pon-ports and PUT
// /api/v1/pon-ports/{id}.
//
// OLTID and PortNumber are left as their plain primitive types
// (uuid.UUID, int) rather than following the string-everywhere rule this
// codebase's other DTOs use for domain enum fields: PONPort has no enum
// field at all (see internal/ponport/model.go's doc comment on what this
// package deliberately does not model — no Status), so there is nothing
// here to decouple from in the first place, the same reasoning
// internal/olt/httpapi.oltRequest gives for its own AccessNetworkID
// field.
//
// It intentionally has no ID or timestamp fields. Identity is either
// server-assigned (POST) or comes from the URL path (PUT); CreatedAt and
// UpdatedAt are metadata the repository owns and a caller cannot set.
type ponPortRequest struct {
	OLTID       uuid.UUID `json:"olt_id"`
	PortNumber  int       `json:"port_number"`
	Description string    `json:"description"`
}

// toPONPort converts a request into a domain ponport.PONPort. id is
// supplied by the caller: uuid.Nil for Create (the repository assigns a
// real one), or the URL path parameter's UUID for Update.
func (req ponPortRequest) toPONPort(id uuid.UUID) ponport.PONPort {
	return ponport.PONPort{
		ID:          id,
		OLTID:       req.OLTID,
		PortNumber:  req.PortNumber,
		Description: req.Description,
	}
}

// ponPortResponse is the JSON representation of a PONPort returned to
// clients. Decoupling the wire format from ponport.PONPort's Go field
// layout and types means a change to how the domain model is composed
// internally can never silently change the API's JSON shape.
type ponPortResponse struct {
	ID          uuid.UUID `json:"id"`
	OLTID       uuid.UUID `json:"olt_id"`
	PortNumber  int       `json:"port_number"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func newPONPortResponse(p ponport.PONPort) ponPortResponse {
	return ponPortResponse{
		ID:          p.ID,
		OLTID:       p.OLTID,
		PortNumber:  p.PortNumber,
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// ponPortListResponse wraps a slice of ports in an object rather than
// returning a bare JSON array — the same reasoning as
// internal/olt/httpapi's oltListResponse: a bare top-level array can
// never gain sibling fields (a total count, a pagination cursor, ...)
// without becoming a breaking change for existing clients, while adding
// a field next to "pon_ports" is not.
type ponPortListResponse struct {
	PONPorts []ponPortResponse `json:"pon_ports"`
}

func newPONPortListResponse(ports []ponport.PONPort) ponPortListResponse {
	resp := ponPortListResponse{PONPorts: make([]ponPortResponse, len(ports))}
	for i, p := range ports {
		resp.PONPorts[i] = newPONPortResponse(p)
	}
	return resp
}
