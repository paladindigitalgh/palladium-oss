package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/auth/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeUserManagementService is the seam httpapi.UserHandler depends on
// (see its unexported userManagementService interface in
// user_handler.go). It lets these tests exercise HTTP-only concerns —
// status codes, JSON shapes, routing, error translation — without a real
// service, repository, or database; internal/auth/service has its own
// tests for the business logic below this layer.
type fakeUserManagementService struct {
	byID map[uuid.UUID]auth.User
	err  error // if set, every method returns this error instead
}

func newFakeUserManagementService(users ...auth.User) *fakeUserManagementService {
	f := &fakeUserManagementService{byID: make(map[uuid.UUID]auth.User)}
	for _, u := range users {
		f.byID[u.ID] = u
	}
	return f
}

func (f *fakeUserManagementService) List(context.Context) ([]auth.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	users := make([]auth.User, 0, len(f.byID))
	for _, u := range f.byID {
		users = append(users, u)
	}
	return users, nil
}

func (f *fakeUserManagementService) Create(_ context.Context, email, _, firstName, lastName string, role auth.Role) (auth.User, error) {
	if f.err != nil {
		return auth.User{}, f.err
	}
	u := auth.User{ID: uuid.New(), Email: email, FirstName: firstName, LastName: lastName, Role: role, Status: auth.UserStatusActive}
	f.byID[u.ID] = u
	return u, nil
}

func (f *fakeUserManagementService) UpdateRole(_ context.Context, id uuid.UUID, role auth.Role) (auth.User, error) {
	if f.err != nil {
		return auth.User{}, f.err
	}
	u, ok := f.byID[id]
	if !ok {
		return auth.User{}, apperror.NotFound("user not found")
	}
	u.Role = role
	f.byID[id] = u
	return u, nil
}

func (f *fakeUserManagementService) Deactivate(_ context.Context, id uuid.UUID) (auth.User, error) {
	if f.err != nil {
		return auth.User{}, f.err
	}
	u, ok := f.byID[id]
	if !ok {
		return auth.User{}, apperror.NotFound("user not found")
	}
	u.Status = auth.UserStatusInactive
	f.byID[id] = u
	return u, nil
}

func (f *fakeUserManagementService) Reactivate(_ context.Context, id uuid.UUID) (auth.User, error) {
	if f.err != nil {
		return auth.User{}, f.err
	}
	u, ok := f.byID[id]
	if !ok {
		return auth.User{}, apperror.NotFound("user not found")
	}
	u.Status = auth.UserStatusActive
	f.byID[id] = u
	return u, nil
}

func newUserTestRouter(svc *fakeUserManagementService) http.Handler {
	handler := httpapi.NewUserHandler(svc)

	r := chi.NewRouter()
	r.Post("/users", handler.Create)
	r.Get("/users", handler.List)
	r.Put("/users/{id}/role", handler.UpdateRole)
	r.Post("/users/{id}/deactivate", handler.Deactivate)
	r.Post("/users/{id}/reactivate", handler.Reactivate)
	return r
}

func TestUserHandlerCreate(t *testing.T) {
	router := newUserTestRouter(newFakeUserManagementService())

	body := `{"email":"jane@example.com","password":"correct horse battery staple","first_name":"Jane","last_name":"Doe","role":"Operator"}`
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp struct {
		ID           string `json:"id"`
		Email        string `json:"email"`
		FirstName    string `json:"first_name"`
		LastName     string `json:"last_name"`
		Role         string `json:"role"`
		Status       string `json:"status"`
		PasswordHash string `json:"password_hash"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Email != "jane@example.com" {
		t.Errorf("email = %q, want %q", resp.Email, "jane@example.com")
	}
	if resp.FirstName != "Jane" || resp.LastName != "Doe" {
		t.Errorf("name = %q %q, want Jane Doe", resp.FirstName, resp.LastName)
	}
	if resp.Role != "Operator" {
		t.Errorf("role = %q, want %q", resp.Role, "Operator")
	}
	if resp.Status != "Active" {
		t.Errorf("status = %q, want %q", resp.Status, "Active")
	}
	if resp.PasswordHash != "" {
		t.Error("response included password_hash, want it never exposed over HTTP")
	}
}

func TestUserHandlerCreateAllowsBlankName(t *testing.T) {
	router := newUserTestRouter(newFakeUserManagementService())

	body := `{"email":"jane@example.com","password":"correct horse battery staple","role":"Operator"}`
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.FirstName != "" || resp.LastName != "" {
		t.Errorf("name = %q %q, want both blank", resp.FirstName, resp.LastName)
	}
}

func TestUserHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newUserTestRouter(newFakeUserManagementService())

	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUserHandlerCreatePropagatesServiceConflict(t *testing.T) {
	svc := newFakeUserManagementService()
	svc.err = apperror.Conflict("email already in use")
	router := newUserTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"email":"jane@example.com","password":"x","role":"Operator"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestUserHandlerList(t *testing.T) {
	a := auth.User{ID: uuid.New(), Email: "a@example.com", Role: auth.RoleAdministrator, Status: auth.UserStatusActive}
	b := auth.User{ID: uuid.New(), Email: "b@example.com", Role: auth.RoleViewer, Status: auth.UserStatusActive}
	router := newUserTestRouter(newFakeUserManagementService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp struct {
		Users []struct {
			ID string `json:"id"`
		} `json:"users"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Users) != 2 {
		t.Fatalf("len(users) = %d, want 2", len(resp.Users))
	}
}

func TestUserHandlerUpdateRole(t *testing.T) {
	u := auth.User{ID: uuid.New(), Email: "jane@example.com", Role: auth.RoleViewer, Status: auth.UserStatusActive}
	router := newUserTestRouter(newFakeUserManagementService(u))

	req := httptest.NewRequest(http.MethodPut, "/users/"+u.ID.String()+"/role", strings.NewReader(`{"role":"Operator"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Role != "Operator" {
		t.Errorf("role = %q, want %q", resp.Role, "Operator")
	}
}

// TestUserHandlerUpdateRolePropagatesLastAdministratorConflict proves the
// handler surfaces UserManagementService's last-active-administrator
// guard (internal/auth/service.user_service_test.go covers the guard
// itself) as 409 Conflict, not a generic 500.
func TestUserHandlerUpdateRolePropagatesLastAdministratorConflict(t *testing.T) {
	admin := auth.User{ID: uuid.New(), Email: "admin@example.com", Role: auth.RoleAdministrator, Status: auth.UserStatusActive}
	svc := newFakeUserManagementService(admin)
	svc.err = apperror.Conflict("cannot remove the last active administrator")
	router := newUserTestRouter(svc)

	req := httptest.NewRequest(http.MethodPut, "/users/"+admin.ID.String()+"/role", strings.NewReader(`{"role":"Operator"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestUserHandlerDeactivate(t *testing.T) {
	u := auth.User{ID: uuid.New(), Email: "jane@example.com", Role: auth.RoleOperator, Status: auth.UserStatusActive}
	router := newUserTestRouter(newFakeUserManagementService(u))

	req := httptest.NewRequest(http.MethodPost, "/users/"+u.ID.String()+"/deactivate", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != "Inactive" {
		t.Errorf("status = %q, want %q", resp.Status, "Inactive")
	}
}

func TestUserHandlerReactivate(t *testing.T) {
	u := auth.User{ID: uuid.New(), Email: "jane@example.com", Role: auth.RoleOperator, Status: auth.UserStatusInactive}
	router := newUserTestRouter(newFakeUserManagementService(u))

	req := httptest.NewRequest(http.MethodPost, "/users/"+u.ID.String()+"/reactivate", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != "Active" {
		t.Errorf("status = %q, want %q", resp.Status, "Active")
	}
}

func TestUserHandlerDeactivateNotFound(t *testing.T) {
	router := newUserTestRouter(newFakeUserManagementService())

	req := httptest.NewRequest(http.MethodPost, "/users/"+uuid.New().String()+"/deactivate", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
