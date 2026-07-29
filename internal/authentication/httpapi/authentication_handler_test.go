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

	"github.com/paladindigitalgh/palladium-oss/internal/authentication"
	"github.com/paladindigitalgh/palladium-oss/internal/authentication/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeAuthenticationService is the seam httpapi.AuthenticationHandler
// depends on (see its unexported authenticationService interface in
// authentication_handler.go). It lets these tests exercise HTTP-only
// concerns — status codes, JSON shapes, error translation — without a
// real service, repository, database, or encryption involved;
// internal/authentication/service and internal/authentication/postgres
// each have their own tests for the layers below this one.
type fakeAuthenticationService struct {
	auths map[uuid.UUID]authentication.Authentication
	err   error // if set, every method returns this error instead
}

func newFakeAuthenticationService(auths ...authentication.Authentication) *fakeAuthenticationService {
	f := &fakeAuthenticationService{auths: make(map[uuid.UUID]authentication.Authentication)}
	for _, a := range auths {
		f.auths[a.ID] = a
	}
	return f
}

func (f *fakeAuthenticationService) Get(_ context.Context, id uuid.UUID) (authentication.Authentication, error) {
	if f.err != nil {
		return authentication.Authentication{}, f.err
	}
	a, ok := f.auths[id]
	if !ok {
		return authentication.Authentication{}, apperror.NotFound("authentication not found")
	}
	return a, nil
}

func (f *fakeAuthenticationService) List(context.Context) ([]authentication.Authentication, error) {
	if f.err != nil {
		return nil, f.err
	}
	auths := make([]authentication.Authentication, 0, len(f.auths))
	for _, a := range f.auths {
		auths = append(auths, a)
	}
	return auths, nil
}

func (f *fakeAuthenticationService) Create(_ context.Context, a authentication.Authentication) (authentication.Authentication, error) {
	if f.err != nil {
		return authentication.Authentication{}, f.err
	}
	a.ID = uuid.New()
	a.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a.UpdatedAt = a.CreatedAt
	f.auths[a.ID] = a
	return a, nil
}

func (f *fakeAuthenticationService) Update(_ context.Context, a authentication.Authentication) (authentication.Authentication, error) {
	if f.err != nil {
		return authentication.Authentication{}, f.err
	}
	if _, ok := f.auths[a.ID]; !ok {
		return authentication.Authentication{}, apperror.NotFound("authentication not found")
	}
	f.auths[a.ID] = a
	return a, nil
}

func (f *fakeAuthenticationService) Delete(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.auths[id]; !ok {
		return apperror.NotFound("authentication not found")
	}
	delete(f.auths, id)
	return nil
}

// newTestRouter mounts an AuthenticationHandler backed by svc on a real
// chi.Router, so tests that need a URL path parameter (Get/Update/
// Delete's {id}) get one populated the same way production code does,
// rather than faking chi's route context by hand.
func newTestRouter(svc *fakeAuthenticationService) http.Handler {
	handler := httpapi.NewAuthenticationHandler(svc)

	r := chi.NewRouter()
	r.Post("/authentication-methods", handler.Create)
	r.Get("/authentication-methods", handler.List)
	r.Get("/authentication-methods/{id}", handler.Get)
	r.Put("/authentication-methods/{id}", handler.Update)
	r.Delete("/authentication-methods/{id}", handler.Delete)
	return r
}

const validBody = `{"name":"Default Device Login","authentication_type":"Password","username":"admin","password":"hunter2"}`

func TestAuthenticationHandlerCreate(t *testing.T) {
	router := newTestRouter(newFakeAuthenticationService())

	req := httptest.NewRequest(http.MethodPost, "/authentication-methods", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["id"] == "" || body["id"] == nil {
		t.Error("response did not include an id")
	}
	if body["name"] != "Default Device Login" {
		t.Errorf("name = %v, want %q", body["name"], "Default Device Login")
	}
	if body["has_password"] != true {
		t.Errorf("has_password = %v, want true", body["has_password"])
	}
	if body["has_private_key"] != false {
		t.Errorf("has_private_key = %v, want false", body["has_private_key"])
	}
}

// TestAuthenticationHandlerCreateNeverEchoesSecrets is the central
// security proof for this handler: no matter what field names exist in
// the raw JSON response, "password" and "private_key" (the wire names a
// naive implementation might have reused from the request) must not
// appear anywhere in the body.
func TestAuthenticationHandlerCreateNeverEchoesSecrets(t *testing.T) {
	router := newTestRouter(newFakeAuthenticationService())

	req := httptest.NewRequest(http.MethodPost, "/authentication-methods", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	body := rec.Body.String()
	if strings.Contains(body, "hunter2") {
		t.Error("response body contains the plaintext password")
	}
	if strings.Contains(body, `"password"`) {
		t.Error(`response body contains a "password" field`)
	}
	if strings.Contains(body, `"private_key"`) {
		t.Error(`response body contains a "private_key" field`)
	}
}

func TestAuthenticationHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newTestRouter(newFakeAuthenticationService())

	req := httptest.NewRequest(http.MethodPost, "/authentication-methods", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestAuthenticationHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := newFakeAuthenticationService()
	svc.err = apperror.Invalid("username: is required")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/authentication-methods", strings.NewReader(`{"name":"x"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestAuthenticationHandlerCreatePropagatesConflictOnDuplicateName(t *testing.T) {
	svc := newFakeAuthenticationService()
	svc.err = apperror.Conflict("create authentication: already exists")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/authentication-methods", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestAuthenticationHandlerList(t *testing.T) {
	a := authentication.Authentication{ID: uuid.New(), Name: "Alpha", AuthenticationType: authentication.AuthenticationTypePassword, Username: "admin", Password: "x"}
	b := authentication.Authentication{ID: uuid.New(), Name: "Beta", AuthenticationType: authentication.AuthenticationTypePassword, Username: "admin", Password: "y"}
	router := newTestRouter(newFakeAuthenticationService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/authentication-methods", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		AuthenticationMethods []struct {
			ID string `json:"id"`
		} `json:"authentication_methods"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.AuthenticationMethods) != 2 {
		t.Fatalf("len(authentication_methods) = %d, want 2", len(body.AuthenticationMethods))
	}
}

func TestAuthenticationHandlerGet(t *testing.T) {
	a := authentication.Authentication{ID: uuid.New(), Name: "Alpha", AuthenticationType: authentication.AuthenticationTypePassword, Username: "admin", Password: "x"}
	router := newTestRouter(newFakeAuthenticationService(a))

	req := httptest.NewRequest(http.MethodGet, "/authentication-methods/"+a.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestAuthenticationHandlerGetNotFound(t *testing.T) {
	router := newTestRouter(newFakeAuthenticationService())

	req := httptest.NewRequest(http.MethodGet, "/authentication-methods/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestAuthenticationHandlerGetRejectsMalformedID(t *testing.T) {
	router := newTestRouter(newFakeAuthenticationService())

	req := httptest.NewRequest(http.MethodGet, "/authentication-methods/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestAuthenticationHandlerUpdate(t *testing.T) {
	a := authentication.Authentication{ID: uuid.New(), Name: "Alpha", AuthenticationType: authentication.AuthenticationTypePassword, Username: "admin", Password: "x"}
	router := newTestRouter(newFakeAuthenticationService(a))

	req := httptest.NewRequest(http.MethodPut, "/authentication-methods/"+a.ID.String(),
		strings.NewReader(`{"name":"Beta","authentication_type":"Password","username":"root","password":"y"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Name     string `json:"name"`
		Username string `json:"username"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Name != "Beta" {
		t.Errorf("name = %q, want %q", body.Name, "Beta")
	}
	if body.Username != "root" {
		t.Errorf("username = %q, want %q", body.Username, "root")
	}
}

func TestAuthenticationHandlerUpdateNotFound(t *testing.T) {
	router := newTestRouter(newFakeAuthenticationService())

	req := httptest.NewRequest(http.MethodPut, "/authentication-methods/"+uuid.New().String(), strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestAuthenticationHandlerDelete(t *testing.T) {
	a := authentication.Authentication{ID: uuid.New(), Name: "Alpha", AuthenticationType: authentication.AuthenticationTypePassword, Username: "admin", Password: "x"}
	router := newTestRouter(newFakeAuthenticationService(a))

	req := httptest.NewRequest(http.MethodDelete, "/authentication-methods/"+a.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204 No Content", rec.Body.String())
	}
}

func TestAuthenticationHandlerDeleteNotFound(t *testing.T) {
	router := newTestRouter(newFakeAuthenticationService())

	req := httptest.NewRequest(http.MethodDelete, "/authentication-methods/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
