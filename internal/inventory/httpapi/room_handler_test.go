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

// fakeRoomService is the seam httpapi.RoomHandler depends on. See
// fakeSiteService's doc comment in site_handler_test.go for why this
// exists.
type fakeRoomService struct {
	rooms map[uuid.UUID]inventory.Room
	err   error
}

func newFakeRoomService(rooms ...inventory.Room) *fakeRoomService {
	f := &fakeRoomService{rooms: make(map[uuid.UUID]inventory.Room)}
	for _, r := range rooms {
		f.rooms[r.ID] = r
	}
	return f
}

func (f *fakeRoomService) Get(_ context.Context, id uuid.UUID) (inventory.Room, error) {
	if f.err != nil {
		return inventory.Room{}, f.err
	}
	r, ok := f.rooms[id]
	if !ok {
		return inventory.Room{}, apperror.NotFound("room not found")
	}
	return r, nil
}

func (f *fakeRoomService) List(context.Context) ([]inventory.Room, error) {
	if f.err != nil {
		return nil, f.err
	}
	rooms := make([]inventory.Room, 0, len(f.rooms))
	for _, r := range f.rooms {
		rooms = append(rooms, r)
	}
	return rooms, nil
}

func (f *fakeRoomService) Create(_ context.Context, room inventory.Room) (inventory.Room, error) {
	if f.err != nil {
		return inventory.Room{}, f.err
	}
	room.ID = uuid.New()
	room.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	room.UpdatedAt = room.CreatedAt
	f.rooms[room.ID] = room
	return room, nil
}

func (f *fakeRoomService) Update(_ context.Context, room inventory.Room) (inventory.Room, error) {
	if f.err != nil {
		return inventory.Room{}, f.err
	}
	if _, ok := f.rooms[room.ID]; !ok {
		return inventory.Room{}, apperror.NotFound("room not found")
	}
	f.rooms[room.ID] = room
	return room, nil
}

func (f *fakeRoomService) Delete(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.rooms[id]; !ok {
		return apperror.NotFound("room not found")
	}
	delete(f.rooms, id)
	return nil
}

// newRoomTestRouter mounts a RoomHandler backed by svc on a real
// chi.Router. See newTestRouter's doc comment in site_handler_test.go for
// why.
func newRoomTestRouter(svc *fakeRoomService) http.Handler {
	handler := httpapi.NewRoomHandler(svc)

	r := chi.NewRouter()
	r.Post("/rooms", handler.Create)
	r.Get("/rooms", handler.List)
	r.Get("/rooms/{id}", handler.Get)
	r.Put("/rooms/{id}", handler.Update)
	r.Delete("/rooms/{id}", handler.Delete)
	return r
}

func TestRoomHandlerCreate(t *testing.T) {
	router := newRoomTestRouter(newFakeRoomService())
	buildingID := uuid.New()

	req := httptest.NewRequest(http.MethodPost, "/rooms", strings.NewReader(
		`{"name":"Server Room","description":"Rack row A","building_id":"`+buildingID.String()+`"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		BuildingID string `json:"building_id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID == "" {
		t.Error("response did not include an id")
	}
	if body.Name != "Server Room" || body.BuildingID != buildingID.String() {
		t.Errorf("body = %+v, want Name=Server Room BuildingID=%s", body, buildingID)
	}
}

func TestRoomHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newRoomTestRouter(newFakeRoomService())

	req := httptest.NewRequest(http.MethodPost, "/rooms", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRoomHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := newFakeRoomService()
	svc.err = apperror.Invalid("name: is required")
	router := newRoomTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/rooms", strings.NewReader(`{"name":""}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestRoomHandlerList(t *testing.T) {
	a := inventory.Room{Metadata: inventory.Metadata{ID: uuid.New(), Name: "A"}, BuildingID: uuid.New()}
	b := inventory.Room{Metadata: inventory.Metadata{ID: uuid.New(), Name: "B"}, BuildingID: uuid.New()}
	router := newRoomTestRouter(newFakeRoomService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/rooms", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		Rooms []struct {
			ID string `json:"id"`
		} `json:"rooms"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Rooms) != 2 {
		t.Fatalf("len(rooms) = %d, want 2", len(body.Rooms))
	}
}

func TestRoomHandlerGet(t *testing.T) {
	room := inventory.Room{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Server Room"}, BuildingID: uuid.New()}
	router := newRoomTestRouter(newFakeRoomService(room))

	req := httptest.NewRequest(http.MethodGet, "/rooms/"+room.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestRoomHandlerGetNotFound(t *testing.T) {
	router := newRoomTestRouter(newFakeRoomService())

	req := httptest.NewRequest(http.MethodGet, "/rooms/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestRoomHandlerGetRejectsMalformedID(t *testing.T) {
	router := newRoomTestRouter(newFakeRoomService())

	req := httptest.NewRequest(http.MethodGet, "/rooms/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRoomHandlerUpdate(t *testing.T) {
	buildingID := uuid.New()
	room := inventory.Room{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Old Name"}, BuildingID: buildingID}
	router := newRoomTestRouter(newFakeRoomService(room))

	req := httptest.NewRequest(http.MethodPut, "/rooms/"+room.ID.String(), strings.NewReader(
		`{"name":"New Name","building_id":"`+buildingID.String()+`"}`))
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

func TestRoomHandlerUpdateNotFound(t *testing.T) {
	router := newRoomTestRouter(newFakeRoomService())

	req := httptest.NewRequest(http.MethodPut, "/rooms/"+uuid.New().String(), strings.NewReader(
		`{"name":"New Name","building_id":"`+uuid.New().String()+`"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestRoomHandlerDelete(t *testing.T) {
	room := inventory.Room{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Temporary"}, BuildingID: uuid.New()}
	router := newRoomTestRouter(newFakeRoomService(room))

	req := httptest.NewRequest(http.MethodDelete, "/rooms/"+room.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204 No Content", rec.Body.String())
	}
}

func TestRoomHandlerDeleteNotFound(t *testing.T) {
	router := newRoomTestRouter(newFakeRoomService())

	req := httptest.NewRequest(http.MethodDelete, "/rooms/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
