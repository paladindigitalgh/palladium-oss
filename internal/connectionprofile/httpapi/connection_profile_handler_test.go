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

	"github.com/paladindigitalgh/palladium-oss/internal/connectionprofile"
	"github.com/paladindigitalgh/palladium-oss/internal/connectionprofile/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeConnectionProfileService is the seam
// httpapi.ConnectionProfileHandler depends on (see its unexported
// connectionProfileService interface in connection_profile_handler.go).
// It lets these tests exercise HTTP-only concerns — status codes, JSON
// shapes, error translation — without a real service, repository, or
// database involved; internal/connectionprofile/service and
// internal/connectionprofile/postgres each have their own tests for the
// layers below this one.
type fakeConnectionProfileService struct {
	profiles map[uuid.UUID]connectionprofile.ConnectionProfile
	err      error // if set, every method returns this error instead
}

func newFakeConnectionProfileService(profiles ...connectionprofile.ConnectionProfile) *fakeConnectionProfileService {
	f := &fakeConnectionProfileService{profiles: make(map[uuid.UUID]connectionprofile.ConnectionProfile)}
	for _, p := range profiles {
		f.profiles[p.ID] = p
	}
	return f
}

func (f *fakeConnectionProfileService) Get(_ context.Context, id uuid.UUID) (connectionprofile.ConnectionProfile, error) {
	if f.err != nil {
		return connectionprofile.ConnectionProfile{}, f.err
	}
	p, ok := f.profiles[id]
	if !ok {
		return connectionprofile.ConnectionProfile{}, apperror.NotFound("connection profile not found")
	}
	return p, nil
}

func (f *fakeConnectionProfileService) List(context.Context) ([]connectionprofile.ConnectionProfile, error) {
	if f.err != nil {
		return nil, f.err
	}
	profiles := make([]connectionprofile.ConnectionProfile, 0, len(f.profiles))
	for _, p := range f.profiles {
		profiles = append(profiles, p)
	}
	return profiles, nil
}

func (f *fakeConnectionProfileService) Create(_ context.Context, p connectionprofile.ConnectionProfile) (connectionprofile.ConnectionProfile, error) {
	if f.err != nil {
		return connectionprofile.ConnectionProfile{}, f.err
	}
	p.ID = uuid.New()
	p.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p.UpdatedAt = p.CreatedAt
	f.profiles[p.ID] = p
	return p, nil
}

func (f *fakeConnectionProfileService) Update(_ context.Context, p connectionprofile.ConnectionProfile) (connectionprofile.ConnectionProfile, error) {
	if f.err != nil {
		return connectionprofile.ConnectionProfile{}, f.err
	}
	if _, ok := f.profiles[p.ID]; !ok {
		return connectionprofile.ConnectionProfile{}, apperror.NotFound("connection profile not found")
	}
	f.profiles[p.ID] = p
	return p, nil
}

func (f *fakeConnectionProfileService) Delete(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.profiles[id]; !ok {
		return apperror.NotFound("connection profile not found")
	}
	delete(f.profiles, id)
	return nil
}

// newTestRouter mounts a ConnectionProfileHandler backed by svc on a
// real chi.Router, so tests that need a URL path parameter (Get/Update/
// Delete's {id}) get one populated the same way production code does,
// rather than faking chi's route context by hand.
func newTestRouter(svc *fakeConnectionProfileService) http.Handler {
	handler := httpapi.NewConnectionProfileHandler(svc)

	r := chi.NewRouter()
	r.Post("/connection-profiles", handler.Create)
	r.Get("/connection-profiles", handler.List)
	r.Get("/connection-profiles/{id}", handler.Get)
	r.Put("/connection-profiles/{id}", handler.Update)
	r.Delete("/connection-profiles/{id}", handler.Delete)
	return r
}

const validBody = `{"name":"Standard SSH","protocol":"SSH","port":22,"timeout":"30s","host_key_policy":"Strict"}`

func TestConnectionProfileHandlerCreate(t *testing.T) {
	router := newTestRouter(newFakeConnectionProfileService())

	req := httptest.NewRequest(http.MethodPost, "/connection-profiles", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		Protocol      string `json:"protocol"`
		Port          int    `json:"port"`
		Timeout       string `json:"timeout"`
		HostKeyPolicy string `json:"host_key_policy"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID == "" {
		t.Error("response did not include an id")
	}
	if body.Name != "Standard SSH" {
		t.Errorf("name = %q, want %q", body.Name, "Standard SSH")
	}
	if body.Protocol != "SSH" {
		t.Errorf("protocol = %q, want %q", body.Protocol, "SSH")
	}
	if body.Port != 22 {
		t.Errorf("port = %d, want 22", body.Port)
	}
	if body.Timeout != "30s" {
		t.Errorf("timeout = %q, want %q", body.Timeout, "30s")
	}
	if body.HostKeyPolicy != "Strict" {
		t.Errorf("host_key_policy = %q, want %q", body.HostKeyPolicy, "Strict")
	}
}

func TestConnectionProfileHandlerCreateWithAuthenticationID(t *testing.T) {
	router := newTestRouter(newFakeConnectionProfileService())

	authID := uuid.New()
	body := `{"name":"Bound Profile","host_key_policy":"Strict","authentication_id":"` + authID.String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/connection-profiles", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var respBody struct {
		AuthenticationID string `json:"authentication_id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&respBody); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if respBody.AuthenticationID != authID.String() {
		t.Errorf("authentication_id = %q, want %q", respBody.AuthenticationID, authID.String())
	}
}

func TestConnectionProfileHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newTestRouter(newFakeConnectionProfileService())

	req := httptest.NewRequest(http.MethodPost, "/connection-profiles", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestConnectionProfileHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := newFakeConnectionProfileService()
	svc.err = apperror.Invalid("name: is required")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/connection-profiles", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestConnectionProfileHandlerCreatePropagatesConflictOnUnknownAuthentication(t *testing.T) {
	svc := newFakeConnectionProfileService()
	svc.err = apperror.Conflict("create connection profile: violates a foreign key relationship")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/connection-profiles", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestConnectionProfileHandlerList(t *testing.T) {
	a := connectionprofile.ConnectionProfile{ID: uuid.New(), Name: "Alpha", HostKeyPolicy: connectionprofile.HostKeyPolicyStrict}
	b := connectionprofile.ConnectionProfile{ID: uuid.New(), Name: "Beta", HostKeyPolicy: connectionprofile.HostKeyPolicyStrict}
	router := newTestRouter(newFakeConnectionProfileService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/connection-profiles", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		ConnectionProfiles []struct {
			ID string `json:"id"`
		} `json:"connection_profiles"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.ConnectionProfiles) != 2 {
		t.Fatalf("len(connection_profiles) = %d, want 2", len(body.ConnectionProfiles))
	}
}

func TestConnectionProfileHandlerGet(t *testing.T) {
	a := connectionprofile.ConnectionProfile{ID: uuid.New(), Name: "Alpha", HostKeyPolicy: connectionprofile.HostKeyPolicyStrict}
	router := newTestRouter(newFakeConnectionProfileService(a))

	req := httptest.NewRequest(http.MethodGet, "/connection-profiles/"+a.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestConnectionProfileHandlerGetNotFound(t *testing.T) {
	router := newTestRouter(newFakeConnectionProfileService())

	req := httptest.NewRequest(http.MethodGet, "/connection-profiles/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestConnectionProfileHandlerGetRejectsMalformedID(t *testing.T) {
	router := newTestRouter(newFakeConnectionProfileService())

	req := httptest.NewRequest(http.MethodGet, "/connection-profiles/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestConnectionProfileHandlerUpdate(t *testing.T) {
	a := connectionprofile.ConnectionProfile{ID: uuid.New(), Name: "Alpha", HostKeyPolicy: connectionprofile.HostKeyPolicyStrict}
	router := newTestRouter(newFakeConnectionProfileService(a))

	req := httptest.NewRequest(http.MethodPut, "/connection-profiles/"+a.ID.String(),
		strings.NewReader(`{"name":"Beta","host_key_policy":"Insecure"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Name          string `json:"name"`
		HostKeyPolicy string `json:"host_key_policy"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Name != "Beta" {
		t.Errorf("name = %q, want %q", body.Name, "Beta")
	}
	if body.HostKeyPolicy != "Insecure" {
		t.Errorf("host_key_policy = %q, want %q", body.HostKeyPolicy, "Insecure")
	}
}

func TestConnectionProfileHandlerUpdateNotFound(t *testing.T) {
	router := newTestRouter(newFakeConnectionProfileService())

	req := httptest.NewRequest(http.MethodPut, "/connection-profiles/"+uuid.New().String(), strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestConnectionProfileHandlerDelete(t *testing.T) {
	a := connectionprofile.ConnectionProfile{ID: uuid.New(), Name: "Alpha", HostKeyPolicy: connectionprofile.HostKeyPolicyStrict}
	router := newTestRouter(newFakeConnectionProfileService(a))

	req := httptest.NewRequest(http.MethodDelete, "/connection-profiles/"+a.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204 No Content", rec.Body.String())
	}
}

func TestConnectionProfileHandlerDeleteNotFound(t *testing.T) {
	router := newTestRouter(newFakeConnectionProfileService())

	req := httptest.NewRequest(http.MethodDelete, "/connection-profiles/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
