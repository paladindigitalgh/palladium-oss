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
	"github.com/paladindigitalgh/palladium-oss/internal/location"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	domainservice "github.com/paladindigitalgh/palladium-oss/internal/service"
)

// serviceService is the seam ServiceHandler depends on instead of a
// concrete *service.ServiceService — the same reasoning
// internal/product/httpapi's productService interface documents: it lets
// handler tests exercise HTTP behavior (status codes, JSON shapes,
// routing, error mapping) against a fake, with no real service,
// repository, or database involved. Unexported for the same reason
// productService is: Go interfaces are satisfied structurally, so
// nothing outside this package needs to name it.
type serviceService interface {
	Get(ctx context.Context, id uuid.UUID) (domainservice.Service, error)
	List(ctx context.Context) ([]domainservice.Service, error)
	Create(ctx context.Context, s domainservice.Service) (domainservice.Service, error)
	Update(ctx context.Context, s domainservice.Service) (domainservice.Service, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// locationGetter is the seam ServiceHandler uses to resolve a Service's
// Location (and, through it, its owning Customer) for its Event
// message — the minimal slice of location.LocationRepository it
// actually needs, the same "depend on the seam, not the concrete type"
// reasoning serviceService above already follows. ServiceService itself
// is deliberately not grown this dependency — see
// internal/inventory/httpapi.DeviceHandler's own eventRecorder for why
// this kind of display-friendly, cross-entity lookup lives in the
// handler layer instead.
type locationGetter interface {
	Get(ctx context.Context, id uuid.UUID) (location.Location, error)
}

// customerGetter is the seam ServiceHandler uses to resolve a Location's
// owning Customer Name.
type customerGetter interface {
	Get(ctx context.Context, id uuid.UUID) (customer.Customer, error)
}

// eventRecorder is the seam ServiceHandler uses to write an operational
// Event after a successful Create or Delete.
type eventRecorder interface {
	Create(ctx context.Context, e event.Event) (event.Event, error)
}

// ServiceHandler serves the Service REST endpoints:
//
//	POST   /api/v1/services
//	GET    /api/v1/services
//	GET    /api/v1/services/{id}
//	PUT    /api/v1/services/{id}
//	DELETE /api/v1/services/{id}
//
// It depends only on serviceService — never a repository directly — so
// it has no knowledge of PostgreSQL, SQL, or any storage technology.
// Every method is a thin decode/delegate/translate, with no business
// logic: that is ServiceService's job.
type ServiceHandler struct {
	services  serviceService
	locations locationGetter
	customers customerGetter
	events    eventRecorder
}

// NewServiceHandler builds a ServiceHandler.
func NewServiceHandler(services serviceService, locations locationGetter, customers customerGetter, events eventRecorder) *ServiceHandler {
	return &ServiceHandler{services: services, locations: locations, customers: customers, events: events}
}

// customerNameForLocation resolves locationID's owning Customer's Name,
// falling back to the raw id (as a string) when either lookup fails --
// an Event's message should never block on this, and the Metadata's
// raw ids always let a caller recover the real records later.
func (h *ServiceHandler) customerNameForLocation(ctx context.Context, locationID uuid.UUID) (customerID uuid.UUID, customerName string) {
	loc, err := h.locations.Get(ctx, locationID)
	if err != nil {
		return uuid.Nil, locationID.String()
	}
	if c, err := h.customers.Get(ctx, loc.CustomerID); err == nil {
		return loc.CustomerID, c.Name
	}
	return loc.CustomerID, loc.CustomerID.String()
}

// Create handles POST /api/v1/services.
func (h *ServiceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req serviceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.services.Create(r.Context(), req.toService(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	customerID, customerName := h.customerNameForLocation(r.Context(), created.LocationID)

	var actorUserID *uuid.UUID
	if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
		actorUserID = &claims.UserID
	}
	if _, err := h.events.Create(r.Context(), event.Event{
		EntityType:  "service",
		EntityID:    created.ID,
		Type:        "service.created",
		Message:     fmt.Sprintf("Added service for %s", customerName),
		ActorUserID: actorUserID,
		Metadata:    map[string]any{"customer_id": customerID.String()},
	}); err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newServiceResponse(created))
}

// List handles GET /api/v1/services.
func (h *ServiceHandler) List(w http.ResponseWriter, r *http.Request) {
	services, err := h.services.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newServiceListResponse(services))
}

// Get handles GET /api/v1/services/{id}.
func (h *ServiceHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	s, err := h.services.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newServiceResponse(s))
}

// Update handles PUT /api/v1/services/{id}.
func (h *ServiceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req serviceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.services.Update(r.Context(), req.toService(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newServiceResponse(updated))
}

// Delete handles DELETE /api/v1/services/{id}. The Service is fetched
// before deletion, not after — its LocationID (and, through it, the
// owning Customer's Name) would otherwise be unrecoverable once the row
// is gone.
func (h *ServiceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	existing, err := h.services.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.services.Delete(r.Context(), id); err != nil {
		httpx.WriteError(w, err)
		return
	}

	customerID, customerName := h.customerNameForLocation(r.Context(), existing.LocationID)

	var actorUserID *uuid.UUID
	if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
		actorUserID = &claims.UserID
	}
	if _, err := h.events.Create(r.Context(), event.Event{
		EntityType:  "service",
		EntityID:    id,
		Type:        "service.removed",
		Message:     fmt.Sprintf("Removed service for %s", customerName),
		ActorUserID: actorUserID,
		Metadata:    map[string]any{"customer_id": customerID.String()},
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
