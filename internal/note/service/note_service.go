// Package service is the Note domain's business logic layer. It sits
// between the HTTP layer and the repository layer: HTTP handlers never
// call a repository directly (see internal/note/httpapi), and
// repositories never validate or otherwise reason about business rules
// (see internal/note/postgres, which trusts its caller) — this is where
// those two responsibilities meet, mirroring internal/contact/service
// exactly.
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/note"
)

// NoteService is the Note domain's business logic.
//
// It depends only on note.NoteRepository — not clock.Clock, for the same
// reason internal/contact/service.ContactService does not: timestamps
// are already the repository's responsibility, and this service has no
// business rule that needs to reason about "now".
type NoteService struct {
	notes note.NoteRepository
}

// NewNoteService builds a NoteService.
func NewNoteService(notes note.NoteRepository) *NoteService {
	return &NoteService{notes: notes}
}

// Create validates n and, if valid, persists it.
//
// Validation happens here — not in the repository, which trusts its
// caller, and not in the HTTP handler, which would then need to
// duplicate this for every other future caller of NoteService — so every
// caller gets the same guarantee for free, and invalid input never costs
// a database round trip. See internal/contact/service.ContactService.Create
// for the identical reasoning applied to Contacts.
func (s *NoteService) Create(ctx context.Context, n note.Note) (note.Note, error) {
	if err := n.Validate(); err != nil {
		return note.Note{}, err
	}
	return s.notes.Create(ctx, n)
}

// ListByEntity returns every Note recorded for one entity, newest first.
func (s *NoteService) ListByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]note.Note, error) {
	return s.notes.ListByEntity(ctx, entityType, entityID)
}
