package httpapi

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/customer"
	"github.com/paladindigitalgh/palladium-oss/internal/event"
	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/location"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	domainservice "github.com/paladindigitalgh/palladium-oss/internal/service"
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

// deviceGetter is the seam ServiceEquipmentHandler uses to resolve the
// attached Device's Name for its Event message.
type deviceGetter interface {
	Get(ctx context.Context, id uuid.UUID) (inventory.Device, error)
}

// serviceGetter is the seam ServiceEquipmentHandler uses to resolve the
// owning Service's LocationID, one hop toward its Customer.
type serviceGetter interface {
	Get(ctx context.Context, id uuid.UUID) (domainservice.Service, error)
}

// locationGetter is the seam ServiceEquipmentHandler uses to resolve a
// Service's Location, one hop further toward its Customer.
type locationGetter interface {
	Get(ctx context.Context, id uuid.UUID) (location.Location, error)
}

// customerGetter is the seam ServiceEquipmentHandler uses to resolve the
// final Customer Name at the end of the Device→Service→Location→Customer
// chain its Event message names.
type customerGetter interface {
	Get(ctx context.Context, id uuid.UUID) (customer.Customer, error)
}

// eventRecorder is the seam ServiceEquipmentHandler uses to write an
// operational Event after a successful Create or Delete.
type eventRecorder interface {
	Create(ctx context.Context, e event.Event) (event.Event, error)
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
//
// The four *Getter dependencies below exist purely to build a
// human-readable Event message ("Attached device test-05 to Acme
// Corp's service") -- see internal/inventory/httpapi.DeviceHandler's own
// eventRecorder doc comment for why this display-friendly, cross-entity
// resolution lives here rather than growing ServiceEquipmentService's
// own dependencies. This is the deepest chain any handler in this
// codebase resolves for an Event message (Device is independent;
// Service→Location→Customer is three more hops) -- deliberately not
// pushed any further than what ServiceEquipmentService's own Create/
// Delete already return (the ServiceEquipment record's DeviceID and
// ServiceID), so no new fetch happens beyond what completing this one
// message requires.
type ServiceEquipmentHandler struct {
	equipment serviceEquipmentService
	devices   deviceGetter
	services  serviceGetter
	locations locationGetter
	customers customerGetter
	events    eventRecorder
}

// NewServiceEquipmentHandler builds a ServiceEquipmentHandler.
func NewServiceEquipmentHandler(
	equipment serviceEquipmentService,
	devices deviceGetter,
	services serviceGetter,
	locations locationGetter,
	customers customerGetter,
	events eventRecorder,
) *ServiceEquipmentHandler {
	return &ServiceEquipmentHandler{
		equipment: equipment,
		devices:   devices,
		services:  services,
		locations: locations,
		customers: customers,
		events:    events,
	}
}

// deviceAndCustomerNames resolves e's Device Name and the Customer Name
// at the end of its Service→Location→Customer chain, falling back to
// the raw id (as a string) at whichever step fails -- an Event's
// message should never block on this, and the Metadata's raw ids always
// let a caller recover the real records later.
func (h *ServiceEquipmentHandler) deviceAndCustomerNames(ctx context.Context, e serviceequipment.ServiceEquipment) (deviceName string, customerID uuid.UUID, customerName string) {
	deviceName = e.DeviceID.String()
	if d, err := h.devices.Get(ctx, e.DeviceID); err == nil {
		deviceName = d.Name
	}

	customerName = e.ServiceID.String()
	svc, err := h.services.Get(ctx, e.ServiceID)
	if err != nil {
		return deviceName, uuid.Nil, customerName
	}
	customerName = svc.LocationID.String()
	loc, err := h.locations.Get(ctx, svc.LocationID)
	if err != nil {
		return deviceName, uuid.Nil, customerName
	}
	customerName = loc.CustomerID.String()
	if c, err := h.customers.Get(ctx, loc.CustomerID); err == nil {
		customerName = c.Name
	}
	return deviceName, loc.CustomerID, customerName
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

	deviceName, customerID, customerName := h.deviceAndCustomerNames(r.Context(), created)

	var actorUserID *uuid.UUID
	if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
		actorUserID = &claims.UserID
	}
	if _, err := h.events.Create(r.Context(), event.Event{
		EntityType:  "service_equipment",
		EntityID:    created.ID,
		Type:        "service_equipment.attached",
		Message:     fmt.Sprintf("Attached device %s to %s's service", deviceName, customerName),
		ActorUserID: actorUserID,
		Metadata: map[string]any{
			"device_id":   created.DeviceID.String(),
			"service_id":  created.ServiceID.String(),
			"customer_id": customerID.String(),
		},
	}); err != nil {
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

// Delete handles DELETE /api/v1/service-equipment/{id}. The record is
// fetched before deletion, not after — its DeviceID/ServiceID would
// otherwise be unrecoverable once the row is gone.
func (h *ServiceEquipmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	existing, err := h.equipment.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.equipment.Delete(r.Context(), id); err != nil {
		httpx.WriteError(w, err)
		return
	}

	deviceName, customerID, customerName := h.deviceAndCustomerNames(r.Context(), existing)

	var actorUserID *uuid.UUID
	if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
		actorUserID = &claims.UserID
	}
	if _, err := h.events.Create(r.Context(), event.Event{
		EntityType:  "service_equipment",
		EntityID:    id,
		Type:        "service_equipment.detached",
		Message:     fmt.Sprintf("Detached device %s from %s's service", deviceName, customerName),
		ActorUserID: actorUserID,
		Metadata: map[string]any{
			"device_id":   existing.DeviceID.String(),
			"service_id":  existing.ServiceID.String(),
			"customer_id": customerID.String(),
		},
	}); err != nil {
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
