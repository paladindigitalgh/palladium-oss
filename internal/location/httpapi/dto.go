// Package httpapi is the Location domain's REST layer. It depends on
// internal/location/service, never on a repository directly, and never
// exposes internal/location's domain types over the wire — see the DTOs
// in this file. It mirrors internal/customer/httpapi exactly.
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/location"
)

// locationRequest is the JSON body for POST /api/v1/locations and
// PUT /api/v1/locations/{id}.
//
// Type and Status are plain strings here, not location.LocationType /
// location.LocationStatus, even though those types would marshal to the
// same JSON today — the same "DTOs only" separation
// internal/customer/httpapi.customerRequest documents, applied to
// individual fields, not just whole structs. The conversion happens once,
// explicitly, in toLocation below; LocationService.Create/Update reject an
// unrecognized value via Location.Validate (see
// internal/location/validate.go) exactly as they would for a request
// built any other way — this handler does not duplicate that check (the
// service is where validation lives).
//
// CustomerID, Latitude, and Longitude are left as their plain primitive
// types (uuid.UUID, *float64) rather than following that same
// string-everywhere rule: they carry no domain enum type to decouple from
// in the first place — a uuid.UUID and a *float64 already are exactly the
// wire-level shape, with no risk of coupling to an internal Go type that
// could change independently of the JSON contract.
//
// It intentionally has no ID or timestamp fields. Identity is either
// server-assigned (POST) or comes from the URL path (PUT); CreatedAt and
// UpdatedAt are metadata the repository owns and a caller cannot set.
type locationRequest struct {
	CustomerID uuid.UUID `json:"customer_id"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	Status     string    `json:"status"`

	Address1   string `json:"address1"`
	Address2   string `json:"address2"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`

	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`

	Description string `json:"description"`
}

// toLocation converts a request into a domain location.Location. id is
// supplied by the caller: uuid.Nil for Create (the repository assigns a
// real one), or the URL path parameter's UUID for Update.
func (req locationRequest) toLocation(id uuid.UUID) location.Location {
	return location.Location{
		ID:         id,
		CustomerID: req.CustomerID,
		Name:       req.Name,
		Type:       location.LocationType(req.Type),
		Status:     location.LocationStatus(req.Status),

		Address1:   req.Address1,
		Address2:   req.Address2,
		City:       req.City,
		State:      req.State,
		PostalCode: req.PostalCode,
		Country:    req.Country,

		Latitude:  req.Latitude,
		Longitude: req.Longitude,

		Description: req.Description,
	}
}

// locationResponse is the JSON representation of a Location returned to
// clients. Decoupling the wire format from location.Location's Go field
// layout and types means a change to how the domain model is composed
// internally can never silently change the API's JSON shape.
type locationResponse struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	Status     string    `json:"status"`

	Address1   string `json:"address1"`
	Address2   string `json:"address2"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`

	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`

	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func newLocationResponse(l location.Location) locationResponse {
	return locationResponse{
		ID:         l.ID,
		CustomerID: l.CustomerID,
		Name:       l.Name,
		Type:       string(l.Type),
		Status:     string(l.Status),

		Address1:   l.Address1,
		Address2:   l.Address2,
		City:       l.City,
		State:      l.State,
		PostalCode: l.PostalCode,
		Country:    l.Country,

		Latitude:  l.Latitude,
		Longitude: l.Longitude,

		Description: l.Description,
		CreatedAt:   l.CreatedAt,
		UpdatedAt:   l.UpdatedAt,
	}
}

// locationListResponse wraps a slice of locations in an object rather
// than returning a bare JSON array — the same reasoning as
// internal/customer/httpapi's customerListResponse: a bare top-level
// array can never gain sibling fields (a total count, a pagination
// cursor, ...) without becoming a breaking change for existing clients,
// while adding a field next to "locations" is not.
type locationListResponse struct {
	Locations []locationResponse `json:"locations"`
}

func newLocationListResponse(locations []location.Location) locationListResponse {
	resp := locationListResponse{Locations: make([]locationResponse, len(locations))}
	for i, l := range locations {
		resp.Locations[i] = newLocationResponse(l)
	}
	return resp
}
