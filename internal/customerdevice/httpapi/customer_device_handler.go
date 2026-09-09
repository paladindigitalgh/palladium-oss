package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/customerdevice"
	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// customerDeviceService is the seam CustomerDeviceHandler depends on
// instead of a concrete *service.CustomerDeviceService — the same
// reasoning internal/serviceequipment/httpapi's serviceEquipmentService
// interface documents: it lets handler tests exercise HTTP behavior
// against a fake, with no real service, repository, or database
// involved.
type customerDeviceService interface {
	Get(ctx context.Context, id uuid.UUID) (customerdevice.CustomerDevice, error)
	List(ctx context.Context) ([]customerdevice.CustomerDevice, error)
	Create(ctx context.Context, cd customerdevice.CustomerDevice) (customerdevice.CustomerDevice, error)
	Update(ctx context.Context, cd customerdevice.CustomerDevice) (customerdevice.CustomerDevice, error)
}

// CustomerDeviceHandler serves the Customer Device REST endpoints:
//
//	POST /api/v1/customer-devices
//	GET  /api/v1/customer-devices
//	GET  /api/v1/customer-devices/{id}
//	PUT  /api/v1/customer-devices/{id}
//
// There is deliberately no DELETE route: this domain has no Delete (see
// customerdevice.CustomerDeviceRepository's own doc comment) — detaching
// a Device is a PUT that sets detached_at, not a removal.
//
// It depends only on customerDeviceService — never a repository
// directly — so it has no knowledge of PostgreSQL, SQL, or any storage
// technology. Every method is a thin decode/delegate/translate, with no
// business logic: that is CustomerDeviceService's job, including the
// active-assignment-uniqueness and detach-blocking rules — this handler
// has no awareness either rule even exists.
type CustomerDeviceHandler struct {
	customerDevices customerDeviceService
}

// NewCustomerDeviceHandler builds a CustomerDeviceHandler.
func NewCustomerDeviceHandler(customerDevices customerDeviceService) *CustomerDeviceHandler {
	return &CustomerDeviceHandler{customerDevices: customerDevices}
}

// Create handles POST /api/v1/customer-devices.
func (h *CustomerDeviceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req customerDeviceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.customerDevices.Create(r.Context(), req.toCustomerDevice(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newCustomerDeviceResponse(created))
}

// List handles GET /api/v1/customer-devices.
func (h *CustomerDeviceHandler) List(w http.ResponseWriter, r *http.Request) {
	records, err := h.customerDevices.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newCustomerDeviceListResponse(records))
}

// Get handles GET /api/v1/customer-devices/{id}.
func (h *CustomerDeviceHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	cd, err := h.customerDevices.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newCustomerDeviceResponse(cd))
}

// Update handles PUT /api/v1/customer-devices/{id}.
func (h *CustomerDeviceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req customerDeviceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.customerDevices.Update(r.Context(), req.toCustomerDevice(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newCustomerDeviceResponse(updated))
}

func pathID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, apperror.Invalid("id must be a valid UUID")
	}
	return id, nil
}
