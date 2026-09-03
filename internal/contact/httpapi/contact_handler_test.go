package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/contact"
	"github.com/paladindigitalgh/palladium-oss/internal/contact/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeContactService is the seam httpapi.ContactHandler depends on (see
// its unexported contactService interface in contact_handler.go). It
// lets these tests exercise HTTP-only concerns — status codes, JSON
// shapes, routing, error translation — without a real service,
// repository, or database; internal/contact/service and
// internal/contact/postgres each have their own tests for the layers
// below this one.
type fakeContactService struct {
	contacts map[uuid.UUID]contact.Contact
	err      error // if set, every method returns this error instead
}

func newFakeContactService(contacts ...contact.Contact) *fakeContactService {
	f := &fakeContactService{contacts: make(map[uuid.UUID]contact.Contact)}
	for _, c := range contacts {
		f.contacts[c.ID] = c
	}
	return f
}

func (f *fakeContactService) Get(_ context.Context, id uuid.UUID) (contact.Contact, error) {
	if f.err != nil {
		return contact.Contact{}, f.err
	}
	c, ok := f.contacts[id]
	if !ok {
		return contact.Contact{}, apperror.NotFound("contact not found")
	}
	return c, nil
}

func (f *fakeContactService) List(context.Context) ([]contact.Contact, error) {
	if f.err != nil {
		return nil, f.err
	}
	contacts := make([]contact.Contact, 0, len(f.contacts))
	for _, c := range f.contacts {
		contacts = append(contacts, c)
	}
	return contacts, nil
}

func (f *fakeContactService) Create(_ context.Context, c contact.Contact) (contact.Contact, error) {
	if f.err != nil {
		return contact.Contact{}, f.err
	}
	c.ID = uuid.New()
	c.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c.UpdatedAt = c.CreatedAt
	f.contacts[c.ID] = c
	return c, nil
}

func (f *fakeContactService) Update(_ context.Context, c contact.Contact) (contact.Contact, error) {
	if f.err != nil {
		return contact.Contact{}, f.err
	}
	if _, ok := f.contacts[c.ID]; !ok {
		return contact.Contact{}, apperror.NotFound("contact not found")
	}
	f.contacts[c.ID] = c
	return c, nil
}

func (f *fakeContactService) Delete(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.contacts[id]; !ok {
		return apperror.NotFound("contact not found")
	}
	delete(f.contacts, id)
	return nil
}

// newTestRouter mounts a ContactHandler backed by svc on a real
// chi.Router, so tests that need a URL path parameter (Get/Update/
// Delete's {id}) get one populated the same way production code does,
// rather than faking chi's route context by hand.
func newTestRouter(svc *fakeContactService) http.Handler {
	handler := httpapi.NewContactHandler(svc)

	r := chi.NewRouter()
	r.Post("/contacts", handler.Create)
	r.Get("/contacts", handler.List)
	r.Get("/contacts/{id}", handler.Get)
	r.Put("/contacts/{id}", handler.Update)
	r.Delete("/contacts/{id}", handler.Delete)
	return r
}

const validBody = `{"customer_id":"11111111-1111-1111-1111-111111111111","name":"Jane Doe","role":"Primary","status":"Active"}`

func TestContactHandlerCreate(t *testing.T) {
	router := newTestRouter(newFakeContactService())

	req := httptest.NewRequest(http.MethodPost, "/contacts", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		ID         string `json:"id"`
		CustomerID string `json:"customer_id"`
		Name       string `json:"name"`
		Role       string `json:"role"`
		Status     string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID == "" {
		t.Error("response did not include an id")
	}
	if body.CustomerID != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("customer_id = %q, want %q", body.CustomerID, "11111111-1111-1111-1111-111111111111")
	}
	if body.Name != "Jane Doe" || body.Role != "Primary" || body.Status != "Active" {
		t.Errorf("body = %+v, want Name=Jane Doe Role=Primary Status=Active", body)
	}
}

func TestContactHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newTestRouter(newFakeContactService())

	req := httptest.NewRequest(http.MethodPost, "/contacts", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestContactHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := newFakeContactService()
	svc.err = apperror.Invalid("name: is required")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/contacts", strings.NewReader(`{"name":""}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestContactHandlerCreatePropagatesConflictOnUnknownCustomer(t *testing.T) {
	svc := newFakeContactService()
	svc.err = apperror.Conflict("create contact: violates a foreign key relationship")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/contacts", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestContactHandlerList(t *testing.T) {
	a := contact.Contact{ID: uuid.New(), CustomerID: uuid.New(), Name: "A", Role: contact.ContactRolePrimary, Status: contact.ContactStatusActive}
	b := contact.Contact{ID: uuid.New(), CustomerID: uuid.New(), Name: "B", Role: contact.ContactRoleBilling, Status: contact.ContactStatusActive}
	router := newTestRouter(newFakeContactService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/contacts", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		Contacts []struct {
			ID string `json:"id"`
		} `json:"contacts"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Contacts) != 2 {
		t.Fatalf("len(contacts) = %d, want 2", len(body.Contacts))
	}
}

func TestContactHandlerGet(t *testing.T) {
	c := contact.Contact{ID: uuid.New(), CustomerID: uuid.New(), Name: "Jane Doe", Role: contact.ContactRolePrimary, Status: contact.ContactStatusActive}
	router := newTestRouter(newFakeContactService(c))

	req := httptest.NewRequest(http.MethodGet, "/contacts/"+c.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestContactHandlerGetNotFound(t *testing.T) {
	router := newTestRouter(newFakeContactService())

	req := httptest.NewRequest(http.MethodGet, "/contacts/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestContactHandlerGetRejectsMalformedID(t *testing.T) {
	router := newTestRouter(newFakeContactService())

	req := httptest.NewRequest(http.MethodGet, "/contacts/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestContactHandlerUpdate(t *testing.T) {
	c := contact.Contact{ID: uuid.New(), CustomerID: uuid.New(), Name: "Old Name", Role: contact.ContactRolePrimary, Status: contact.ContactStatusActive}
	router := newTestRouter(newFakeContactService(c))

	req := httptest.NewRequest(http.MethodPut, "/contacts/"+c.ID.String(),
		strings.NewReader(`{"customer_id":"`+c.CustomerID.String()+`","name":"New Name","role":"Billing","status":"Inactive"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Name   string `json:"name"`
		Role   string `json:"role"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Name != "New Name" || body.Role != "Billing" || body.Status != "Inactive" {
		t.Errorf("body = %+v, want Name=New Name Role=Billing Status=Inactive", body)
	}
}

func TestContactHandlerUpdateNotFound(t *testing.T) {
	router := newTestRouter(newFakeContactService())

	req := httptest.NewRequest(http.MethodPut, "/contacts/"+uuid.New().String(), strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestContactHandlerDelete(t *testing.T) {
	c := contact.Contact{ID: uuid.New(), CustomerID: uuid.New(), Name: "Temporary", Role: contact.ContactRolePrimary, Status: contact.ContactStatusActive}
	router := newTestRouter(newFakeContactService(c))

	req := httptest.NewRequest(http.MethodDelete, "/contacts/"+c.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204 No Content", rec.Body.String())
	}
}

func TestContactHandlerDeleteNotFound(t *testing.T) {
	router := newTestRouter(newFakeContactService())

	req := httptest.NewRequest(http.MethodDelete, "/contacts/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
