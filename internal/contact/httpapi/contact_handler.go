package httpapi

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/contact"
	"github.com/paladindigitalgh/palladium-oss/internal/customer"
	"github.com/paladindigitalgh/palladium-oss/internal/event"
	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// contactService is the seam ContactHandler depends on instead of a
// concrete *service.ContactService — the same reasoning
// internal/location/httpapi's locationService interface documents: it
// lets handler tests exercise HTTP behavior (status codes, JSON shapes,
// routing, error mapping) against a fake, with no real service,
// repository, or database involved. Unexported for the same reason
// locationService is: Go interfaces are satisfied structurally, so
// nothing outside this package needs to name it.
type contactService interface {
	Get(ctx context.Context, id uuid.UUID) (contact.Contact, error)
	List(ctx context.Context) ([]contact.Contact, error)
	Create(ctx context.Context, c contact.Contact) (contact.Contact, error)
	Update(ctx context.Context, c contact.Contact) (contact.Contact, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// customerGetter is the seam ContactHandler uses to resolve the owning
// Customer's Name for its Event message — the minimal slice of
// customer.CustomerRepository it actually needs, the same "depend on
// the seam, not the concrete type" reasoning contactService above
// already follows. ContactService itself is deliberately not grown a
// CustomerRepository dependency for this — see
// internal/inventory/httpapi.DeviceHandler's own eventRecorder for why
// this kind of display-friendly, cross-entity lookup lives in the
// handler layer instead.
type customerGetter interface {
	Get(ctx context.Context, id uuid.UUID) (customer.Customer, error)
}

// eventRecorder is the seam ContactHandler uses to write an operational
// Event after a successful Create.
type eventRecorder interface {
	Create(ctx context.Context, e event.Event) (event.Event, error)
}

// ContactHandler serves the Contact REST endpoints:
//
//	POST   /api/v1/contacts
//	GET    /api/v1/contacts
//	GET    /api/v1/contacts/{id}
//	PUT    /api/v1/contacts/{id}
//	DELETE /api/v1/contacts/{id}
//
// It depends only on contactService — never a repository directly — so
// it has no knowledge of PostgreSQL, SQL, or any storage technology.
// Every method is a thin decode/delegate/translate, with no business
// logic: that is ContactService's job.
type ContactHandler struct {
	contacts  contactService
	customers customerGetter
	events    eventRecorder
}

// NewContactHandler builds a ContactHandler.
func NewContactHandler(contacts contactService, customers customerGetter, events eventRecorder) *ContactHandler {
	return &ContactHandler{contacts: contacts, customers: customers, events: events}
}

// Create handles POST /api/v1/contacts.
func (h *ContactHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req contactRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.contacts.Create(r.Context(), req.toContact(uuid.Nil))
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
		EntityType:  "contact",
		EntityID:    created.ID,
		Type:        "contact.created",
		Message:     fmt.Sprintf("Added contact %s to %s", created.Name, customerName),
		ActorUserID: actorUserID,
		Metadata:    map[string]any{"customer_id": created.CustomerID.String()},
	}); err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newContactResponse(created))
}

// List handles GET /api/v1/contacts.
func (h *ContactHandler) List(w http.ResponseWriter, r *http.Request) {
	contacts, err := h.contacts.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newContactListResponse(contacts))
}

// Get handles GET /api/v1/contacts/{id}.
func (h *ContactHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	c, err := h.contacts.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newContactResponse(c))
}

// Update handles PUT /api/v1/contacts/{id}.
func (h *ContactHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req contactRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.contacts.Update(r.Context(), req.toContact(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newContactResponse(updated))
}

// Delete handles DELETE /api/v1/contacts/{id}.
func (h *ContactHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.contacts.Delete(r.Context(), id); err != nil {
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
