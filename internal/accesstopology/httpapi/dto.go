// Package httpapi is the Access Topology domain's REST layer. It depends
// on internal/accesstopology directly (there is no separate service
// layer — see this package's own handler doc comment for why) and never
// exposes internal/accesstopology's domain types over the wire — see the
// DTOs in this file.
package httpapi

import (
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accesstopology"
)

// customerLocationResponse is the JSON representation of one
// accesstopology.CustomerLocation: one of a Customer's currently-
// attached pieces of equipment, and where it resolves to on the access
// network. OLTID and Interface are exactly what
// POST /api/v1/diagnostics/olts/{oltId}/... (internal/diagnostics/kontron/httpapi)
// expects — a caller resolves a Customer's equipment here first, then
// runs a specific diagnostic against the OLT/interface pair it gets
// back, rather than this package exposing a second, duplicate way to run
// the same commands.
type customerLocationResponse struct {
	ServiceEquipmentID uuid.UUID `json:"service_equipment_id"`
	OLTID              uuid.UUID `json:"olt_id"`
	Interface          string    `json:"interface"`
}

// customerLocationsResponse wraps a slice of locations in an object
// rather than returning a bare JSON array — the same reasoning as every
// other list response in this codebase (see e.g.
// internal/olt/httpapi.oltListResponse): a bare top-level array can
// never gain sibling fields without becoming a breaking change for
// existing clients, while adding a field next to "locations" is not.
type customerLocationsResponse struct {
	Locations []customerLocationResponse `json:"locations"`
}

func newCustomerLocationsResponse(locations []accesstopology.CustomerLocation) customerLocationsResponse {
	resp := customerLocationsResponse{Locations: make([]customerLocationResponse, len(locations))}
	for i, l := range locations {
		resp.Locations[i] = customerLocationResponse{
			ServiceEquipmentID: l.ServiceEquipmentID,
			OLTID:              l.Location.OLTID,
			Interface:          l.Location.Interface,
		}
	}
	return resp
}
