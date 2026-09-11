package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/note"
	"github.com/paladindigitalgh/palladium-oss/internal/note/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeNoteService is the seam httpapi.NoteHandler depends on (see its
// unexported noteService interface in note_handler.go). It lets these
// tests exercise HTTP-only concerns -- status codes, JSON shapes,
// claims-to-author-field capture, error translation -- without a real
// service, repository, or database; internal/note/service and
// internal/note/postgres each have their own tests for the layers below
// this one.
type fakeNoteService struct {
	notes []note.Note
	err   error

	// gotCreate records the Note NoteHandler.Create last passed through,
	// so tests can assert on AuthorUserID/AuthorEmail without a real
	// repository behind it.
	gotCreate note.Note
}

func (f *fakeNoteService) Create(_ context.Context, n note.Note) (note.Note, error) {
	if f.err != nil {
		return note.Note{}, f.err
	}
	f.gotCreate = n
	n.ID = uuid.New()
	return n, nil
}

func (f *fakeNoteService) ListByEntity(context.Context, string, uuid.UUID) ([]note.Note, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.notes, nil
}

// fakeUserLookup is the seam httpapi.NoteHandler's userLookup dependency
// uses. Keyed by ID so a test can plant exactly the author a given
// claims.UserID should resolve to, without a real auth.UserRepository.
type fakeUserLookup struct {
	byID map[uuid.UUID]auth.User
}

func (f *fakeUserLookup) GetByID(_ context.Context, id uuid.UUID) (auth.User, error) {
	user, ok := f.byID[id]
	if !ok {
		return auth.User{}, apperror.NotFound("user not found")
	}
	return user, nil
}

func TestNoteHandlerCreateCapturesAuthorFromClaims(t *testing.T) {
	svc := &fakeNoteService{}
	claims := auth.Claims{UserID: uuid.New(), Email: "jane@example.com"}
	users := &fakeUserLookup{byID: map[uuid.UUID]auth.User{
		claims.UserID: {ID: claims.UserID, Email: claims.Email, FirstName: "Jane", LastName: "Doe"},
	}}
	h := httpapi.NewNoteHandler(svc, users)

	entityID := uuid.New()
	body := `{"entity_type":"customer","entity_id":"` + entityID.String() + `","body":"Called back, resolved."}`

	req := httptest.NewRequest(http.MethodPost, "/notes", strings.NewReader(body))
	req = req.WithContext(auth.ContextWithClaims(req.Context(), claims))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if svc.gotCreate.AuthorUserID != claims.UserID {
		t.Errorf("AuthorUserID = %v, want %v", svc.gotCreate.AuthorUserID, claims.UserID)
	}
	if svc.gotCreate.AuthorEmail != claims.Email {
		t.Errorf("AuthorEmail = %q, want %q", svc.gotCreate.AuthorEmail, claims.Email)
	}
	if svc.gotCreate.AuthorFirstName != "Jane" || svc.gotCreate.AuthorLastName != "Doe" {
		t.Errorf("author name = %q %q, want Jane Doe", svc.gotCreate.AuthorFirstName, svc.gotCreate.AuthorLastName)
	}
	if svc.gotCreate.EntityType != "customer" || svc.gotCreate.EntityID != entityID {
		t.Errorf("entity = %s/%v, want customer/%v", svc.gotCreate.EntityType, svc.gotCreate.EntityID, entityID)
	}
	if svc.gotCreate.Body != "Called back, resolved." {
		t.Errorf("Body = %q, want %q", svc.gotCreate.Body, "Called back, resolved.")
	}

	var resp struct {
		AuthorEmail     string `json:"author_email"`
		AuthorFirstName string `json:"author_first_name"`
		AuthorLastName  string `json:"author_last_name"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.AuthorEmail != "jane@example.com" {
		t.Errorf("response author_email = %q, want %q", resp.AuthorEmail, "jane@example.com")
	}
	if resp.AuthorFirstName != "Jane" || resp.AuthorLastName != "Doe" {
		t.Errorf("response author name = %q %q, want Jane Doe", resp.AuthorFirstName, resp.AuthorLastName)
	}
}

func TestNoteHandlerCreateRejectsMalformedJSON(t *testing.T) {
	h := httpapi.NewNoteHandler(&fakeNoteService{}, &fakeUserLookup{})

	req := httptest.NewRequest(http.MethodPost, "/notes", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestNoteHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := &fakeNoteService{err: apperror.Invalid("body: is required")}
	h := httpapi.NewNoteHandler(svc, &fakeUserLookup{})

	req := httptest.NewRequest(http.MethodPost, "/notes", strings.NewReader(`{"entity_type":"customer","entity_id":"`+uuid.New().String()+`"}`))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestNoteHandlerListRequiresEntityType(t *testing.T) {
	h := httpapi.NewNoteHandler(&fakeNoteService{}, &fakeUserLookup{})

	req := httptest.NewRequest(http.MethodGet, "/notes?entity_id="+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestNoteHandlerListRequiresValidEntityID(t *testing.T) {
	h := httpapi.NewNoteHandler(&fakeNoteService{}, &fakeUserLookup{})

	req := httptest.NewRequest(http.MethodGet, "/notes?entity_type=customer&entity_id=not-a-uuid", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestNoteHandlerListReturnsNotes(t *testing.T) {
	entityID := uuid.New()
	svc := &fakeNoteService{notes: []note.Note{
		{ID: uuid.New(), EntityType: "customer", EntityID: entityID, AuthorEmail: "jane@example.com", Body: "First"},
	}}
	h := httpapi.NewNoteHandler(svc, &fakeUserLookup{})

	req := httptest.NewRequest(http.MethodGet, "/notes?entity_type=customer&entity_id="+entityID.String(), nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Notes []struct {
			ID string `json:"id"`
		} `json:"notes"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Notes) != 1 {
		t.Fatalf("len(notes) = %d, want 1", len(body.Notes))
	}
}
