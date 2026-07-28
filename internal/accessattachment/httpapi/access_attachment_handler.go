package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessattachment"
	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// accessAttachmentService is the seam AccessAttachmentHandler depends on
// instead of a concrete *service.AccessAttachmentService — the same
// reasoning internal/serviceequipment/httpapi's serviceEquipmentService
// interface documents: it lets handler tests exercise HTTP behavior
// (status codes, JSON shapes, routing, error mapping) against a fake,
// with no real service, repository, or database involved. Unexported
// for the same reason serviceEquipmentService is: Go interfaces are
// satisfied structurally, so nothing outside this package needs to name
// it.
type accessAttachmentService interface {
	Get(ctx context.Context, id uuid.UUID) (accessattachment.AccessAttachment, error)
	List(ctx context.Context) ([]accessattachment.AccessAttachment, error)
	Create(ctx context.Context, a accessattachment.AccessAttachment) (accessattachment.AccessAttachment, error)
	Update(ctx context.Context, a accessattachment.AccessAttachment) (accessattachment.AccessAttachment, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// AccessAttachmentHandler serves the Access Attachment REST endpoints:
//
//	POST   /api/v1/access-attachments
//	GET    /api/v1/access-attachments
//	GET    /api/v1/access-attachments/{id}
//	PUT    /api/v1/access-attachments/{id}
//	DELETE /api/v1/access-attachments/{id}
//
// It depends only on accessAttachmentService — never a repository
// directly — so it has no knowledge of PostgreSQL, SQL, or any storage
// technology. Every method is a thin decode/delegate/translate, with no
// business logic: that is AccessAttachmentService's job, including the
// active-attachment-uniqueness rule (this milestone's goal 2) — this
// handler has no awareness that rule even exists, it only ever sees
// whatever error (or success) the service layer returns.
type AccessAttachmentHandler struct {
	attachments accessAttachmentService
}

// NewAccessAttachmentHandler builds an AccessAttachmentHandler.
func NewAccessAttachmentHandler(attachments accessAttachmentService) *AccessAttachmentHandler {
	return &AccessAttachmentHandler{attachments: attachments}
}

// Create handles POST /api/v1/access-attachments.
func (h *AccessAttachmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req accessAttachmentRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	created, err := h.attachments.Create(r.Context(), req.toAccessAttachment(uuid.Nil))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newAccessAttachmentResponse(created))
}

// List handles GET /api/v1/access-attachments.
func (h *AccessAttachmentHandler) List(w http.ResponseWriter, r *http.Request) {
	attachments, err := h.attachments.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newAccessAttachmentListResponse(attachments))
}

// Get handles GET /api/v1/access-attachments/{id}.
func (h *AccessAttachmentHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	a, err := h.attachments.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newAccessAttachmentResponse(a))
}

// Update handles PUT /api/v1/access-attachments/{id}.
func (h *AccessAttachmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req accessAttachmentRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	updated, err := h.attachments.Update(r.Context(), req.toAccessAttachment(id))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newAccessAttachmentResponse(updated))
}

// Delete handles DELETE /api/v1/access-attachments/{id}.
func (h *AccessAttachmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := h.attachments.Delete(r.Context(), id); err != nil {
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
