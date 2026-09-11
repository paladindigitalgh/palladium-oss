// Package httpapi is the Note domain's REST layer. It depends on
// internal/note/service, never on a repository directly, and never
// exposes internal/note's domain types over the wire — see the DTOs in
// this file. It mirrors internal/contact/httpapi's shape, with one
// deliberate difference: there is no Update, Get, or Delete route — only
// Create and List — the same "immutable, append-only" reasoning
// internal/event/httpapi documents for Events, applied here to a domain
// that (unlike Event) a client actually writes to.
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/note"
)

// noteCreateRequest is the JSON body for POST /api/v1/notes.
//
// It intentionally has no ID, AuthorUserID, AuthorEmail, or CreatedAt
// field: identity and the authored-by fields are server-assigned — the
// latter from the caller's authenticated JWT claims, never from
// anything a client could set itself (see note_handler.go's Create) —
// and CreatedAt is metadata the repository owns.
type noteCreateRequest struct {
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	Body       string    `json:"body"`
}

// noteResponse is the JSON representation of a Note returned to clients.
// Decoupling the wire format from note.Note's Go field layout means a
// change to how the domain model is composed internally can never
// silently change the API's JSON shape.
type noteResponse struct {
	ID              uuid.UUID `json:"id"`
	EntityType      string    `json:"entity_type"`
	EntityID        uuid.UUID `json:"entity_id"`
	AuthorUserID    uuid.UUID `json:"author_user_id"`
	AuthorEmail     string    `json:"author_email"`
	AuthorFirstName string    `json:"author_first_name"`
	AuthorLastName  string    `json:"author_last_name"`
	Body            string    `json:"body"`
	CreatedAt       time.Time `json:"created_at"`
}

func newNoteResponse(n note.Note) noteResponse {
	return noteResponse{
		ID:              n.ID,
		EntityType:      n.EntityType,
		EntityID:        n.EntityID,
		AuthorUserID:    n.AuthorUserID,
		AuthorEmail:     n.AuthorEmail,
		AuthorFirstName: n.AuthorFirstName,
		AuthorLastName:  n.AuthorLastName,
		Body:            n.Body,
		CreatedAt:       n.CreatedAt,
	}
}

// noteListResponse wraps a slice of notes in an object rather than
// returning a bare JSON array — the same reasoning as every other list
// response in this codebase.
type noteListResponse struct {
	Notes []noteResponse `json:"notes"`
}

func newNoteListResponse(notes []note.Note) noteListResponse {
	resp := noteListResponse{Notes: make([]noteResponse, len(notes))}
	for i, n := range notes {
		resp.Notes[i] = newNoteResponse(n)
	}
	return resp
}
