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

	"github.com/paladindigitalgh/palladium-oss/internal/accessattachment"
	"github.com/paladindigitalgh/palladium-oss/internal/accessattachment/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeAccessAttachmentService is the seam
// httpapi.AccessAttachmentHandler depends on (see its unexported
// accessAttachmentService interface in access_attachment_handler.go). It
// lets these tests exercise HTTP-only concerns — status codes, JSON
// shapes, routing, error translation — without a real service,
// repository, or database; internal/accessattachment/service and
// internal/accessattachment/postgres each have their own tests for the
// layers below this one, including the active-attachment-uniqueness
// rule, which this fake deliberately does not reimplement — that
// business rule belongs to the service layer, not this HTTP-only test
// double.
type fakeAccessAttachmentService struct {
	attachments map[uuid.UUID]accessattachment.AccessAttachment
	err         error // if set, every method returns this error instead
}

func newFakeAccessAttachmentService(attachments ...accessattachment.AccessAttachment) *fakeAccessAttachmentService {
	f := &fakeAccessAttachmentService{attachments: make(map[uuid.UUID]accessattachment.AccessAttachment)}
	for _, a := range attachments {
		f.attachments[a.ID] = a
	}
	return f
}

func (f *fakeAccessAttachmentService) Get(_ context.Context, id uuid.UUID) (accessattachment.AccessAttachment, error) {
	if f.err != nil {
		return accessattachment.AccessAttachment{}, f.err
	}
	a, ok := f.attachments[id]
	if !ok {
		return accessattachment.AccessAttachment{}, apperror.NotFound("access attachment not found")
	}
	return a, nil
}

func (f *fakeAccessAttachmentService) List(context.Context) ([]accessattachment.AccessAttachment, error) {
	if f.err != nil {
		return nil, f.err
	}
	attachments := make([]accessattachment.AccessAttachment, 0, len(f.attachments))
	for _, a := range f.attachments {
		attachments = append(attachments, a)
	}
	return attachments, nil
}

func (f *fakeAccessAttachmentService) Create(_ context.Context, a accessattachment.AccessAttachment) (accessattachment.AccessAttachment, error) {
	if f.err != nil {
		return accessattachment.AccessAttachment{}, f.err
	}
	a.ID = uuid.New()
	a.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a.UpdatedAt = a.CreatedAt
	f.attachments[a.ID] = a
	return a, nil
}

func (f *fakeAccessAttachmentService) Update(_ context.Context, a accessattachment.AccessAttachment) (accessattachment.AccessAttachment, error) {
	if f.err != nil {
		return accessattachment.AccessAttachment{}, f.err
	}
	if _, ok := f.attachments[a.ID]; !ok {
		return accessattachment.AccessAttachment{}, apperror.NotFound("access attachment not found")
	}
	f.attachments[a.ID] = a
	return a, nil
}

func (f *fakeAccessAttachmentService) Delete(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.attachments[id]; !ok {
		return apperror.NotFound("access attachment not found")
	}
	delete(f.attachments, id)
	return nil
}

// newTestRouter mounts an AccessAttachmentHandler backed by svc on a
// real chi.Router, so tests that need a URL path parameter (Get/Update/
// Delete's {id}) get one populated the same way production code does,
// rather than faking chi's route context by hand.
func newTestRouter(svc *fakeAccessAttachmentService) http.Handler {
	handler := httpapi.NewAccessAttachmentHandler(svc)

	r := chi.NewRouter()
	r.Post("/access-attachments", handler.Create)
	r.Get("/access-attachments", handler.List)
	r.Get("/access-attachments/{id}", handler.Get)
	r.Put("/access-attachments/{id}", handler.Update)
	r.Delete("/access-attachments/{id}", handler.Delete)
	return r
}

const validBody = `{"access_interface_id":"11111111-1111-1111-1111-111111111111","service_equipment_id":"22222222-2222-2222-2222-222222222222"}`

func TestAccessAttachmentHandlerCreate(t *testing.T) {
	router := newTestRouter(newFakeAccessAttachmentService())

	req := httptest.NewRequest(http.MethodPost, "/access-attachments", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		ID                 string `json:"id"`
		AccessInterfaceID  string `json:"access_interface_id"`
		ServiceEquipmentID string `json:"service_equipment_id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID == "" {
		t.Error("response did not include an id")
	}
	if body.AccessInterfaceID != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("access_interface_id = %q, want %q", body.AccessInterfaceID, "11111111-1111-1111-1111-111111111111")
	}
	if body.ServiceEquipmentID != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("service_equipment_id = %q, want %q", body.ServiceEquipmentID, "22222222-2222-2222-2222-222222222222")
	}
}

func TestAccessAttachmentHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newTestRouter(newFakeAccessAttachmentService())

	req := httptest.NewRequest(http.MethodPost, "/access-attachments", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestAccessAttachmentHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := newFakeAccessAttachmentService()
	svc.err = apperror.Invalid("service_equipment_id: is required")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/access-attachments", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestAccessAttachmentHandlerCreatePropagatesConflictOnUnknownAccessInterface(t *testing.T) {
	svc := newFakeAccessAttachmentService()
	svc.err = apperror.Conflict("create access attachment: violates a foreign key relationship")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/access-attachments", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

// TestAccessAttachmentHandlerCreatePropagatesConflictOnActiveAttachmentUniquenessViolation
// proves the handler passes through AccessAttachmentService's
// active-attachment-uniqueness Conflict error (this milestone's goal 2)
// exactly like any other Conflict — the handler itself has no
// awareness of that rule, only of the error it produces.
func TestAccessAttachmentHandlerCreatePropagatesConflictOnActiveAttachmentUniquenessViolation(t *testing.T) {
	svc := newFakeAccessAttachmentService()
	svc.err = apperror.Conflict("service equipment already has an active access attachment")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/access-attachments", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestAccessAttachmentHandlerList(t *testing.T) {
	a := accessattachment.AccessAttachment{ID: uuid.New(), AccessInterfaceID: uuid.New(), ServiceEquipmentID: uuid.New()}
	b := accessattachment.AccessAttachment{ID: uuid.New(), AccessInterfaceID: uuid.New(), ServiceEquipmentID: uuid.New()}
	router := newTestRouter(newFakeAccessAttachmentService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/access-attachments", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		AccessAttachments []struct {
			ID string `json:"id"`
		} `json:"access_attachments"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.AccessAttachments) != 2 {
		t.Fatalf("len(access_attachments) = %d, want 2", len(body.AccessAttachments))
	}
}

func TestAccessAttachmentHandlerGet(t *testing.T) {
	a := accessattachment.AccessAttachment{ID: uuid.New(), AccessInterfaceID: uuid.New(), ServiceEquipmentID: uuid.New()}
	router := newTestRouter(newFakeAccessAttachmentService(a))

	req := httptest.NewRequest(http.MethodGet, "/access-attachments/"+a.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestAccessAttachmentHandlerGetNotFound(t *testing.T) {
	router := newTestRouter(newFakeAccessAttachmentService())

	req := httptest.NewRequest(http.MethodGet, "/access-attachments/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestAccessAttachmentHandlerGetRejectsMalformedID(t *testing.T) {
	router := newTestRouter(newFakeAccessAttachmentService())

	req := httptest.NewRequest(http.MethodGet, "/access-attachments/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestAccessAttachmentHandlerUpdate(t *testing.T) {
	a := accessattachment.AccessAttachment{ID: uuid.New(), AccessInterfaceID: uuid.New(), ServiceEquipmentID: uuid.New()}
	router := newTestRouter(newFakeAccessAttachmentService(a))

	req := httptest.NewRequest(http.MethodPut, "/access-attachments/"+a.ID.String(),
		strings.NewReader(`{"access_interface_id":"`+a.AccessInterfaceID.String()+`","service_equipment_id":"`+a.ServiceEquipmentID.String()+`","removal_reason":"Customer moved"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		RemovalReason string `json:"removal_reason"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.RemovalReason != "Customer moved" {
		t.Errorf("removal_reason = %q, want %q", body.RemovalReason, "Customer moved")
	}
}

func TestAccessAttachmentHandlerUpdateNotFound(t *testing.T) {
	router := newTestRouter(newFakeAccessAttachmentService())

	req := httptest.NewRequest(http.MethodPut, "/access-attachments/"+uuid.New().String(), strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestAccessAttachmentHandlerDelete(t *testing.T) {
	a := accessattachment.AccessAttachment{ID: uuid.New(), AccessInterfaceID: uuid.New(), ServiceEquipmentID: uuid.New()}
	router := newTestRouter(newFakeAccessAttachmentService(a))

	req := httptest.NewRequest(http.MethodDelete, "/access-attachments/"+a.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204 No Content", rec.Body.String())
	}
}

func TestAccessAttachmentHandlerDeleteNotFound(t *testing.T) {
	router := newTestRouter(newFakeAccessAttachmentService())

	req := httptest.NewRequest(http.MethodDelete, "/access-attachments/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
