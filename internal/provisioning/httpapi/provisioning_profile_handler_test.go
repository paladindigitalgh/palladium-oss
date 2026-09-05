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

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning/httpapi"
)

// fakeProvisioningProfileService is the seam
// httpapi.ProvisioningProfileHandler depends on (see its unexported
// provisioningProfileService interface in
// provisioning_profile_handler.go). It lets these tests exercise
// HTTP-only concerns — status codes, JSON shapes, routing, error
// translation — without a real service, repository, or database;
// internal/provisioning/service and internal/provisioning/postgres each
// have their own tests for the layers below this one.
type fakeProvisioningProfileService struct {
	profiles map[uuid.UUID]provisioning.ProvisioningProfile
	err      error // if set, every method returns this error instead
}

func newFakeProvisioningProfileService(profiles ...provisioning.ProvisioningProfile) *fakeProvisioningProfileService {
	f := &fakeProvisioningProfileService{profiles: make(map[uuid.UUID]provisioning.ProvisioningProfile)}
	for _, p := range profiles {
		f.profiles[p.ID] = p
	}
	return f
}

func (f *fakeProvisioningProfileService) Get(_ context.Context, id uuid.UUID) (provisioning.ProvisioningProfile, error) {
	if f.err != nil {
		return provisioning.ProvisioningProfile{}, f.err
	}
	p, ok := f.profiles[id]
	if !ok {
		return provisioning.ProvisioningProfile{}, apperror.NotFound("provisioning profile not found")
	}
	return p, nil
}

func (f *fakeProvisioningProfileService) List(context.Context) ([]provisioning.ProvisioningProfile, error) {
	if f.err != nil {
		return nil, f.err
	}
	profiles := make([]provisioning.ProvisioningProfile, 0, len(f.profiles))
	for _, p := range f.profiles {
		profiles = append(profiles, p)
	}
	return profiles, nil
}

func (f *fakeProvisioningProfileService) Create(_ context.Context, p provisioning.ProvisioningProfile) (provisioning.ProvisioningProfile, error) {
	if f.err != nil {
		return provisioning.ProvisioningProfile{}, f.err
	}
	p.ID = uuid.New()
	p.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p.UpdatedAt = p.CreatedAt
	f.profiles[p.ID] = p
	return p, nil
}

func (f *fakeProvisioningProfileService) Update(_ context.Context, p provisioning.ProvisioningProfile) (provisioning.ProvisioningProfile, error) {
	if f.err != nil {
		return provisioning.ProvisioningProfile{}, f.err
	}
	if _, ok := f.profiles[p.ID]; !ok {
		return provisioning.ProvisioningProfile{}, apperror.NotFound("provisioning profile not found")
	}
	f.profiles[p.ID] = p
	return p, nil
}

func (f *fakeProvisioningProfileService) Delete(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.profiles[id]; !ok {
		return apperror.NotFound("provisioning profile not found")
	}
	delete(f.profiles, id)
	return nil
}

// newTestRouter mounts a ProvisioningProfileHandler backed by svc on a
// real chi.Router, so tests that need a URL path parameter (Get/Update/
// Delete's {id}) get one populated the same way production code does,
// rather than faking chi's route context by hand.
func newTestRouter(svc *fakeProvisioningProfileService) http.Handler {
	handler := httpapi.NewProvisioningProfileHandler(svc)

	r := chi.NewRouter()
	r.Post("/provisioning-profiles", handler.Create)
	r.Get("/provisioning-profiles", handler.List)
	r.Get("/provisioning-profiles/{id}", handler.Get)
	r.Put("/provisioning-profiles/{id}", handler.Update)
	r.Delete("/provisioning-profiles/{id}", handler.Delete)
	return r
}

const validBody = `{"product_id":"11111111-1111-1111-1111-111111111111","vendor":"Kontron","profile_name":"RES-500M"}`

func TestProvisioningProfileHandlerCreate(t *testing.T) {
	router := newTestRouter(newFakeProvisioningProfileService())

	req := httptest.NewRequest(http.MethodPost, "/provisioning-profiles", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		ID          string `json:"id"`
		ProductID   string `json:"product_id"`
		Vendor      string `json:"vendor"`
		ProfileName string `json:"profile_name"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID == "" {
		t.Error("response did not include an id")
	}
	if body.ProductID != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("product_id = %q, want %q", body.ProductID, "11111111-1111-1111-1111-111111111111")
	}
	if body.Vendor != "Kontron" || body.ProfileName != "RES-500M" {
		t.Errorf("body = %+v, want Vendor=Kontron ProfileName=RES-500M", body)
	}
}

func TestProvisioningProfileHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newTestRouter(newFakeProvisioningProfileService())

	req := httptest.NewRequest(http.MethodPost, "/provisioning-profiles", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestProvisioningProfileHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := newFakeProvisioningProfileService()
	svc.err = apperror.Invalid("vendor: is required")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/provisioning-profiles", strings.NewReader(`{"vendor":""}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestProvisioningProfileHandlerCreatePropagatesConflictOnUnknownProduct(t *testing.T) {
	svc := newFakeProvisioningProfileService()
	svc.err = apperror.Conflict("create provisioning profile: violates a foreign key relationship")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/provisioning-profiles", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestProvisioningProfileHandlerList(t *testing.T) {
	a := provisioning.ProvisioningProfile{ID: uuid.New(), ProductID: uuid.New(), Vendor: "Kontron", ProfileName: "RES-100M"}
	b := provisioning.ProvisioningProfile{ID: uuid.New(), ProductID: uuid.New(), Vendor: "Kontron", ProfileName: "RES-250M"}
	router := newTestRouter(newFakeProvisioningProfileService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/provisioning-profiles", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		ProvisioningProfiles []struct {
			ID string `json:"id"`
		} `json:"provisioning_profiles"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.ProvisioningProfiles) != 2 {
		t.Fatalf("len(provisioning_profiles) = %d, want 2", len(body.ProvisioningProfiles))
	}
}

func TestProvisioningProfileHandlerGet(t *testing.T) {
	p := provisioning.ProvisioningProfile{ID: uuid.New(), ProductID: uuid.New(), Vendor: "Kontron", ProfileName: "RES-500M"}
	router := newTestRouter(newFakeProvisioningProfileService(p))

	req := httptest.NewRequest(http.MethodGet, "/provisioning-profiles/"+p.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestProvisioningProfileHandlerGetNotFound(t *testing.T) {
	router := newTestRouter(newFakeProvisioningProfileService())

	req := httptest.NewRequest(http.MethodGet, "/provisioning-profiles/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestProvisioningProfileHandlerGetRejectsMalformedID(t *testing.T) {
	router := newTestRouter(newFakeProvisioningProfileService())

	req := httptest.NewRequest(http.MethodGet, "/provisioning-profiles/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestProvisioningProfileHandlerUpdate(t *testing.T) {
	p := provisioning.ProvisioningProfile{ID: uuid.New(), ProductID: uuid.New(), Vendor: "Kontron", ProfileName: "Old Name"}
	router := newTestRouter(newFakeProvisioningProfileService(p))

	req := httptest.NewRequest(http.MethodPut, "/provisioning-profiles/"+p.ID.String(),
		strings.NewReader(`{"product_id":"`+p.ProductID.String()+`","vendor":"Kontron","profile_name":"New Name"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		ProfileName string `json:"profile_name"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ProfileName != "New Name" {
		t.Errorf("profile_name = %q, want %q", body.ProfileName, "New Name")
	}
}

func TestProvisioningProfileHandlerUpdateNotFound(t *testing.T) {
	router := newTestRouter(newFakeProvisioningProfileService())

	req := httptest.NewRequest(http.MethodPut, "/provisioning-profiles/"+uuid.New().String(), strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestProvisioningProfileHandlerDelete(t *testing.T) {
	p := provisioning.ProvisioningProfile{ID: uuid.New(), ProductID: uuid.New(), Vendor: "Kontron", ProfileName: "Temporary"}
	router := newTestRouter(newFakeProvisioningProfileService(p))

	req := httptest.NewRequest(http.MethodDelete, "/provisioning-profiles/"+p.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204 No Content", rec.Body.String())
	}
}

func TestProvisioningProfileHandlerDeleteNotFound(t *testing.T) {
	router := newTestRouter(newFakeProvisioningProfileService())

	req := httptest.NewRequest(http.MethodDelete, "/provisioning-profiles/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
