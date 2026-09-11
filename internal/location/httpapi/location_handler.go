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
)

// locationService is the seam LocationHandler depends on instead of a
// concrete *service.LocationService — the same reasoning
// internal/customer/httpapi's customerService interface documents: it
// lets handler tests exercise HTTP behavior (status codes, JSON shapes,
// routing, error mapping) against a fake, with no real service,
// repository, or database involved. Unexported for the same reason
// customerService is: Go interfaces are satisfied structurally, so
// nothing outside this package needs to name it.
type locationService interface {
	Get(ctx context.Context, id uuid.UUID) (location.Location, error)
	List(ctx context.Context) ([]location.Location, error)
	Create(ctx context.Context, l location.Location) (location.Location, error)
	Update(ctx context.Context, l location.Location) (location.Location, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// customerGetter is the seam LocationHandler uses to resolve the owning
// Customer's Name for its Event message — see
// internal/contact/httpapi.ContactHandler's own customerGetter for the
// identical reasoning, applied here one domain over.
type customerGetter interface {
	Get(ctx context.Context, id uuid.UUID) (customer.Customer, error)
}

// eventRecorder is the seam LocationHandler uses to write an operational
// Event after a successful Create.
type eventRecorder interface {
	Create(ctx context.Context, e event.Event) (event.Event, error)
}

// LocationHandler serves the Location REST endpoints:
//
//	POST   /api/v1/locations
//	GET    /api/v1/locations
//	GET    /api/v1/locations/{id}
//	PUT    /api/v1/locations/{id}
//	DELETE /api/v1/locations/{id}
//
// It depends only on locationService — never a repository directly — so
// it has no knowledge of PostgreSQL, SQL, or any storage technology.
// Every method is a thin decode/delegate/translate, with no business
// logic: that is LocationService's job.
type LocationHandler struct {
	locations locationService
	customers customerGetter
	events    eventRecorder
}

// NewLocationHandler builds a LocationHandler.
func NewLocationHandler(locations locationService, customers customerGetter, events eventRecorder) *LocationHandler {
	return &LocationHandler{locations: locations, customers: customers, events: events}
}

// Create handles POST /api/v1/locations.
func (h *LocationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req locationRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.locations.Create(r.Context(), req.toLocation(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	customerName := created.CustomerID.String()
	if c, err := h.customers.Get(r.Context(), created.CustomerID); err == nil {
		customerName = c.Name
	}

	var actorUserID *uuid.UUID
	if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
		actorUserID = &claims.UserID
	}
	if _, err := h.events.Create(r.Context(), event.Event{
		EntityType:  "location",
		EntityID:    created.ID,
		Type:        "location.created",
		Message:     fmt.Sprintf("Added location %s to %s", created.Name, customerName),
		ActorUserID: actorUserID,
		Metadata:    map[string]any{"customer_id": created.CustomerID.String()},
	}); err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newLocationResponse(created))
}

// List handles GET /api/v1/locations.
func (h *LocationHandler) List(w http.ResponseWriter, r *http.Request) {
	locations, err := h.locations.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newLocationListResponse(locations))
}

// Get handles GET /api/v1/locations/{id}.
func (h *LocationHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	l, err := h.locations.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newLocationResponse(l))
}

// Update handles PUT /api/v1/locations/{id}.
func (h *LocationHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req locationRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.locations.Update(r.Context(), req.toLocation(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newLocationResponse(updated))
}

// Delete handles DELETE /api/v1/locations/{id}.
func (h *LocationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.locations.Delete(r.Context(), id); err != nil {
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
