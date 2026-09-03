package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/contact"
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
	contacts contactService
}

// NewContactHandler builds a ContactHandler.
func NewContactHandler(contacts contactService) *ContactHandler {
	return &ContactHandler{contacts: contacts}
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
