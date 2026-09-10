// Package httpapi is the Device Manufacturer domain's REST layer. It
// depends on internal/devicemanufacturer/service, never on a repository
// directly, and never exposes internal/devicemanufacturer's domain
// types over the wire — see the DTOs in this file. It mirrors
// internal/oltmodel/httpapi exactly.
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/devicemanufacturer"
)

// deviceManufacturerRequest is the JSON body for POST
// /api/v1/device-manufacturers and PUT /api/v1/device-manufacturers/{id}.
//
// It intentionally has no ID or timestamp fields. Identity is either
// server-assigned (POST) or comes from the URL path (PUT); CreatedAt and
// UpdatedAt are metadata the repository owns and a caller cannot set.
type deviceManufacturerRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// toDeviceManufacturer converts a request into a domain
// devicemanufacturer.DeviceManufacturer. id is supplied by the caller:
// uuid.Nil for Create (the repository assigns a real one), or the URL
// path parameter's UUID for Update.
func (req deviceManufacturerRequest) toDeviceManufacturer(id uuid.UUID) devicemanufacturer.DeviceManufacturer {
	return devicemanufacturer.DeviceManufacturer{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
	}
}

// setDeviceManufacturerDefaultRequest is the JSON body for PUT
// /api/v1/device-manufacturers/{id}/default -- deliberately its own tiny
// request type rather than reusing deviceManufacturerRequest, since
// setting the default is a distinct action from an ordinary edit (see
// DeviceManufacturerRepository.SetDefault's own doc comment on why they
// stay separate all the way down).
type setDeviceManufacturerDefaultRequest struct {
	IsDefault bool `json:"is_default"`
}

// deviceManufacturerResponse is the JSON representation of a
// DeviceManufacturer returned to clients. Decoupling the wire format
// from devicemanufacturer.DeviceManufacturer's Go field layout means a
// change to how the domain model is composed internally can never
// silently change the API's JSON shape.
type deviceManufacturerResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsDefault   bool      `json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func newDeviceManufacturerResponse(m devicemanufacturer.DeviceManufacturer) deviceManufacturerResponse {
	return deviceManufacturerResponse{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		IsDefault:   m.IsDefault,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// deviceManufacturerListResponse wraps a slice of manufacturers in an
// object rather than returning a bare JSON array — the same reasoning
// as internal/oltmodel/httpapi's oltModelListResponse.
type deviceManufacturerListResponse struct {
	DeviceManufacturers []deviceManufacturerResponse `json:"device_manufacturers"`
}

func newDeviceManufacturerListResponse(manufacturers []devicemanufacturer.DeviceManufacturer) deviceManufacturerListResponse {
	resp := deviceManufacturerListResponse{DeviceManufacturers: make([]deviceManufacturerResponse, len(manufacturers))}
	for i, m := range manufacturers {
		resp.DeviceManufacturers[i] = newDeviceManufacturerResponse(m)
	}
	return resp
}
