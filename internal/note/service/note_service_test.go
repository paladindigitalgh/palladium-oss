package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/note"
	"github.com/paladindigitalgh/palladium-oss/internal/note/service"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeNoteRepository is an in-memory note.NoteRepository. Like
// internal/contact/service/contact_service_test.go's
// fakeContactRepository, it exists so NoteService's business logic —
// validate, then delegate — is tested without a real database;
// internal/note/postgres/note_test.go already covers the repository
// itself against real PostgreSQL. It tracks whether Create was actually
// invoked, which is what lets
// TestNoteServiceCreateRejectsInvalidNoteWithoutPersisting prove
// validation happens before any repository call.
type fakeNoteRepository struct {
	byID         map[uuid.UUID]note.Note
	createCalled bool
}

func newFakeNoteRepository(notes ...note.Note) *fakeNoteRepository {
	f := &fakeNoteRepository{byID: make(map[uuid.UUID]note.Note)}
	for _, n := range notes {
		f.byID[n.ID] = n
	}
	return f
}

func (f *fakeNoteRepository) Create(_ context.Context, n note.Note) (note.Note, error) {
	f.createCalled = true
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	f.byID[n.ID] = n
	return n, nil
}

func (f *fakeNoteRepository) ListByEntity(_ context.Context, entityType string, entityID uuid.UUID) ([]note.Note, error) {
	notes := make([]note.Note, 0)
	for _, n := range f.byID {
		if n.EntityType == entityType && n.EntityID == entityID {
			notes = append(notes, n)
		}
	}
	return notes, nil
}

var _ note.NoteRepository = (*fakeNoteRepository)(nil)

func validNote() note.Note {
	return note.Note{
		EntityType:   "customer",
		EntityID:     uuid.New(),
		AuthorUserID: uuid.New(),
		AuthorEmail:  "operator@example.com",
		Body:         "Called the customer back, issue resolved.",
	}
}

func TestNoteServiceCreateSucceeds(t *testing.T) {
	repo := newFakeNoteRepository()
	svc := service.NewNoteService(repo)

	created, err := svc.Create(context.Background(), validNote())
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if !repo.createCalled {
		t.Error("repository Create() was never called")
	}
}

func TestNoteServiceCreateRejectsInvalidNoteWithoutPersisting(t *testing.T) {
	repo := newFakeNoteRepository()
	svc := service.NewNoteService(repo)

	_, err := svc.Create(context.Background(), note.Note{}) // no EntityType, EntityID, AuthorUserID, Body

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestNoteServiceListByEntityDelegatesToRepository(t *testing.T) {
	entityID := uuid.New()
	a := validNote()
	a.ID = uuid.New()
	a.EntityID = entityID
	b := validNote()
	b.ID = uuid.New()
	b.EntityID = entityID
	other := validNote()
	other.ID = uuid.New()
	repo := newFakeNoteRepository(a, b, other)
	svc := service.NewNoteService(repo)

	notes, err := svc.ListByEntity(context.Background(), "customer", entityID)
	if err != nil {
		t.Fatalf("ListByEntity() = %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("len(ListByEntity()) = %d, want 2", len(notes))
	}
}
