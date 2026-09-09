// Package httpapi is the Kontron ONU-authorization REST layer. It
// depends on internal/provisioning/kontron/service, never on
// internal/provisioning/kontron or internal/olt/connect directly, and
// never exposes their types over the wire — see the DTOs in this file.
// It mirrors internal/diagnostics/kontron/httpapi's shape exactly, even
// though this one has no repository beneath it at all.
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// authorizeONURequest is the JSON body for POST
// /api/v1/provisioning/olts/{oltId}/authorize-onu.
//
// Neither field is validated for shape beyond non-empty here: Port comes
// from Palladium's own blacklist-check data (see
// internal/diagnostics/kontron.BlacklistEntry.Interface) once a caller
// has it, and SerialNumber likewise — see
// internal/provisioning/kontron.Client's own ErrInvalidInterface/
// ErrInvalidSerialNumber doc comments for the one thing that actually is
// validated (no embedded newline), enforced further down the stack, not
// here.
type authorizeONURequest struct {
	Port         string `json:"port"`
	SerialNumber string `json:"serial_number"`
}

func (req authorizeONURequest) validate() error {
	if req.Port == "" {
		return apperror.Invalid("port is required")
	}
	if req.SerialNumber == "" {
		return apperror.Invalid("serial_number is required")
	}
	return nil
}

// authorizeONUResponse is the JSON representation of AuthorizeONU's
// result: the interface it assigned the new ONU. DeauthorizationHandler
// reuses this same shape for DeauthorizeONU's result — both are exactly
// "the interface this call acted on," so a second, identically-shaped
// type would add nothing.
type authorizeONUResponse struct {
	Interface string `json:"interface"`
}

// authorizeAndCreateDeviceRequest is the JSON body for POST
// /api/v1/provisioning/olts/{oltId}/authorize-and-create-device. Port and
// SerialNumber are the same authorize-onu inputs authorizeONURequest
// carries; the rest is exactly internal/inventory/httpapi's own
// deviceRequest shape (duplicated here rather than imported -- that type
// is unexported in its own package, and this endpoint's request body is
// a distinct wire contract even though it constructs the same domain
// type underneath).
type authorizeAndCreateDeviceRequest struct {
	Port          string     `json:"port"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	RackID        *uuid.UUID `json:"rack_id"`
	DeviceModelID uuid.UUID  `json:"device_model_id"`
	SerialNumber  string     `json:"serial_number"`
	AssetTag      string     `json:"asset_tag"`
	Status        string     `json:"status"`
}

func (req authorizeAndCreateDeviceRequest) validate() error {
	if req.Port == "" {
		return apperror.Invalid("port is required")
	}
	if req.SerialNumber == "" {
		return apperror.Invalid("serial_number is required")
	}
	return nil
}

// toDevice converts req into a domain inventory.Device with no ID --
// AuthorizeAndCreateDeviceService's own deviceCreator (the real
// inventory DeviceService) generates one, the same as every other
// Device-creating caller.
func (req authorizeAndCreateDeviceRequest) toDevice() inventory.Device {
	return inventory.Device{
		Metadata: inventory.Metadata{
			Name:        req.Name,
			Description: req.Description,
		},
		RackID:        req.RackID,
		DeviceModelID: req.DeviceModelID,
		SerialNumber:  req.SerialNumber,
		AssetTag:      req.AssetTag,
		Status:        inventory.DeviceStatus(req.Status),
	}
}

// authorizeAndCreateDeviceResponse is the JSON representation of
// AuthorizeAndCreateDevice's result: the created Device's fields
// (matching internal/inventory/httpapi's own deviceResponse field names
// exactly, so the frontend can parse this response with the same DeviceDto
// shape it already uses for POST /devices) plus the OLT interface the
// ONU was authorized on.
type authorizeAndCreateDeviceResponse struct {
	Interface     string     `json:"interface"`
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	RackID        *uuid.UUID `json:"rack_id"`
	DeviceModelID uuid.UUID  `json:"device_model_id"`
	SerialNumber  string     `json:"serial_number"`
	AssetTag      string     `json:"asset_tag"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func newAuthorizeAndCreateDeviceResponse(device inventory.Device, iface string) authorizeAndCreateDeviceResponse {
	return authorizeAndCreateDeviceResponse{
		Interface:     iface,
		ID:            device.ID,
		Name:          device.Name,
		Description:   device.Description,
		RackID:        device.RackID,
		DeviceModelID: device.DeviceModelID,
		SerialNumber:  device.SerialNumber,
		AssetTag:      device.AssetTag,
		Status:        string(device.Status),
		CreatedAt:     device.CreatedAt,
		UpdatedAt:     device.UpdatedAt,
	}
}
