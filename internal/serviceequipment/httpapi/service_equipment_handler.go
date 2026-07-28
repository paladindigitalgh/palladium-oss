package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// serviceEquipmentService is the seam ServiceEquipmentHandler depends on
// instead of a concrete *service.ServiceEquipmentService — the same
// reasoning internal/service/httpapi's serviceService interface
// documents: it lets handler tests exercise HTTP behavior (status codes,
// JSON shapes, routing, error mapping) against a fake, with no real
// service, repository, or database involved. Unexported for the same
// reason serviceService is: Go interfaces are satisfied structurally, so
// nothing outside this package needs to name it.
type serviceEquipmentService interface {
	Get(ctx context.Context, id uuid.UUID) (serviceequipment.ServiceEquipment, error)
	List(ctx context.Context) ([]serviceequipment.ServiceEquipment, error)
	Create(ctx context.Context, e serviceequipment.ServiceEquipment) (serviceequipment.ServiceEquipment, error)
	Update(ctx context.Context, e serviceequipment.ServiceEquipment) (serviceequipment.ServiceEquipment, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// ServiceEquipmentHandler serves the Service Equipment REST endpoints:
//
//	POST   /api/v1/service-equipment
//	GET    /api/v1/service-equipment
//	GET    /api/v1/service-equipment/{id}
//	PUT    /api/v1/service-equipment/{id}
//	DELETE /api/v1/service-equipment/{id}
//
// It depends only on serviceEquipmentService — never a repository
// directly — so it has no knowledge of PostgreSQL, SQL, or any storage
// technology. Every method is a thin decode/delegate/translate, with no
// business logic: that is ServiceEquipmentService's job, including the
// active-assignment-uniqueness rule (goal 2) — this handler has no
// awareness that rule even exists, it only ever sees whatever error (or
// success) the service layer returns.
type ServiceEquipmentHandler struct {
	equipment serviceEquipmentService
}

// NewServiceEquipmentHandler builds a ServiceEquipmentHandler.
func NewServiceEquipmentHandler(equipment serviceEquipmentService) *ServiceEquipmentHandler {
	return &ServiceEquipmentHandler{equipment: equipment}
}

// Create handles POST /api/v1/service-equipment.
func (h *ServiceEquipmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req serviceEquipmentRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.equipment.Create(r.Context(), req.toServiceEquipment(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newServiceEquipmentResponse(created))
}

// List handles GET /api/v1/service-equipment.
func (h *ServiceEquipmentHandler) List(w http.ResponseWriter, r *http.Request) {
	equipment, err := h.equipment.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newServiceEquipmentListResponse(equipment))
}

// Get handles GET /api/v1/service-equipment/{id}.
func (h *ServiceEquipmentHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	e, err := h.equipment.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newServiceEquipmentResponse(e))
}

// Update handles PUT /api/v1/service-equipment/{id}.
func (h *ServiceEquipmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req serviceEquipmentRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.equipment.Update(r.Context(), req.toServiceEquipment(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newServiceEquipmentResponse(updated))
}

// Delete handles DELETE /api/v1/service-equipment/{id}.
func (h *ServiceEquipmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.equipment.Delete(r.Context(), id); err != nil {
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
