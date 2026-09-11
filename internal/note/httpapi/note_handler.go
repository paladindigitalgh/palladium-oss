package httpapi

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/note"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// noteService is the seam NoteHandler depends on instead of a concrete
// *service.NoteService, so handler tests can exercise HTTP behavior
// against a fake.
type noteService interface {
	Create(ctx context.Context, n note.Note) (note.Note, error)
	ListByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]note.Note, error)
}

// userLookup is the seam NoteHandler uses to resolve the current
// AuthorFirstName/AuthorLastName snapshot at Create time — the minimal
// slice of auth.UserRepository it actually needs, not the whole
// interface, the same "depend on the seam, not the concrete type"
// reasoning noteService above already follows.
type userLookup interface {
	GetByID(ctx context.Context, id uuid.UUID) (auth.User, error)
}

// NoteHandler serves the Note domain's two REST endpoints:
//
//	POST /api/v1/notes
//	GET  /api/v1/notes?entity_type=&entity_id=
//
// List's two query parameters are both required — the same "an unbounded
// listing with no entity filter has no legitimate UI use case" reasoning
// internal/event/httpapi.EventHandler's own List documents, applied here
// identically: a Notes section always belongs to one Customer, Device, or
// Service.
type NoteHandler struct {
	notes noteService
	users userLookup
}

// NewNoteHandler builds a NoteHandler.
func NewNoteHandler(notes noteService, users userLookup) *NoteHandler {
	return &NoteHandler{notes: notes, users: users}
}

// Create handles POST /api/v1/notes. AuthorUserID and AuthorEmail are
// never read from the request body — they come from the caller's
// authenticated JWT claims, the same way workflow_handler.go's Create
// stamps RequestedByUserID. Every route this handler is mounted under
// requires auth.Middleware (see internal/server/router.go), so claims
// are always present in practice; if they were somehow absent, the zero
// UUID left on AuthorUserID fails note.Note.Validate() with a clear
// "author_user_id is required" error rather than silently attributing
// the Note to nobody.
//
// AuthorFirstName/AuthorLastName cannot come from claims the same way —
// a JWT carries only ID and Email (see auth.Claims's doc comment), never
// a display name that could go stale mid-session — so this handler looks
// the caller up by ID instead, once, at write time, and snapshots
// whatever FirstName/LastName that User currently has (see note.Note's
// doc comment on why this is a snapshot, not a read-time join). A lookup
// failure is not fatal to creating the Note: it just leaves both blank,
// the same as any User who never set a name, so a transient lookup
// problem never blocks an operator from recording a Note.
func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req noteCreateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	n := note.Note{
		EntityType: req.EntityType,
		EntityID:   req.EntityID,
		Body:       req.Body,
	}
	if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
		n.AuthorUserID = claims.UserID
		n.AuthorEmail = claims.Email

		if author, err := h.users.GetByID(r.Context(), claims.UserID); err == nil {
			n.AuthorFirstName = author.FirstName
			n.AuthorLastName = author.LastName
		}
	}

	created, err := h.notes.Create(r.Context(), n)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newNoteResponse(created))
}

// List handles GET /api/v1/notes.
func (h *NoteHandler) List(w http.ResponseWriter, r *http.Request) {
	entityType := r.URL.Query().Get("entity_type")
	if entityType == "" {
		httpx.WriteError(w, apperror.Invalid("entity_type is required"))
		return
	}

	entityID, err := uuid.Parse(r.URL.Query().Get("entity_id"))
	if err != nil {
		httpx.WriteError(w, apperror.Invalid("entity_id must be a valid UUID"))
		return
	}

	notes, err := h.notes.ListByEntity(r.Context(), entityType, entityID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newNoteListResponse(notes))
}
