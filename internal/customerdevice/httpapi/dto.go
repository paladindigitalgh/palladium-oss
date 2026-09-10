// Package httpapi is the Customer Device domain's REST layer. It
// depends on internal/customerdevice/service, never on a repository
// directly, and never exposes internal/customerdevice's domain types
// over the wire — see the DTOs in this file. It mirrors
// internal/serviceequipment/httpapi exactly.
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/customerdevice"
)

// customerDeviceRequest is the JSON body for POST /api/v1/customer-devices
// and PUT /api/v1/customer-devices/{id}.
//
// It intentionally has no ID or CreatedAt/UpdatedAt fields. Identity is
// either server-assigned (POST) or comes from the URL path (PUT);
// CreatedAt and UpdatedAt are metadata the repository owns and a caller
// cannot set. Mirrors
// internal/serviceequipment/httpapi.serviceEquipmentRequest exactly.
type customerDeviceRequest struct {
	CustomerID uuid.UUID  `json:"customer_id"`
	DeviceID   uuid.UUID  `json:"device_id"`
	LocationID *uuid.UUID `json:"location_id"`

	AttachedAt *time.Time `json:"attached_at"`
	DetachedAt *time.Time `json:"detached_at"`
}

// toCustomerDevice converts a request into a domain CustomerDevice. id is
// supplied by the caller: uuid.Nil for Create (the repository assigns a
// real one), or the URL path parameter's UUID for Update.
func (req customerDeviceRequest) toCustomerDevice(id uuid.UUID) customerdevice.CustomerDevice {
	return customerdevice.CustomerDevice{
		ID:         id,
		CustomerID: req.CustomerID,
		DeviceID:   req.DeviceID,
		LocationID: req.LocationID,

		AttachedAt: req.AttachedAt,
		DetachedAt: req.DetachedAt,
	}
}

// customerDeviceResponse is the JSON representation of a CustomerDevice
// returned to clients. Decoupling the wire format from CustomerDevice's
// Go field layout means a change to how the domain model is composed
// internally can never silently change the API's JSON shape.
type customerDeviceResponse struct {
	ID         uuid.UUID  `json:"id"`
	CustomerID uuid.UUID  `json:"customer_id"`
	DeviceID   uuid.UUID  `json:"device_id"`
	LocationID *uuid.UUID `json:"location_id"`

	AttachedAt *time.Time `json:"attached_at"`
	DetachedAt *time.Time `json:"detached_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func newCustomerDeviceResponse(cd customerdevice.CustomerDevice) customerDeviceResponse {
	return customerDeviceResponse{
		ID:         cd.ID,
		CustomerID: cd.CustomerID,
		DeviceID:   cd.DeviceID,
		LocationID: cd.LocationID,

		AttachedAt: cd.AttachedAt,
		DetachedAt: cd.DetachedAt,

		CreatedAt: cd.CreatedAt,
		UpdatedAt: cd.UpdatedAt,
	}
}

// customerDeviceListResponse wraps a slice of records in an object
// rather than returning a bare JSON array, the same reasoning as
// internal/serviceequipment/httpapi's serviceEquipmentListResponse.
type customerDeviceListResponse struct {
	CustomerDevices []customerDeviceResponse `json:"customer_devices"`
}

func newCustomerDeviceListResponse(records []customerdevice.CustomerDevice) customerDeviceListResponse {
	resp := customerDeviceListResponse{CustomerDevices: make([]customerDeviceResponse, len(records))}
	for i, cd := range records {
		resp.CustomerDevices[i] = newCustomerDeviceResponse(cd)
	}
	return resp
}
