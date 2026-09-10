package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/devicemodel"
	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// deviceModelService is the seam DeviceModelHandler depends on instead
// of a concrete *service.DeviceModelService — the same reasoning
// internal/oltmodel/httpapi's oltModelService interface documents: it
// lets handler tests exercise HTTP behavior (status codes, JSON shapes,
// routing, error mapping) against a fake, with no real service,
// repository, or database involved.
type deviceModelService interface {
	Get(ctx context.Context, id uuid.UUID) (devicemodel.DeviceModel, error)
	List(ctx context.Context) ([]devicemodel.DeviceModel, error)
	Create(ctx context.Context, m devicemodel.DeviceModel) (devicemodel.DeviceModel, error)
	Update(ctx context.Context, m devicemodel.DeviceModel) (devicemodel.DeviceModel, error)
	Delete(ctx context.Context, id uuid.UUID) error
	SetDefault(ctx context.Context, id uuid.UUID, isDefault bool) error
}

// DeviceModelHandler serves the Device Model REST endpoints:
//
//	POST   /api/v1/device-models
//	GET    /api/v1/device-models
//	GET    /api/v1/device-models/{id}
//	PUT    /api/v1/device-models/{id}
//	PUT    /api/v1/device-models/{id}/default
//	DELETE /api/v1/device-models/{id}
//
// It depends only on deviceModelService — never a repository directly —
// so it has no knowledge of PostgreSQL, SQL, or any storage technology.
// Every method is a thin decode/delegate/translate, with no business
// logic: that is DeviceModelService's job.
type DeviceModelHandler struct {
	models deviceModelService
}

// NewDeviceModelHandler builds a DeviceModelHandler.
func NewDeviceModelHandler(models deviceModelService) *DeviceModelHandler {
	return &DeviceModelHandler{models: models}
}

// Create handles POST /api/v1/device-models.
func (h *DeviceModelHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req deviceModelRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.models.Create(r.Context(), req.toDeviceModel(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newDeviceModelResponse(created))
}

// List handles GET /api/v1/device-models.
func (h *DeviceModelHandler) List(w http.ResponseWriter, r *http.Request) {
	models, err := h.models.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newDeviceModelListResponse(models))
}

// Get handles GET /api/v1/device-models/{id}.
func (h *DeviceModelHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	m, err := h.models.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newDeviceModelResponse(m))
}

// Update handles PUT /api/v1/device-models/{id}.
func (h *DeviceModelHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req deviceModelRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.models.Update(r.Context(), req.toDeviceModel(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newDeviceModelResponse(updated))
}

// SetDefault handles PUT /api/v1/device-models/{id}/default.
func (h *DeviceModelHandler) SetDefault(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req setDeviceModelDefaultRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.models.SetDefault(r.Context(), id, req.IsDefault); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.models.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newDeviceModelResponse(updated))
}

// Delete handles DELETE /api/v1/device-models/{id}.
func (h *DeviceModelHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.models.Delete(r.Context(), id); err != nil {
		httpx.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func pathID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, apperror.Invalid("id must be a valid UUID")
	}
	return id, nil
}
