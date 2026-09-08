// Package httpapi is the OLT Model domain's REST layer. It depends on
// internal/oltmodel/service, never on a repository directly, and never
// exposes internal/oltmodel's domain types over the wire — see the DTOs
// in this file. It mirrors internal/provider/httpapi exactly.
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/oltmodel"
)

// oltModelRequest is the JSON body for POST /api/v1/olt-models and
// PUT /api/v1/olt-models/{id}.
//
// Vendor is a plain string here, not oltmodel.Vendor, even though that
// type would marshal to the same JSON today — the same "DTOs only"
// separation internal/provider/httpapi.providerRequest documents. The
// conversion happens once, explicitly, in toOLTModel below;
// OLTModelService.Create/Update reject an unrecognized value via
// OLTModel.Validate (see internal/oltmodel/validate.go) exactly as they
// would for a request built any other way — this handler does not
// duplicate that check (the service is where validation lives).
//
// It intentionally has no ID or timestamp fields. Identity is either
// server-assigned (POST) or comes from the URL path (PUT); CreatedAt and
// UpdatedAt are metadata the repository owns and a caller cannot set.
type oltModelRequest struct {
	Vendor       string `json:"vendor"`
	Name         string `json:"name"`
	PONPortCount int    `json:"pon_port_count"`
	Description  string `json:"description"`
}

// toOLTModel converts a request into a domain oltmodel.OLTModel. id is
// supplied by the caller: uuid.Nil for Create (the repository assigns a
// real one), or the URL path parameter's UUID for Update.
func (req oltModelRequest) toOLTModel(id uuid.UUID) oltmodel.OLTModel {
	return oltmodel.OLTModel{
		ID:           id,
		Vendor:       oltmodel.Vendor(req.Vendor),
		Name:         req.Name,
		PONPortCount: req.PONPortCount,
		Description:  req.Description,
	}
}

// oltModelResponse is the JSON representation of an OLTModel returned to
// clients. Decoupling the wire format from oltmodel.OLTModel's Go field
// layout and types means a change to how the domain model is composed
// internally can never silently change the API's JSON shape.
type oltModelResponse struct {
	ID           uuid.UUID `json:"id"`
	Vendor       string    `json:"vendor"`
	Name         string    `json:"name"`
	PONPortCount int       `json:"pon_port_count"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func newOLTModelResponse(m oltmodel.OLTModel) oltModelResponse {
	return oltModelResponse{
		ID:           m.ID,
		Vendor:       string(m.Vendor),
		Name:         m.Name,
		PONPortCount: m.PONPortCount,
		Description:  m.Description,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

// oltModelListResponse wraps a slice of OLTModels in an object rather
// than returning a bare JSON array — the same reasoning as
// internal/provider/httpapi's providerListResponse.
type oltModelListResponse struct {
	OLTModels []oltModelResponse `json:"olt_models"`
}

func newOLTModelListResponse(models []oltmodel.OLTModel) oltModelListResponse {
	resp := oltModelListResponse{OLTModels: make([]oltModelResponse, len(models))}
	for i, m := range models {
		resp.OLTModels[i] = newOLTModelResponse(m)
	}
	return resp
}
