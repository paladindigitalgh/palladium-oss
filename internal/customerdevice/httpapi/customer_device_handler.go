package httpapi

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/customer"
	"github.com/paladindigitalgh/palladium-oss/internal/customerdevice"
	"github.com/paladindigitalgh/palladium-oss/internal/event"
	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
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

// deviceGetter is the seam CustomerDeviceHandler uses to resolve the
// attached Device's Name for its Event message.
type deviceGetter interface {
	Get(ctx context.Context, id uuid.UUID) (inventory.Device, error)
}

// customerGetter is the seam CustomerDeviceHandler uses to resolve the
// Customer's Name for its Event message. Unlike Service/ServiceEquipment
// elsewhere, CustomerDevice already carries CustomerID directly (no
// Location hop needed) — see customerdevice.CustomerDevice's own doc
// comment.
type customerGetter interface {
	Get(ctx context.Context, id uuid.UUID) (customer.Customer, error)
}

// eventRecorder is the seam CustomerDeviceHandler uses to write an
// operational Event after a successful attach (Create) or detach
// (an Update transitioning Active() true -> false).
type eventRecorder interface {
	Create(ctx context.Context, e event.Event) (event.Event, error)
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
	devices         deviceGetter
	customers       customerGetter
	events          eventRecorder
}

// NewCustomerDeviceHandler builds a CustomerDeviceHandler.
func NewCustomerDeviceHandler(customerDevices customerDeviceService, devices deviceGetter, customers customerGetter, events eventRecorder) *CustomerDeviceHandler {
	return &CustomerDeviceHandler{customerDevices: customerDevices, devices: devices, customers: customers, events: events}
}

// deviceAndCustomerNames resolves cd's Device Name and Customer Name,
// falling back to the raw id (as a string) at whichever lookup fails --
// an Event's message should never block on this.
func (h *CustomerDeviceHandler) deviceAndCustomerNames(ctx context.Context, cd customerdevice.CustomerDevice) (deviceName, customerName string) {
	deviceName = cd.DeviceID.String()
	if d, err := h.devices.Get(ctx, cd.DeviceID); err == nil {
		deviceName = d.Name
	}
	customerName = cd.CustomerID.String()
	if c, err := h.customers.Get(ctx, cd.CustomerID); err == nil {
		customerName = c.Name
	}
	return deviceName, customerName
}

func (h *CustomerDeviceHandler) recordEvent(r *http.Request, cd customerdevice.CustomerDevice, eventType, verb string) error {
	deviceName, customerName := h.deviceAndCustomerNames(r.Context(), cd)

	var actorUserID *uuid.UUID
	if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
		actorUserID = &claims.UserID
	}
	_, err := h.events.Create(r.Context(), event.Event{
		EntityType:  "customer_device",
		EntityID:    cd.ID,
		Type:        eventType,
		Message:     fmt.Sprintf("%s device %s %s %s", verb, deviceName, prepositionFor(verb), customerName),
		ActorUserID: actorUserID,
		Metadata: map[string]any{
			"device_id":   cd.DeviceID.String(),
			"customer_id": cd.CustomerID.String(),
		},
	})
	return err
}

func prepositionFor(verb string) string {
	if verb == "Detached" {
		return "from"
	}
	return "to"
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

	if err := h.recordEvent(r, created, "customer_device.attached", "Attached"); err != nil {
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

// Update handles PUT /api/v1/customer-devices/{id}. A "customer_device
// .detached" Event is recorded only when this specific call is the one
// that transitions Active() true -> false — a plain edit of an already-
// detached (or still-active) record writes no Event, since nothing
// about the attachment relationship itself actually changed.
func (h *CustomerDeviceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	existing, err := h.customerDevices.Get(r.Context(), id)
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

	if existing.Active() && !updated.Active() {
		if err := h.recordEvent(r, updated, "customer_device.detached", "Detached"); err != nil {
			httpx.WriteError(w, err)
			return
		}
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
