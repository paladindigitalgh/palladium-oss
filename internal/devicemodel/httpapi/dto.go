// Package httpapi is the Device Model domain's REST layer. It depends
// on internal/devicemodel/service, never on a repository directly, and
// never exposes internal/devicemodel's domain types over the wire — see
// the DTOs in this file. It mirrors internal/oltmodel/httpapi exactly.
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/devicemodel"
)

// deviceModelRequest is the JSON body for POST /api/v1/device-models and
// PUT /api/v1/device-models/{id}.
//
// It intentionally has no ID or timestamp fields. Identity is either
// server-assigned (POST) or comes from the URL path (PUT); CreatedAt and
// UpdatedAt are metadata the repository owns and a caller cannot set.
type deviceModelRequest struct {
	ManufacturerID uuid.UUID `json:"manufacturer_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
}

// toDeviceModel converts a request into a domain
// devicemodel.DeviceModel. id is supplied by the caller: uuid.Nil for
// Create (the repository assigns a real one), or the URL path
// parameter's UUID for Update.
func (req deviceModelRequest) toDeviceModel(id uuid.UUID) devicemodel.DeviceModel {
	return devicemodel.DeviceModel{
		ID:             id,
		ManufacturerID: req.ManufacturerID,
		Name:           req.Name,
		Description:    req.Description,
	}
}

// deviceModelResponse is the JSON representation of a DeviceModel
// returned to clients. Decoupling the wire format from
// devicemodel.DeviceModel's Go field layout means a change to how the
// domain model is composed internally can never silently change the
// API's JSON shape.
type deviceModelResponse struct {
	ID             uuid.UUID `json:"id"`
	ManufacturerID uuid.UUID `json:"manufacturer_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func newDeviceModelResponse(m devicemodel.DeviceModel) deviceModelResponse {
	return deviceModelResponse{
		ID:             m.ID,
		ManufacturerID: m.ManufacturerID,
		Name:           m.Name,
		Description:    m.Description,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

// deviceModelListResponse wraps a slice of models in an object rather
// than returning a bare JSON array — the same reasoning as
// internal/oltmodel/httpapi's oltModelListResponse.
type deviceModelListResponse struct {
	DeviceModels []deviceModelResponse `json:"device_models"`
}

func newDeviceModelListResponse(models []devicemodel.DeviceModel) deviceModelListResponse {
	resp := deviceModelListResponse{DeviceModels: make([]deviceModelResponse, len(models))}
	for i, m := range models {
		resp.DeviceModels[i] = newDeviceModelResponse(m)
	}
	return resp
}
