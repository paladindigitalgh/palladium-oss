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

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeRackService is the seam httpapi.RackHandler depends on. See
// fakeSiteService's doc comment in site_handler_test.go for why this
// exists.
type fakeRackService struct {
	racks map[uuid.UUID]inventory.Rack
	err   error
}

func newFakeRackService(racks ...inventory.Rack) *fakeRackService {
	f := &fakeRackService{racks: make(map[uuid.UUID]inventory.Rack)}
	for _, r := range racks {
		f.racks[r.ID] = r
	}
	return f
}

func (f *fakeRackService) Get(_ context.Context, id uuid.UUID) (inventory.Rack, error) {
	if f.err != nil {
		return inventory.Rack{}, f.err
	}
	r, ok := f.racks[id]
	if !ok {
		return inventory.Rack{}, apperror.NotFound("rack not found")
	}
	return r, nil
}

func (f *fakeRackService) List(context.Context) ([]inventory.Rack, error) {
	if f.err != nil {
		return nil, f.err
	}
	racks := make([]inventory.Rack, 0, len(f.racks))
	for _, r := range f.racks {
		racks = append(racks, r)
	}
	return racks, nil
}

func (f *fakeRackService) Create(_ context.Context, rack inventory.Rack) (inventory.Rack, error) {
	if f.err != nil {
		return inventory.Rack{}, f.err
	}
	rack.ID = uuid.New()
	rack.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rack.UpdatedAt = rack.CreatedAt
	f.racks[rack.ID] = rack
	return rack, nil
}

func (f *fakeRackService) Update(_ context.Context, rack inventory.Rack) (inventory.Rack, error) {
	if f.err != nil {
		return inventory.Rack{}, f.err
	}
	if _, ok := f.racks[rack.ID]; !ok {
		return inventory.Rack{}, apperror.NotFound("rack not found")
	}
	f.racks[rack.ID] = rack
	return rack, nil
}

func (f *fakeRackService) Delete(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.racks[id]; !ok {
		return apperror.NotFound("rack not found")
	}
	delete(f.racks, id)
	return nil
}

// newRackTestRouter mounts a RackHandler backed by svc on a real
// chi.Router. See newTestRouter's doc comment in site_handler_test.go for
// why.
func newRackTestRouter(svc *fakeRackService) http.Handler {
	handler := httpapi.NewRackHandler(svc)

	r := chi.NewRouter()
	r.Post("/racks", handler.Create)
	r.Get("/racks", handler.List)
	r.Get("/racks/{id}", handler.Get)
	r.Put("/racks/{id}", handler.Update)
	r.Delete("/racks/{id}", handler.Delete)
	return r
}

func TestRackHandlerCreate(t *testing.T) {
	router := newRackTestRouter(newFakeRackService())

	req := httptest.NewRequest(http.MethodPost, "/racks", strings.NewReader(`{"name":"Rack A1","description":"Row A"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		ID     string  `json:"id"`
		Name   string  `json:"name"`
		RoomID *string `json:"room_id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID == "" {
		t.Error("response did not include an id")
	}
	if body.Name != "Rack A1" {
		t.Errorf("Name = %q, want %q", body.Name, "Rack A1")
	}
	if body.RoomID != nil {
		t.Errorf("RoomID = %v, want nil when not supplied", body.RoomID)
	}
}

func TestRackHandlerCreateWithRoomID(t *testing.T) {
	router := newRackTestRouter(newFakeRackService())
	roomID := uuid.New()

	req := httptest.NewRequest(http.MethodPost, "/racks", strings.NewReader(
		`{"name":"Rack A1","room_id":"`+roomID.String()+`"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		RoomID *string `json:"room_id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.RoomID == nil || *body.RoomID != roomID.String() {
		t.Errorf("RoomID = %v, want %s", body.RoomID, roomID)
	}
}

func TestRackHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newRackTestRouter(newFakeRackService())

	req := httptest.NewRequest(http.MethodPost, "/racks", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRackHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := newFakeRackService()
	svc.err = apperror.Invalid("name: is required")
	router := newRackTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/racks", strings.NewReader(`{"name":""}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestRackHandlerList(t *testing.T) {
	a := inventory.Rack{Metadata: inventory.Metadata{ID: uuid.New(), Name: "A"}}
	b := inventory.Rack{Metadata: inventory.Metadata{ID: uuid.New(), Name: "B"}}
	router := newRackTestRouter(newFakeRackService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/racks", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		Racks []struct {
			ID string `json:"id"`
		} `json:"racks"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Racks) != 2 {
		t.Fatalf("len(racks) = %d, want 2", len(body.Racks))
	}
}

func TestRackHandlerGet(t *testing.T) {
	rack := inventory.Rack{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Rack A1"}}
	router := newRackTestRouter(newFakeRackService(rack))

	req := httptest.NewRequest(http.MethodGet, "/racks/"+rack.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestRackHandlerGetNotFound(t *testing.T) {
	router := newRackTestRouter(newFakeRackService())

	req := httptest.NewRequest(http.MethodGet, "/racks/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestRackHandlerGetRejectsMalformedID(t *testing.T) {
	router := newRackTestRouter(newFakeRackService())

	req := httptest.NewRequest(http.MethodGet, "/racks/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRackHandlerUpdate(t *testing.T) {
	rack := inventory.Rack{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Old Name"}}
	router := newRackTestRouter(newFakeRackService(rack))

	req := httptest.NewRequest(http.MethodPut, "/racks/"+rack.ID.String(), strings.NewReader(`{"name":"New Name"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Name != "New Name" {
		t.Errorf("Name = %q, want %q", body.Name, "New Name")
	}
}

func TestRackHandlerUpdateNotFound(t *testing.T) {
	router := newRackTestRouter(newFakeRackService())

	req := httptest.NewRequest(http.MethodPut, "/racks/"+uuid.New().String(), strings.NewReader(`{"name":"New Name"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestRackHandlerDelete(t *testing.T) {
	rack := inventory.Rack{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Temporary"}}
	router := newRackTestRouter(newFakeRackService(rack))

	req := httptest.NewRequest(http.MethodDelete, "/racks/"+rack.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204 No Content", rec.Body.String())
	}
}

func TestRackHandlerDeleteNotFound(t *testing.T) {
	router := newRackTestRouter(newFakeRackService())

	req := httptest.NewRequest(http.MethodDelete, "/racks/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
