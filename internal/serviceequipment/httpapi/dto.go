// Package httpapi is the Service Equipment domain's REST layer. It
// depends on internal/serviceequipment/service, never on a repository
// directly, and never exposes internal/serviceequipment's domain types
// over the wire — see the DTOs in this file. It mirrors
// internal/service/httpapi exactly.
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// serviceEquipmentRequest is the JSON body for POST
// /api/v1/service-equipment and PUT /api/v1/service-equipment/{id}.
//
// Role is a plain string here, not serviceequipment.EquipmentRole, even
// though that type would marshal to the same JSON today — the same
// "DTOs only" separation internal/service/httpapi.serviceRequest
// documents. The conversion happens once, explicitly, in toServiceEquipment
// below; ServiceEquipmentService.Create/Update reject an unrecognized
// value via ServiceEquipment.Validate (see
// internal/serviceequipment/validate.go) exactly as they would for a
// request built any other way — this handler does not duplicate that
// check (the service is where validation, and the active-assignment
// uniqueness rule, live).
//
// ServiceID, DeviceID, and the two lifecycle timestamps are left as their
// plain primitive types (uuid.UUID, *time.Time) rather than following
// that same string-everywhere rule: they carry no domain enum type to
// decouple from in the first place — the same reasoning
// internal/service/httpapi.serviceRequest gives for its own
// LocationID/ProductID and ActivatedAt/SuspendedAt/DisconnectedAt fields.
//
// It intentionally has no ID or CreatedAt/UpdatedAt fields. Identity is
// either server-assigned (POST) or comes from the URL path (PUT);
// CreatedAt and UpdatedAt are metadata the repository owns and a caller
// cannot set.
type serviceEquipmentRequest struct {
	ServiceID   uuid.UUID `json:"service_id"`
	DeviceID    uuid.UUID `json:"device_id"`
	Role        string    `json:"role"`
	Description string    `json:"description"`

	InstalledAt *time.Time `json:"installed_at"`
	RemovedAt   *time.Time `json:"removed_at"`
}

// toServiceEquipment converts a request into a domain ServiceEquipment.
// id is supplied by the caller: uuid.Nil for Create (the repository
// assigns a real one), or the URL path parameter's UUID for Update.
func (req serviceEquipmentRequest) toServiceEquipment(id uuid.UUID) serviceequipment.ServiceEquipment {
	return serviceequipment.ServiceEquipment{
		ID:          id,
		ServiceID:   req.ServiceID,
		DeviceID:    req.DeviceID,
		Role:        serviceequipment.EquipmentRole(req.Role),
		Description: req.Description,

		InstalledAt: req.InstalledAt,
		RemovedAt:   req.RemovedAt,
	}
}

// serviceEquipmentResponse is the JSON representation of a
// ServiceEquipment returned to clients. Decoupling the wire format from
// ServiceEquipment's Go field layout and types means a change to how the
// domain model is composed internally can never silently change the
// API's JSON shape.
type serviceEquipmentResponse struct {
	ID          uuid.UUID `json:"id"`
	ServiceID   uuid.UUID `json:"service_id"`
	DeviceID    uuid.UUID `json:"device_id"`
	Role        string    `json:"role"`
	Description string    `json:"description"`

	InstalledAt *time.Time `json:"installed_at"`
	RemovedAt   *time.Time `json:"removed_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func newServiceEquipmentResponse(e serviceequipment.ServiceEquipment) serviceEquipmentResponse {
	return serviceEquipmentResponse{
		ID:          e.ID,
		ServiceID:   e.ServiceID,
		DeviceID:    e.DeviceID,
		Role:        string(e.Role),
		Description: e.Description,

		InstalledAt: e.InstalledAt,
		RemovedAt:   e.RemovedAt,

		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

// serviceEquipmentListResponse wraps a slice of assignments in an object
// rather than returning a bare JSON array — the same reasoning as
// internal/service/httpapi's serviceListResponse: a bare top-level array
// can never gain sibling fields (a total count, a pagination cursor, ...)
// without becoming a breaking change for existing clients, while adding a
// field next to "service_equipment" is not.
type serviceEquipmentListResponse struct {
	ServiceEquipment []serviceEquipmentResponse `json:"service_equipment"`
}

func newServiceEquipmentListResponse(equipment []serviceequipment.ServiceEquipment) serviceEquipmentListResponse {
	resp := serviceEquipmentListResponse{ServiceEquipment: make([]serviceEquipmentResponse, len(equipment))}
	for i, e := range equipment {
		resp.ServiceEquipment[i] = newServiceEquipmentResponse(e)
	}
	return resp
}
