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

// fakeBuildingService is the seam httpapi.BuildingHandler depends on. See
// fakeSiteService's doc comment in site_handler_test.go for why this
// exists.
type fakeBuildingService struct {
	buildings map[uuid.UUID]inventory.Building
	err       error
}

func newFakeBuildingService(buildings ...inventory.Building) *fakeBuildingService {
	f := &fakeBuildingService{buildings: make(map[uuid.UUID]inventory.Building)}
	for _, b := range buildings {
		f.buildings[b.ID] = b
	}
	return f
}

func (f *fakeBuildingService) Get(_ context.Context, id uuid.UUID) (inventory.Building, error) {
	if f.err != nil {
		return inventory.Building{}, f.err
	}
	b, ok := f.buildings[id]
	if !ok {
		return inventory.Building{}, apperror.NotFound("building not found")
	}
	return b, nil
}

func (f *fakeBuildingService) List(context.Context) ([]inventory.Building, error) {
	if f.err != nil {
		return nil, f.err
	}
	buildings := make([]inventory.Building, 0, len(f.buildings))
	for _, b := range f.buildings {
		buildings = append(buildings, b)
	}
	return buildings, nil
}

func (f *fakeBuildingService) Create(_ context.Context, building inventory.Building) (inventory.Building, error) {
	if f.err != nil {
		return inventory.Building{}, f.err
	}
	building.ID = uuid.New()
	building.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	building.UpdatedAt = building.CreatedAt
	f.buildings[building.ID] = building
	return building, nil
}

func (f *fakeBuildingService) Update(_ context.Context, building inventory.Building) (inventory.Building, error) {
	if f.err != nil {
		return inventory.Building{}, f.err
	}
	if _, ok := f.buildings[building.ID]; !ok {
		return inventory.Building{}, apperror.NotFound("building not found")
	}
	f.buildings[building.ID] = building
	return building, nil
}

func (f *fakeBuildingService) Delete(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.buildings[id]; !ok {
		return apperror.NotFound("building not found")
	}
	delete(f.buildings, id)
	return nil
}

// newBuildingTestRouter mounts a BuildingHandler backed by svc on a real
// chi.Router. See newTestRouter's doc comment in site_handler_test.go for
// why.
func newBuildingTestRouter(svc *fakeBuildingService) http.Handler {
	handler := httpapi.NewBuildingHandler(svc)

	r := chi.NewRouter()
	r.Post("/buildings", handler.Create)
	r.Get("/buildings", handler.List)
	r.Get("/buildings/{id}", handler.Get)
	r.Put("/buildings/{id}", handler.Update)
	r.Delete("/buildings/{id}", handler.Delete)
	return r
}

func TestBuildingHandlerCreate(t *testing.T) {
	router := newBuildingTestRouter(newFakeBuildingService())
	siteID := uuid.New()

	req := httptest.NewRequest(http.MethodPost, "/buildings", strings.NewReader(
		`{"name":"Main Office","description":"HQ","site_id":"`+siteID.String()+`"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		SiteID string `json:"site_id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID == "" {
		t.Error("response did not include an id")
	}
	if body.Name != "Main Office" || body.SiteID != siteID.String() {
		t.Errorf("body = %+v, want Name=Main Office SiteID=%s", body, siteID)
	}
}

func TestBuildingHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newBuildingTestRouter(newFakeBuildingService())

	req := httptest.NewRequest(http.MethodPost, "/buildings", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestBuildingHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := newFakeBuildingService()
	svc.err = apperror.Invalid("name: is required")
	router := newBuildingTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/buildings", strings.NewReader(`{"name":""}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestBuildingHandlerList(t *testing.T) {
	a := inventory.Building{Metadata: inventory.Metadata{ID: uuid.New(), Name: "A"}, SiteID: uuid.New()}
	b := inventory.Building{Metadata: inventory.Metadata{ID: uuid.New(), Name: "B"}, SiteID: uuid.New()}
	router := newBuildingTestRouter(newFakeBuildingService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/buildings", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		Buildings []struct {
			ID string `json:"id"`
		} `json:"buildings"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Buildings) != 2 {
		t.Fatalf("len(buildings) = %d, want 2", len(body.Buildings))
	}
}

func TestBuildingHandlerGet(t *testing.T) {
	building := inventory.Building{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Main Office"}, SiteID: uuid.New()}
	router := newBuildingTestRouter(newFakeBuildingService(building))

	req := httptest.NewRequest(http.MethodGet, "/buildings/"+building.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestBuildingHandlerGetNotFound(t *testing.T) {
	router := newBuildingTestRouter(newFakeBuildingService())

	req := httptest.NewRequest(http.MethodGet, "/buildings/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestBuildingHandlerGetRejectsMalformedID(t *testing.T) {
	router := newBuildingTestRouter(newFakeBuildingService())

	req := httptest.NewRequest(http.MethodGet, "/buildings/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestBuildingHandlerUpdate(t *testing.T) {
	siteID := uuid.New()
	building := inventory.Building{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Old Name"}, SiteID: siteID}
	router := newBuildingTestRouter(newFakeBuildingService(building))

	req := httptest.NewRequest(http.MethodPut, "/buildings/"+building.ID.String(), strings.NewReader(
		`{"name":"New Name","site_id":"`+siteID.String()+`"}`))
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

func TestBuildingHandlerUpdateNotFound(t *testing.T) {
	router := newBuildingTestRouter(newFakeBuildingService())

	req := httptest.NewRequest(http.MethodPut, "/buildings/"+uuid.New().String(), strings.NewReader(
		`{"name":"New Name","site_id":"`+uuid.New().String()+`"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestBuildingHandlerDelete(t *testing.T) {
	building := inventory.Building{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Temporary"}, SiteID: uuid.New()}
	router := newBuildingTestRouter(newFakeBuildingService(building))

	req := httptest.NewRequest(http.MethodDelete, "/buildings/"+building.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204 No Content", rec.Body.String())
	}
}

func TestBuildingHandlerDeleteNotFound(t *testing.T) {
	router := newBuildingTestRouter(newFakeBuildingService())

	req := httptest.NewRequest(http.MethodDelete, "/buildings/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
