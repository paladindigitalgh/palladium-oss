// Package httpapi is the Service domain's REST layer. It depends on
// internal/service/service, never on a repository directly, and never
// exposes internal/service's domain types over the wire — see the DTOs
// in this file. It mirrors internal/product/httpapi exactly.
package httpapi

import (
	"time"

	"github.com/google/uuid"

	domainservice "github.com/paladindigitalgh/palladium-oss/internal/service"
)

// serviceRequest is the JSON body for POST /api/v1/services and
// PUT /api/v1/services/{id}.
//
// Status is a plain string here, not domainservice.ServiceStatus, even
// though that type would marshal to the same JSON today — the same
// "DTOs only" separation internal/product/httpapi.productRequest
// documents. The conversion happens once, explicitly, in toService below;
// ServiceService.Create/Update reject an unrecognized value via
// Service.Validate (see internal/service/validate.go) exactly as they
// would for a request built any other way — this handler does not
// duplicate that check (the service is where validation lives).
//
// LocationID, ProductID, and the three lifecycle timestamps are left as
// their plain primitive types (uuid.UUID, *time.Time) rather than
// following that same string-everywhere rule: they carry no domain enum
// type to decouple from in the first place — the same reasoning
// internal/location/httpapi.locationRequest gives for its own CustomerID
// field, and internal/location/httpapi gives for Latitude/Longitude.
//
// It intentionally has no ID or CreatedAt/UpdatedAt fields. Identity is
// either server-assigned (POST) or comes from the URL path (PUT);
// CreatedAt and UpdatedAt are metadata the repository owns and a caller
// cannot set.
type serviceRequest struct {
	LocationID  uuid.UUID `json:"location_id"`
	ProductID   uuid.UUID `json:"product_id"`
	Status      string    `json:"status"`
	Description string    `json:"description"`

	ActivatedAt    *time.Time `json:"activated_at"`
	SuspendedAt    *time.Time `json:"suspended_at"`
	DisconnectedAt *time.Time `json:"disconnected_at"`
}

// toService converts a request into a domain Service. id is supplied by
// the caller: uuid.Nil for Create (the repository assigns a real one), or
// the URL path parameter's UUID for Update.
func (req serviceRequest) toService(id uuid.UUID) domainservice.Service {
	return domainservice.Service{
		ID:          id,
		LocationID:  req.LocationID,
		ProductID:   req.ProductID,
		Status:      domainservice.ServiceStatus(req.Status),
		Description: req.Description,

		ActivatedAt:    req.ActivatedAt,
		SuspendedAt:    req.SuspendedAt,
		DisconnectedAt: req.DisconnectedAt,
	}
}

// serviceResponse is the JSON representation of a Service returned to
// clients. Decoupling the wire format from Service's Go field layout and
// types means a change to how the domain model is composed internally
// can never silently change the API's JSON shape.
type serviceResponse struct {
	ID          uuid.UUID `json:"id"`
	LocationID  uuid.UUID `json:"location_id"`
	ProductID   uuid.UUID `json:"product_id"`
	Status      string    `json:"status"`
	Description string    `json:"description"`

	ActivatedAt    *time.Time `json:"activated_at"`
	SuspendedAt    *time.Time `json:"suspended_at"`
	DisconnectedAt *time.Time `json:"disconnected_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func newServiceResponse(s domainservice.Service) serviceResponse {
	return serviceResponse{
		ID:          s.ID,
		LocationID:  s.LocationID,
		ProductID:   s.ProductID,
		Status:      string(s.Status),
		Description: s.Description,

		ActivatedAt:    s.ActivatedAt,
		SuspendedAt:    s.SuspendedAt,
		DisconnectedAt: s.DisconnectedAt,

		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

// serviceListResponse wraps a slice of services in an object rather than
// returning a bare JSON array — the same reasoning as
// internal/product/httpapi's productListResponse: a bare top-level array
// can never gain sibling fields (a total count, a pagination cursor, ...)
// without becoming a breaking change for existing clients, while adding a
// field next to "services" is not.
type serviceListResponse struct {
	Services []serviceResponse `json:"services"`
}

func newServiceListResponse(services []domainservice.Service) serviceListResponse {
	resp := serviceListResponse{Services: make([]serviceResponse, len(services))}
	for i, s := range services {
		resp.Services[i] = newServiceResponse(s)
	}
	return resp
}
