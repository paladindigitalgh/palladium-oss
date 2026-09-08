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

	"github.com/paladindigitalgh/palladium-oss/internal/oltmodel"
	"github.com/paladindigitalgh/palladium-oss/internal/oltmodel/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeOLTModelService is the seam httpapi.OLTModelHandler depends on
// (see its unexported oltModelService interface in
// olt_model_handler.go). It lets these tests exercise HTTP-only
// concerns — status codes, JSON shapes, routing, error translation —
// without a real service, repository, or database;
// internal/oltmodel/service and internal/oltmodel/postgres each have
// their own tests for the layers below this one.
type fakeOLTModelService struct {
	models map[uuid.UUID]oltmodel.OLTModel
	err    error // if set, every method returns this error instead
}

func newFakeOLTModelService(models ...oltmodel.OLTModel) *fakeOLTModelService {
	f := &fakeOLTModelService{models: make(map[uuid.UUID]oltmodel.OLTModel)}
	for _, m := range models {
		f.models[m.ID] = m
	}
	return f
}

func (f *fakeOLTModelService) Get(_ context.Context, id uuid.UUID) (oltmodel.OLTModel, error) {
	if f.err != nil {
		return oltmodel.OLTModel{}, f.err
	}
	m, ok := f.models[id]
	if !ok {
		return oltmodel.OLTModel{}, apperror.NotFound("olt model not found")
	}
	return m, nil
}

func (f *fakeOLTModelService) List(context.Context) ([]oltmodel.OLTModel, error) {
	if f.err != nil {
		return nil, f.err
	}
	models := make([]oltmodel.OLTModel, 0, len(f.models))
	for _, m := range f.models {
		models = append(models, m)
	}
	return models, nil
}

func (f *fakeOLTModelService) Create(_ context.Context, m oltmodel.OLTModel) (oltmodel.OLTModel, error) {
	if f.err != nil {
		return oltmodel.OLTModel{}, f.err
	}
	m.ID = uuid.New()
	m.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	m.UpdatedAt = m.CreatedAt
	f.models[m.ID] = m
	return m, nil
}

func (f *fakeOLTModelService) Update(_ context.Context, m oltmodel.OLTModel) (oltmodel.OLTModel, error) {
	if f.err != nil {
		return oltmodel.OLTModel{}, f.err
	}
	if _, ok := f.models[m.ID]; !ok {
		return oltmodel.OLTModel{}, apperror.NotFound("olt model not found")
	}
	f.models[m.ID] = m
	return m, nil
}

func (f *fakeOLTModelService) Delete(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.models[id]; !ok {
		return apperror.NotFound("olt model not found")
	}
	delete(f.models, id)
	return nil
}

// newTestRouter mounts an OLTModelHandler backed by svc on a real
// chi.Router, so tests that need a URL path parameter (Get/Update/
// Delete's {id}) get one populated the same way production code does,
// rather than faking chi's route context by hand.
func newTestRouter(svc *fakeOLTModelService) http.Handler {
	handler := httpapi.NewOLTModelHandler(svc)

	r := chi.NewRouter()
	r.Post("/olt-models", handler.Create)
	r.Get("/olt-models", handler.List)
	r.Get("/olt-models/{id}", handler.Get)
	r.Put("/olt-models/{id}", handler.Update)
	r.Delete("/olt-models/{id}", handler.Delete)
	return r
}

const validBody = `{"vendor":"Kontron","name":"C16","pon_port_count":16}`

func TestOLTModelHandlerCreate(t *testing.T) {
	router := newTestRouter(newFakeOLTModelService())

	req := httptest.NewRequest(http.MethodPost, "/olt-models", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		ID           string `json:"id"`
		Vendor       string `json:"vendor"`
		Name         string `json:"name"`
		PONPortCount int    `json:"pon_port_count"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID == "" {
		t.Error("response did not include an id")
	}
	if body.Vendor != "Kontron" {
		t.Errorf("vendor = %q, want %q", body.Vendor, "Kontron")
	}
	if body.Name != "C16" {
		t.Errorf("name = %q, want %q", body.Name, "C16")
	}
	if body.PONPortCount != 16 {
		t.Errorf("pon_port_count = %d, want %d", body.PONPortCount, 16)
	}
}

func TestOLTModelHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newTestRouter(newFakeOLTModelService())

	req := httptest.NewRequest(http.MethodPost, "/olt-models", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestOLTModelHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := newFakeOLTModelService()
	svc.err = apperror.Invalid("name: is required")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/olt-models", strings.NewReader(`{"name":""}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestOLTModelHandlerList(t *testing.T) {
	a := oltmodel.OLTModel{ID: uuid.New(), Vendor: oltmodel.VendorKontron, Name: "C8", PONPortCount: 8}
	b := oltmodel.OLTModel{ID: uuid.New(), Vendor: oltmodel.VendorKontron, Name: "C16", PONPortCount: 16}
	router := newTestRouter(newFakeOLTModelService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/olt-models", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		OLTModels []struct {
			ID string `json:"id"`
		} `json:"olt_models"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.OLTModels) != 2 {
		t.Fatalf("len(olt_models) = %d, want 2", len(body.OLTModels))
	}
}

func TestOLTModelHandlerGet(t *testing.T) {
	m := oltmodel.OLTModel{ID: uuid.New(), Vendor: oltmodel.VendorKontron, Name: "C16", PONPortCount: 16}
	router := newTestRouter(newFakeOLTModelService(m))

	req := httptest.NewRequest(http.MethodGet, "/olt-models/"+m.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestOLTModelHandlerGetNotFound(t *testing.T) {
	router := newTestRouter(newFakeOLTModelService())

	req := httptest.NewRequest(http.MethodGet, "/olt-models/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestOLTModelHandlerGetRejectsMalformedID(t *testing.T) {
	router := newTestRouter(newFakeOLTModelService())

	req := httptest.NewRequest(http.MethodGet, "/olt-models/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestOLTModelHandlerUpdate(t *testing.T) {
	m := oltmodel.OLTModel{ID: uuid.New(), Vendor: oltmodel.VendorKontron, Name: "C16", PONPortCount: 16}
	router := newTestRouter(newFakeOLTModelService(m))

	req := httptest.NewRequest(http.MethodPut, "/olt-models/"+m.ID.String(),
		strings.NewReader(`{"vendor":"Kontron","name":"C32","pon_port_count":32}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Name         string `json:"name"`
		PONPortCount int    `json:"pon_port_count"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Name != "C32" {
		t.Errorf("name = %q, want %q", body.Name, "C32")
	}
	if body.PONPortCount != 32 {
		t.Errorf("pon_port_count = %d, want %d", body.PONPortCount, 32)
	}
}

func TestOLTModelHandlerUpdateNotFound(t *testing.T) {
	router := newTestRouter(newFakeOLTModelService())

	req := httptest.NewRequest(http.MethodPut, "/olt-models/"+uuid.New().String(), strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestOLTModelHandlerDelete(t *testing.T) {
	m := oltmodel.OLTModel{ID: uuid.New(), Vendor: oltmodel.VendorKontron, Name: "C16", PONPortCount: 16}
	router := newTestRouter(newFakeOLTModelService(m))

	req := httptest.NewRequest(http.MethodDelete, "/olt-models/"+m.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204 No Content", rec.Body.String())
	}
}

func TestOLTModelHandlerDeleteNotFound(t *testing.T) {
	router := newTestRouter(newFakeOLTModelService())

	req := httptest.NewRequest(http.MethodDelete, "/olt-models/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
