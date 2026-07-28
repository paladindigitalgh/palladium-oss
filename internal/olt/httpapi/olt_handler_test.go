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

	"github.com/paladindigitalgh/palladium-oss/internal/olt"
	"github.com/paladindigitalgh/palladium-oss/internal/olt/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeOLTService is the seam httpapi.OLTHandler depends on (see its
// unexported oltService interface in olt_handler.go). It lets these
// tests exercise HTTP-only concerns — status codes, JSON shapes,
// routing, error translation — without a real service, repository, or
// database; internal/olt/service and internal/olt/postgres each have
// their own tests for the layers below this one.
type fakeOLTService struct {
	olts map[uuid.UUID]olt.OLT
	err  error // if set, every method returns this error instead
}

func newFakeOLTService(olts ...olt.OLT) *fakeOLTService {
	f := &fakeOLTService{olts: make(map[uuid.UUID]olt.OLT)}
	for _, o := range olts {
		f.olts[o.ID] = o
	}
	return f
}

func (f *fakeOLTService) Get(_ context.Context, id uuid.UUID) (olt.OLT, error) {
	if f.err != nil {
		return olt.OLT{}, f.err
	}
	o, ok := f.olts[id]
	if !ok {
		return olt.OLT{}, apperror.NotFound("olt not found")
	}
	return o, nil
}

func (f *fakeOLTService) List(context.Context) ([]olt.OLT, error) {
	if f.err != nil {
		return nil, f.err
	}
	olts := make([]olt.OLT, 0, len(f.olts))
	for _, o := range f.olts {
		olts = append(olts, o)
	}
	return olts, nil
}

func (f *fakeOLTService) Create(_ context.Context, o olt.OLT) (olt.OLT, error) {
	if f.err != nil {
		return olt.OLT{}, f.err
	}
	o.ID = uuid.New()
	o.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	o.UpdatedAt = o.CreatedAt
	f.olts[o.ID] = o
	return o, nil
}

func (f *fakeOLTService) Update(_ context.Context, o olt.OLT) (olt.OLT, error) {
	if f.err != nil {
		return olt.OLT{}, f.err
	}
	if _, ok := f.olts[o.ID]; !ok {
		return olt.OLT{}, apperror.NotFound("olt not found")
	}
	f.olts[o.ID] = o
	return o, nil
}

func (f *fakeOLTService) Delete(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.olts[id]; !ok {
		return apperror.NotFound("olt not found")
	}
	delete(f.olts, id)
	return nil
}

// newTestRouter mounts an OLTHandler backed by svc on a real chi.Router,
// so tests that need a URL path parameter (Get/Update/Delete's {id}) get
// one populated the same way production code does, rather than faking
// chi's route context by hand.
func newTestRouter(svc *fakeOLTService) http.Handler {
	handler := httpapi.NewOLTHandler(svc)

	r := chi.NewRouter()
	r.Post("/olts", handler.Create)
	r.Get("/olts", handler.List)
	r.Get("/olts/{id}", handler.Get)
	r.Put("/olts/{id}", handler.Update)
	r.Delete("/olts/{id}", handler.Delete)
	return r
}

const validBody = `{"access_network_id":"11111111-1111-1111-1111-111111111111","name":"OLT-01","vendor":"Kontron"}`

func TestOLTHandlerCreate(t *testing.T) {
	router := newTestRouter(newFakeOLTService())

	req := httptest.NewRequest(http.MethodPost, "/olts", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		ID              string `json:"id"`
		AccessNetworkID string `json:"access_network_id"`
		Name            string `json:"name"`
		Vendor          string `json:"vendor"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID == "" {
		t.Error("response did not include an id")
	}
	if body.AccessNetworkID != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("access_network_id = %q, want %q", body.AccessNetworkID, "11111111-1111-1111-1111-111111111111")
	}
	if body.Name != "OLT-01" || body.Vendor != "Kontron" {
		t.Errorf("body = %+v, want Name=OLT-01 Vendor=Kontron", body)
	}
}

func TestOLTHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newTestRouter(newFakeOLTService())

	req := httptest.NewRequest(http.MethodPost, "/olts", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestOLTHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := newFakeOLTService()
	svc.err = apperror.Invalid("name: is required")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/olts", strings.NewReader(`{"name":""}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestOLTHandlerCreatePropagatesConflictOnUnknownAccessNetwork(t *testing.T) {
	svc := newFakeOLTService()
	svc.err = apperror.Conflict("create olt: violates a foreign key relationship")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/olts", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestOLTHandlerList(t *testing.T) {
	a := olt.OLT{ID: uuid.New(), AccessNetworkID: uuid.New(), Name: "A", Vendor: olt.VendorKontron}
	b := olt.OLT{ID: uuid.New(), AccessNetworkID: uuid.New(), Name: "B", Vendor: olt.VendorNokia}
	router := newTestRouter(newFakeOLTService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/olts", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		OLTs []struct {
			ID string `json:"id"`
		} `json:"olts"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.OLTs) != 2 {
		t.Fatalf("len(olts) = %d, want 2", len(body.OLTs))
	}
}

func TestOLTHandlerGet(t *testing.T) {
	o := olt.OLT{ID: uuid.New(), AccessNetworkID: uuid.New(), Name: "OLT-01", Vendor: olt.VendorKontron}
	router := newTestRouter(newFakeOLTService(o))

	req := httptest.NewRequest(http.MethodGet, "/olts/"+o.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestOLTHandlerGetNotFound(t *testing.T) {
	router := newTestRouter(newFakeOLTService())

	req := httptest.NewRequest(http.MethodGet, "/olts/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestOLTHandlerGetRejectsMalformedID(t *testing.T) {
	router := newTestRouter(newFakeOLTService())

	req := httptest.NewRequest(http.MethodGet, "/olts/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestOLTHandlerUpdate(t *testing.T) {
	o := olt.OLT{ID: uuid.New(), AccessNetworkID: uuid.New(), Name: "Old Name", Vendor: olt.VendorKontron}
	router := newTestRouter(newFakeOLTService(o))

	req := httptest.NewRequest(http.MethodPut, "/olts/"+o.ID.String(),
		strings.NewReader(`{"access_network_id":"`+o.AccessNetworkID.String()+`","name":"New Name","vendor":"Nokia"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Name   string `json:"name"`
		Vendor string `json:"vendor"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Name != "New Name" || body.Vendor != "Nokia" {
		t.Errorf("body = %+v, want Name=New Name Vendor=Nokia", body)
	}
}

func TestOLTHandlerUpdateNotFound(t *testing.T) {
	router := newTestRouter(newFakeOLTService())

	req := httptest.NewRequest(http.MethodPut, "/olts/"+uuid.New().String(), strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestOLTHandlerDelete(t *testing.T) {
	o := olt.OLT{ID: uuid.New(), AccessNetworkID: uuid.New(), Name: "Temporary", Vendor: olt.VendorKontron}
	router := newTestRouter(newFakeOLTService(o))

	req := httptest.NewRequest(http.MethodDelete, "/olts/"+o.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204 No Content", rec.Body.String())
	}
}

func TestOLTHandlerDeleteNotFound(t *testing.T) {
	router := newTestRouter(newFakeOLTService())

	req := httptest.NewRequest(http.MethodDelete, "/olts/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
