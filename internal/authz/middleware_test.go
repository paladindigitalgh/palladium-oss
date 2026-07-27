package authz_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/authz"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeUserRepository is an in-memory auth.UserRepository. Only GetByID
// matters to Middleware, but the type must satisfy the full interface;
// the other methods are unused stubs. This mirrors the same fake pattern
// used throughout the auth/inventory packages (e.g.
// internal/auth/service_test.go's fakeUserRepository).
type fakeUserRepository struct {
	usersByID map[uuid.UUID]auth.User
	err       error // if set, GetByID always returns this instead
}

func newFakeUserRepository(users ...auth.User) *fakeUserRepository {
	f := &fakeUserRepository{usersByID: make(map[uuid.UUID]auth.User)}
	for _, u := range users {
		f.usersByID[u.ID] = u
	}
	return f
}

func (f *fakeUserRepository) GetByID(_ context.Context, id uuid.UUID) (auth.User, error) {
	if f.err != nil {
		return auth.User{}, f.err
	}
	u, ok := f.usersByID[id]
	if !ok {
		return auth.User{}, apperror.NotFound("user not found")
	}
	return u, nil
}

func (f *fakeUserRepository) GetByEmail(context.Context, string) (auth.User, error) {
	return auth.User{}, apperror.NotFound("not implemented")
}
func (f *fakeUserRepository) Create(_ context.Context, u auth.User) (auth.User, error) { return u, nil }
func (f *fakeUserRepository) UpdatePasswordHash(context.Context, uuid.UUID, string) (auth.User, error) {
	return auth.User{}, apperror.NotFound("not implemented")
}
func (f *fakeUserRepository) Count(context.Context) (int, error) { return len(f.usersByID), nil }

var _ auth.UserRepository = (*fakeUserRepository)(nil)

// requestAs builds a request carrying auth.Claims for user, exactly as
// auth.Middleware would have already stored them by the time
// authz.Middleware runs (see auth.Middleware's doc comment: this
// middleware must run after it). Using auth.ContextWithClaims directly,
// rather than a real JWT, is deliberate: authz.Middleware never validates
// a token itself (goal 3, "do not duplicate JWT validation"), so these
// tests exercise exactly what it actually reads.
func requestAs(user auth.User) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	claims := auth.Claims{UserID: user.ID, Email: user.Email}
	return req.WithContext(auth.ContextWithClaims(req.Context(), claims))
}

func newUser(role auth.Role) auth.User {
	return auth.User{ID: uuid.New(), Email: "user@example.com", Role: role}
}

func assertAllowed(t *testing.T, mw func(http.Handler) http.Handler, req *http.Request) {
	t.Helper()

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req)

	if !nextCalled {
		t.Error("next handler was not called, want it to be")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func assertDenied(t *testing.T, mw func(http.Handler) http.Handler, req *http.Request, wantStatus int) {
	t.Helper()

	nextCalled := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { nextCalled = true })

	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req)

	if nextCalled {
		t.Error("next handler was called, want the request rejected before reaching it")
	}
	if rec.Code != wantStatus {
		t.Errorf("status = %d, want %d; body: %s", rec.Code, wantStatus, rec.Body.String())
	}
}

// TestViewerCannotWriteInventory and TestOperatorCanWriteInventory and
// TestAdministratorCanWriteInventory are this milestone's goal 7 items
// "Viewer cannot modify inventory", "Operator can modify inventory", and
// "Administrator can modify inventory", proven at the middleware level
// (internal/server/router_test.go proves the same facts again through the
// real, fully wired router).

func TestViewerCannotWriteInventory(t *testing.T) {
	viewer := newUser(auth.RoleViewer)
	users := newFakeUserRepository(viewer)
	mw := authz.NewMiddleware(users).RequireInventoryWrite()

	assertDenied(t, mw, requestAs(viewer), http.StatusForbidden)
}

func TestOperatorCanWriteInventory(t *testing.T) {
	operator := newUser(auth.RoleOperator)
	users := newFakeUserRepository(operator)
	mw := authz.NewMiddleware(users).RequireInventoryWrite()

	assertAllowed(t, mw, requestAs(operator))
}

func TestAdministratorCanWriteInventory(t *testing.T) {
	admin := newUser(auth.RoleAdministrator)
	users := newFakeUserRepository(admin)
	mw := authz.NewMiddleware(users).RequireInventoryWrite()

	assertAllowed(t, mw, requestAs(admin))
}

// TestViewerCanReadInventory is goal 7's "Viewer can read inventory".
func TestViewerCanReadInventory(t *testing.T) {
	viewer := newUser(auth.RoleViewer)
	users := newFakeUserRepository(viewer)
	mw := authz.NewMiddleware(users).RequireInventoryRead()

	assertAllowed(t, mw, requestAs(viewer))
}

func TestOperatorCanReadInventory(t *testing.T) {
	operator := newUser(auth.RoleOperator)
	users := newFakeUserRepository(operator)
	mw := authz.NewMiddleware(users).RequireInventoryRead()

	assertAllowed(t, mw, requestAs(operator))
}

func TestAdministratorCanReadInventory(t *testing.T) {
	admin := newUser(auth.RoleAdministrator)
	users := newFakeUserRepository(admin)
	mw := authz.NewMiddleware(users).RequireInventoryRead()

	assertAllowed(t, mw, requestAs(admin))
}

func TestOnlyAdministratorCanManageUsers(t *testing.T) {
	for _, role := range []auth.Role{auth.RoleOperator, auth.RoleViewer} {
		user := newUser(role)
		users := newFakeUserRepository(user)
		mw := authz.NewMiddleware(users).RequireUserManagement()

		assertDenied(t, mw, requestAs(user), http.StatusForbidden)
	}

	admin := newUser(auth.RoleAdministrator)
	users := newFakeUserRepository(admin)
	mw := authz.NewMiddleware(users).RequireUserManagement()

	assertAllowed(t, mw, requestAs(admin))
}

// TestMiddlewareRejectsRequestWithNoAuthenticationClaims proves this
// fails closed (401), rather than treating a request with no upstream
// auth.Middleware claims as anonymous-but-allowed.
func TestMiddlewareRejectsRequestWithNoAuthenticationClaims(t *testing.T) {
	users := newFakeUserRepository()
	mw := authz.NewMiddleware(users).RequireInventoryRead()

	req := httptest.NewRequest(http.MethodGet, "/", nil) // no claims in context

	assertDenied(t, mw, req, http.StatusUnauthorized)
}

// TestMiddlewarePropagatesUserLookupNotFound covers the edge case where a
// JWT is still validly signed and unexpired but the account behind it no
// longer exists (e.g. deleted directly in the database, since there is no
// user management API to do it through yet).
func TestMiddlewarePropagatesUserLookupNotFound(t *testing.T) {
	users := newFakeUserRepository() // empty: GetByID will find nothing
	mw := authz.NewMiddleware(users).RequireInventoryRead()

	ghost := auth.User{ID: uuid.New()}

	assertDenied(t, mw, requestAs(ghost), http.StatusNotFound)
}

// TestMiddlewareDeniesUnrecognizedRole proves the fail-closed default in
// capabilities.go's switch statements actually reaches the middleware: a
// Role that somehow is neither of the three defined values (e.g. stale
// data from before a role was renamed) is never granted anything.
func TestMiddlewareDeniesUnrecognizedRole(t *testing.T) {
	corrupted := newUser(auth.Role("Nonsense"))
	users := newFakeUserRepository(corrupted)
	mw := authz.NewMiddleware(users).RequireInventoryRead()

	assertDenied(t, mw, requestAs(corrupted), http.StatusForbidden)
}
