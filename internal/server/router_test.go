package api_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessattachment"
	accessattachmenthttpapi "github.com/paladindigitalgh/palladium-oss/internal/accessattachment/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/accessinterface"
	accessinterfacehttpapi "github.com/paladindigitalgh/palladium-oss/internal/accessinterface/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/accessnetwork"
	accessnetworkhttpapi "github.com/paladindigitalgh/palladium-oss/internal/accessnetwork/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	authhttpapi "github.com/paladindigitalgh/palladium-oss/internal/auth/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/authentication"
	authenticationhttpapi "github.com/paladindigitalgh/palladium-oss/internal/authentication/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/authz"
	"github.com/paladindigitalgh/palladium-oss/internal/catalog"
	cataloghttpapi "github.com/paladindigitalgh/palladium-oss/internal/catalog/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/connectionprofile"
	connectionprofilehttpapi "github.com/paladindigitalgh/palladium-oss/internal/connectionprofile/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/customer"
	customerhttpapi "github.com/paladindigitalgh/palladium-oss/internal/customer/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/diagnostics"
	diagnosticshttpapi "github.com/paladindigitalgh/palladium-oss/internal/diagnostics/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/location"
	locationhttpapi "github.com/paladindigitalgh/palladium-oss/internal/location/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/olt"
	olthttpapi "github.com/paladindigitalgh/palladium-oss/internal/olt/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/ponport"
	ponporthttpapi "github.com/paladindigitalgh/palladium-oss/internal/ponport/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/product"
	producthttpapi "github.com/paladindigitalgh/palladium-oss/internal/product/httpapi"
	api "github.com/paladindigitalgh/palladium-oss/internal/server"
	domainservice "github.com/paladindigitalgh/palladium-oss/internal/service"
	servicehttpapi "github.com/paladindigitalgh/palladium-oss/internal/service/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
	serviceequipmenthttpapi "github.com/paladindigitalgh/palladium-oss/internal/serviceequipment/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceprofile"
	serviceprofilehttpapi "github.com/paladindigitalgh/palladium-oss/internal/serviceprofile/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/workflow"
	workflowhttpapi "github.com/paladindigitalgh/palladium-oss/internal/workflow/httpapi"
)

func TestRouterMountsInventorySchemaEndpoint(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := api.NewRouter(api.Dependencies{
		Logger:  logger,
		Version: "test",
		Commit:  "test",
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/inventory/schema", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

// stubSiteService satisfies whatever interface httpapi.SiteHandler needs
// structurally (Go interfaces need no explicit "implements" declaration),
// so these tests can build a real *httpapi.SiteHandler without a database.
type stubSiteService struct{}

func (stubSiteService) Get(context.Context, uuid.UUID) (inventory.Site, error) {
	return inventory.Site{}, apperror.NotFound("site not found")
}
func (stubSiteService) List(context.Context) ([]inventory.Site, error) { return nil, nil }
func (stubSiteService) Create(_ context.Context, s inventory.Site) (inventory.Site, error) {
	return s, nil
}
func (stubSiteService) Update(_ context.Context, s inventory.Site) (inventory.Site, error) {
	return s, nil
}
func (stubSiteService) Delete(context.Context, uuid.UUID) error { return nil }

// stubUserRepository satisfies auth.UserRepository structurally, always
// reporting the configured role for GetByID regardless of which ID is
// asked for — enough for authz.Middleware, which is all these router
// tests need it for.
type stubUserRepository struct {
	role auth.Role
}

func (s stubUserRepository) GetByID(context.Context, uuid.UUID) (auth.User, error) {
	return auth.User{Role: s.role}, nil
}
func (s stubUserRepository) GetByEmail(context.Context, string) (auth.User, error) {
	return auth.User{}, apperror.NotFound("not implemented in this stub")
}
func (s stubUserRepository) Create(_ context.Context, u auth.User) (auth.User, error) { return u, nil }
func (s stubUserRepository) UpdatePasswordHash(context.Context, uuid.UUID, string) (auth.User, error) {
	return auth.User{}, apperror.NotFound("not implemented in this stub")
}
func (s stubUserRepository) Count(context.Context) (int, error) { return 0, nil }

var _ auth.UserRepository = stubUserRepository{}

// newRouterWithSites builds the real production router (api.NewRouter),
// with a stub service and a stub user repository standing in for the
// database, so these tests prove something no other test file can: that
// /api/v1/sites is actually wired up behind both auth.Middleware and
// authz.Middleware in this file's router.go, with the exact capability
// each HTTP method requires — not just that authentication and
// authorization work correctly in isolation (see
// internal/inventory/httpapi/authenticated_test.go and
// internal/authz/middleware_test.go for far more thorough versions of
// those checks). If someone editing router.go ever forgot to add
// RequireInventoryWrite() to the POST/PUT/DELETE group, or accidentally
// required it on GET too, these tests would catch it; the more isolated
// test files could not.
func newRouterWithSites(tokens *auth.TokenIssuer, role auth.Role) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return api.NewRouter(api.Dependencies{
		Logger:      logger,
		Version:     "test",
		Commit:      "test",
		SiteHandler: httpapi.NewSiteHandler(stubSiteService{}),
		Tokens:      tokens,
		Authz:       authz.NewMiddleware(stubUserRepository{role: role}),
	})
}

func mustIssueToken(t *testing.T, tokens *auth.TokenIssuer) string {
	t.Helper()
	token, err := tokens.IssueToken(auth.User{ID: uuid.New(), Email: "jane@example.com"})
	if err != nil {
		t.Fatalf("IssueToken() = %v", err)
	}
	return token
}

func TestRouterRejectsUnauthenticatedSiteRequests(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithSites(tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/sites/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

// TestRouterViewerCanReadSites is goal 7's "Viewer can read inventory",
// proven through the real, fully wired router.
func TestRouterViewerCanReadSites(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithSites(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sites/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// TestRouterViewerCannotWriteSites is goal 7's "Viewer cannot modify
// inventory", proven through the real, fully wired router.
func TestRouterViewerCannotWriteSites(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithSites(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sites/", strings.NewReader(`{"name":"Test Site"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

// TestRouterOperatorCanWriteSites is goal 7's "Operator can modify
// inventory", proven through the real, fully wired router.
func TestRouterOperatorCanWriteSites(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithSites(tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sites/", strings.NewReader(`{"name":"Test Site"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// TestRouterAdministratorCanWriteSites is goal 7's "Administrator can
// modify inventory", proven through the real, fully wired router.
func TestRouterAdministratorCanWriteSites(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithSites(tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sites/", strings.NewReader(`{"name":"Test Site"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// stubBuildingService satisfies whatever interface httpapi.BuildingHandler
// needs structurally, the same technique stubSiteService above uses.
type stubBuildingService struct{}

func (stubBuildingService) Get(context.Context, uuid.UUID) (inventory.Building, error) {
	return inventory.Building{}, apperror.NotFound("building not found")
}
func (stubBuildingService) List(context.Context) ([]inventory.Building, error) { return nil, nil }
func (stubBuildingService) Create(_ context.Context, b inventory.Building) (inventory.Building, error) {
	return b, nil
}
func (stubBuildingService) Update(_ context.Context, b inventory.Building) (inventory.Building, error) {
	return b, nil
}
func (stubBuildingService) Delete(context.Context, uuid.UUID) error { return nil }

// newRouterWithBuildings mirrors newRouterWithSites exactly, one entity
// over in the same Inventory hierarchy: it proves /api/v1/buildings is
// wired up behind both auth.Middleware and authz.Middleware in the real
// production router, with RequireInventoryRead/RequireInventoryWrite
// applied to the right HTTP methods — the same capabilities /sites uses,
// since Building is Inventory, not a domain of its own.
func newRouterWithBuildings(tokens *auth.TokenIssuer, role auth.Role) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return api.NewRouter(api.Dependencies{
		Logger:          logger,
		Version:         "test",
		Commit:          "test",
		BuildingHandler: httpapi.NewBuildingHandler(stubBuildingService{}),
		Tokens:          tokens,
		Authz:           authz.NewMiddleware(stubUserRepository{role: role}),
	})
}

const validBuildingBody = `{"name":"Test Building","site_id":"` + "00000000-0000-0000-0000-000000000001" + `"}`

func TestRouterRejectsUnauthenticatedBuildingRequests(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithBuildings(tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/buildings/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

// TestRouterViewerCanReadBuildings is goal 7's "Viewer can read
// inventory", proven through the real, fully wired router.
func TestRouterViewerCanReadBuildings(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithBuildings(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/buildings/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// TestRouterViewerCannotWriteBuildings is goal 7's "Viewer cannot modify
// inventory", proven through the real, fully wired router.
func TestRouterViewerCannotWriteBuildings(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithBuildings(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/buildings/", strings.NewReader(validBuildingBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

// TestRouterOperatorCanWriteBuildings is goal 7's "Operator can modify
// inventory", proven through the real, fully wired router.
func TestRouterOperatorCanWriteBuildings(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithBuildings(tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/buildings/", strings.NewReader(validBuildingBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// TestRouterAdministratorCanWriteBuildings is goal 7's "Administrator can
// modify inventory", proven through the real, fully wired router.
func TestRouterAdministratorCanWriteBuildings(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithBuildings(tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/buildings/", strings.NewReader(validBuildingBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// stubRoomService satisfies whatever interface httpapi.RoomHandler needs
// structurally, the same technique stubSiteService above uses.
type stubRoomService struct{}

func (stubRoomService) Get(context.Context, uuid.UUID) (inventory.Room, error) {
	return inventory.Room{}, apperror.NotFound("room not found")
}
func (stubRoomService) List(context.Context) ([]inventory.Room, error) { return nil, nil }
func (stubRoomService) Create(_ context.Context, r inventory.Room) (inventory.Room, error) {
	return r, nil
}
func (stubRoomService) Update(_ context.Context, r inventory.Room) (inventory.Room, error) {
	return r, nil
}
func (stubRoomService) Delete(context.Context, uuid.UUID) error { return nil }

// newRouterWithRooms mirrors newRouterWithSites exactly, two entities
// over in the same Inventory hierarchy: it proves /api/v1/rooms is wired
// up behind both auth.Middleware and authz.Middleware in the real
// production router, with RequireInventoryRead/RequireInventoryWrite
// applied to the right HTTP methods — the same capabilities /sites uses,
// since Room is Inventory, not a domain of its own.
func newRouterWithRooms(tokens *auth.TokenIssuer, role auth.Role) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return api.NewRouter(api.Dependencies{
		Logger:      logger,
		Version:     "test",
		Commit:      "test",
		RoomHandler: httpapi.NewRoomHandler(stubRoomService{}),
		Tokens:      tokens,
		Authz:       authz.NewMiddleware(stubUserRepository{role: role}),
	})
}

const validRoomBody = `{"name":"Test Room","building_id":"` + "00000000-0000-0000-0000-000000000001" + `"}`

func TestRouterRejectsUnauthenticatedRoomRequests(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithRooms(tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/rooms/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

// TestRouterViewerCanReadRooms is goal 7's "Viewer can read inventory",
// proven through the real, fully wired router.
func TestRouterViewerCanReadRooms(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithRooms(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rooms/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// TestRouterViewerCannotWriteRooms is goal 7's "Viewer cannot modify
// inventory", proven through the real, fully wired router.
func TestRouterViewerCannotWriteRooms(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithRooms(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rooms/", strings.NewReader(validRoomBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

// TestRouterOperatorCanWriteRooms is goal 7's "Operator can modify
// inventory", proven through the real, fully wired router.
func TestRouterOperatorCanWriteRooms(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithRooms(tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rooms/", strings.NewReader(validRoomBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// TestRouterAdministratorCanWriteRooms is goal 7's "Administrator can
// modify inventory", proven through the real, fully wired router.
func TestRouterAdministratorCanWriteRooms(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithRooms(tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rooms/", strings.NewReader(validRoomBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// stubDeviceService satisfies whatever interface httpapi.DeviceHandler
// needs structurally, the same technique stubSiteService above uses.
type stubDeviceService struct{}

func (stubDeviceService) Get(context.Context, uuid.UUID) (inventory.Device, error) {
	return inventory.Device{}, apperror.NotFound("device not found")
}
func (stubDeviceService) List(context.Context) ([]inventory.Device, error) { return nil, nil }
func (stubDeviceService) Create(_ context.Context, d inventory.Device) (inventory.Device, error) {
	return d, nil
}
func (stubDeviceService) Update(_ context.Context, d inventory.Device) (inventory.Device, error) {
	return d, nil
}
func (stubDeviceService) Delete(context.Context, uuid.UUID) error { return nil }

// newRouterWithDevices mirrors newRouterWithSites exactly, one entity
// over in the same Inventory hierarchy: it proves /api/v1/devices is
// wired up behind both auth.Middleware and authz.Middleware in the real
// production router, with RequireInventoryRead/RequireInventoryWrite
// applied to the right HTTP methods — the same capabilities /sites uses,
// since Device is Inventory, not a domain of its own.
func newRouterWithDevices(tokens *auth.TokenIssuer, role auth.Role) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return api.NewRouter(api.Dependencies{
		Logger:        logger,
		Version:       "test",
		Commit:        "test",
		DeviceHandler: httpapi.NewDeviceHandler(stubDeviceService{}),
		Tokens:        tokens,
		Authz:         authz.NewMiddleware(stubUserRepository{role: role}),
	})
}

const validDeviceBody = `{"name":"Test Device","manufacturer":"Calix","model":"716GE","serial_number":"CXNK00112233","status":"InStock"}`

func TestRouterRejectsUnauthenticatedDeviceRequests(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithDevices(tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/devices/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

// TestRouterViewerCanReadDevices is goal 7's "Viewer can read
// inventory", proven through the real, fully wired router.
func TestRouterViewerCanReadDevices(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithDevices(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// TestRouterViewerCannotWriteDevices is goal 7's "Viewer cannot modify
// inventory", proven through the real, fully wired router.
func TestRouterViewerCannotWriteDevices(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithDevices(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/", strings.NewReader(validDeviceBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

// TestRouterOperatorCanWriteDevices is goal 7's "Operator can modify
// inventory", proven through the real, fully wired router.
func TestRouterOperatorCanWriteDevices(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithDevices(tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/", strings.NewReader(validDeviceBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// TestRouterAdministratorCanWriteDevices is goal 7's "Administrator can
// modify inventory", proven through the real, fully wired router.
func TestRouterAdministratorCanWriteDevices(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithDevices(tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/", strings.NewReader(validDeviceBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// stubCustomerService satisfies whatever interface
// customerhttpapi.CustomerHandler needs structurally, the same technique
// stubSiteService above uses.
type stubCustomerService struct{}

func (stubCustomerService) Get(context.Context, uuid.UUID) (customer.Customer, error) {
	return customer.Customer{}, apperror.NotFound("customer not found")
}
func (stubCustomerService) List(context.Context) ([]customer.Customer, error) { return nil, nil }
func (stubCustomerService) Create(_ context.Context, c customer.Customer) (customer.Customer, error) {
	return c, nil
}
func (stubCustomerService) Update(_ context.Context, c customer.Customer) (customer.Customer, error) {
	return c, nil
}
func (stubCustomerService) Delete(context.Context, uuid.UUID) error { return nil }

// newRouterWithCustomers mirrors newRouterWithSites exactly, one resource
// over: it proves /api/v1/customers is wired up behind both
// auth.Middleware and authz.Middleware in the real production router,
// with RequireCustomerRead/RequireCustomerWrite applied to the right HTTP
// methods (goal 6: "apply the same authorization model as Sites"). See
// internal/customer/httpapi/authenticated_test.go for a far more thorough
// version of the same checks, scoped to that package.
func newRouterWithCustomers(tokens *auth.TokenIssuer, role auth.Role) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return api.NewRouter(api.Dependencies{
		Logger:          logger,
		Version:         "test",
		Commit:          "test",
		CustomerHandler: customerhttpapi.NewCustomerHandler(stubCustomerService{}),
		Tokens:          tokens,
		Authz:           authz.NewMiddleware(stubUserRepository{role: role}),
	})
}

const validCustomerBody = `{"name":"Test Customer","customer_type":"Residential","status":"Active"}`

func TestRouterRejectsUnauthenticatedCustomerRequests(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithCustomers(tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/customers/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

// TestRouterViewerCanReadCustomers is goal 7's "Viewer can read
// inventory", applied to Customers, proven through the real, fully wired
// router.
func TestRouterViewerCanReadCustomers(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithCustomers(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// TestRouterViewerCannotWriteCustomers is goal 7's "Viewer cannot modify
// inventory", applied to Customers, proven through the real, fully wired
// router.
func TestRouterViewerCannotWriteCustomers(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithCustomers(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/customers/", strings.NewReader(validCustomerBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

// TestRouterOperatorCanWriteCustomers is goal 7's "Operator can modify
// inventory", applied to Customers, proven through the real, fully wired
// router.
func TestRouterOperatorCanWriteCustomers(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithCustomers(tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/customers/", strings.NewReader(validCustomerBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// TestRouterAdministratorCanWriteCustomers is goal 7's "Administrator can
// modify inventory", applied to Customers, proven through the real, fully
// wired router.
func TestRouterAdministratorCanWriteCustomers(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithCustomers(tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/customers/", strings.NewReader(validCustomerBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// stubLocationService satisfies whatever interface
// locationhttpapi.LocationHandler needs structurally, the same technique
// stubSiteService and stubCustomerService above use.
type stubLocationService struct{}

func (stubLocationService) Get(context.Context, uuid.UUID) (location.Location, error) {
	return location.Location{}, apperror.NotFound("location not found")
}
func (stubLocationService) List(context.Context) ([]location.Location, error) { return nil, nil }
func (stubLocationService) Create(_ context.Context, l location.Location) (location.Location, error) {
	return l, nil
}
func (stubLocationService) Update(_ context.Context, l location.Location) (location.Location, error) {
	return l, nil
}
func (stubLocationService) Delete(context.Context, uuid.UUID) error { return nil }

// newRouterWithLocations mirrors newRouterWithCustomers exactly, one
// resource over: it proves /api/v1/locations is wired up behind both
// auth.Middleware and authz.Middleware in the real production router, with
// RequireLocationRead/RequireLocationWrite applied to the right HTTP
// methods ("match Customer permissions"). See
// internal/location/httpapi/authenticated_test.go for a far more thorough
// version of the same checks, scoped to that package.
func newRouterWithLocations(tokens *auth.TokenIssuer, role auth.Role) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return api.NewRouter(api.Dependencies{
		Logger:          logger,
		Version:         "test",
		Commit:          "test",
		LocationHandler: locationhttpapi.NewLocationHandler(stubLocationService{}),
		Tokens:          tokens,
		Authz:           authz.NewMiddleware(stubUserRepository{role: role}),
	})
}

const validLocationBody = `{"customer_id":"11111111-1111-1111-1111-111111111111","name":"Test Location","type":"Service","status":"Active"}`

func TestRouterRejectsUnauthenticatedLocationRequests(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithLocations(tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/locations/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

// TestRouterViewerCanReadLocations is "match Customer permissions",
// proven through the real, fully wired router.
func TestRouterViewerCanReadLocations(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithLocations(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/locations/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// TestRouterViewerCannotWriteLocations is "match Customer permissions",
// proven through the real, fully wired router.
func TestRouterViewerCannotWriteLocations(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithLocations(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/locations/", strings.NewReader(validLocationBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

// TestRouterOperatorCanWriteLocations is "match Customer permissions",
// proven through the real, fully wired router.
func TestRouterOperatorCanWriteLocations(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithLocations(tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/locations/", strings.NewReader(validLocationBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// TestRouterAdministratorCanWriteLocations is "match Customer
// permissions", proven through the real, fully wired router.
func TestRouterAdministratorCanWriteLocations(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithLocations(tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/locations/", strings.NewReader(validLocationBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// stubCatalogService satisfies whatever interface
// cataloghttpapi.CatalogHandler needs structurally, the same technique
// stubSiteService, stubCustomerService, and stubLocationService above use.
type stubCatalogService struct{}

func (stubCatalogService) Get(context.Context, uuid.UUID) (catalog.ProductCatalog, error) {
	return catalog.ProductCatalog{}, apperror.NotFound("catalog not found")
}
func (stubCatalogService) List(context.Context) ([]catalog.ProductCatalog, error) { return nil, nil }
func (stubCatalogService) Create(_ context.Context, c catalog.ProductCatalog) (catalog.ProductCatalog, error) {
	return c, nil
}
func (stubCatalogService) Update(_ context.Context, c catalog.ProductCatalog) (catalog.ProductCatalog, error) {
	return c, nil
}
func (stubCatalogService) Delete(context.Context, uuid.UUID) error { return nil }

// stubProductService satisfies whatever interface
// producthttpapi.ProductHandler needs structurally, the same technique
// used above.
type stubProductService struct{}

func (stubProductService) Get(context.Context, uuid.UUID) (product.Product, error) {
	return product.Product{}, apperror.NotFound("product not found")
}
func (stubProductService) List(context.Context) ([]product.Product, error) { return nil, nil }
func (stubProductService) Create(_ context.Context, p product.Product) (product.Product, error) {
	return p, nil
}
func (stubProductService) Update(_ context.Context, p product.Product) (product.Product, error) {
	return p, nil
}
func (stubProductService) Delete(context.Context, uuid.UUID) error { return nil }

// newRouterWithCatalog mirrors newRouterWithLocations exactly, one
// resource over: it proves /api/v1/catalogs and /api/v1/products are both
// wired up behind auth.Middleware and authz.Middleware in the real
// production router, sharing RequireCatalogRead/RequireCatalogWrite (see
// authz.CanReadCatalog's doc comment for why one capability pair guards
// both resources). See internal/catalog/httpapi/authenticated_test.go and
// internal/product/httpapi/authenticated_test.go for far more thorough
// versions of the same checks, scoped to each package.
func newRouterWithCatalog(tokens *auth.TokenIssuer, role auth.Role) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return api.NewRouter(api.Dependencies{
		Logger:         logger,
		Version:        "test",
		Commit:         "test",
		CatalogHandler: cataloghttpapi.NewCatalogHandler(stubCatalogService{}),
		ProductHandler: producthttpapi.NewProductHandler(stubProductService{}),
		Tokens:         tokens,
		Authz:          authz.NewMiddleware(stubUserRepository{role: role}),
	})
}

const validCatalogBody = `{"name":"Test Catalog","status":"Active"}`
const validProductBody = `{"catalog_id":"11111111-1111-1111-1111-111111111111","name":"Test Product","category":"Internet","status":"Active"}`

func TestRouterRejectsUnauthenticatedCatalogRequests(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithCatalog(tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/catalogs/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestRouterRejectsUnauthenticatedProductRequests(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithCatalog(tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/products/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

// TestRouterViewerCanReadCatalog is "apply the standard RBAC matrix",
// proven through the real, fully wired router.
func TestRouterViewerCanReadCatalog(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithCatalog(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/catalogs/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// TestRouterViewerCanReadProducts is "apply the standard RBAC matrix",
// proven through the real, fully wired router.
func TestRouterViewerCanReadProducts(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithCatalog(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// TestRouterViewerCannotWriteCatalog is "apply the standard RBAC matrix",
// proven through the real, fully wired router.
func TestRouterViewerCannotWriteCatalog(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithCatalog(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/catalogs/", strings.NewReader(validCatalogBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

// TestRouterViewerCannotWriteProducts is "apply the standard RBAC matrix",
// proven through the real, fully wired router.
func TestRouterViewerCannotWriteProducts(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithCatalog(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/products/", strings.NewReader(validProductBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

// TestRouterOperatorCanWriteCatalog is "apply the standard RBAC matrix",
// proven through the real, fully wired router.
func TestRouterOperatorCanWriteCatalog(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithCatalog(tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/catalogs/", strings.NewReader(validCatalogBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// TestRouterOperatorCanWriteProducts is "apply the standard RBAC matrix",
// proven through the real, fully wired router.
func TestRouterOperatorCanWriteProducts(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithCatalog(tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/products/", strings.NewReader(validProductBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// TestRouterAdministratorCanWriteCatalog is "apply the standard RBAC
// matrix", proven through the real, fully wired router.
func TestRouterAdministratorCanWriteCatalog(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithCatalog(tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/catalogs/", strings.NewReader(validCatalogBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// TestRouterAdministratorCanWriteProducts is "apply the standard RBAC
// matrix", proven through the real, fully wired router.
func TestRouterAdministratorCanWriteProducts(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithCatalog(tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/products/", strings.NewReader(validProductBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// stubServiceService satisfies whatever interface
// servicehttpapi.ServiceHandler needs structurally, the same technique
// stubSiteService, stubCustomerService, stubLocationService, and
// stubCatalogService/stubProductService above use.
type stubServiceService struct{}

func (stubServiceService) Get(context.Context, uuid.UUID) (domainservice.Service, error) {
	return domainservice.Service{}, apperror.NotFound("service not found")
}
func (stubServiceService) List(context.Context) ([]domainservice.Service, error) { return nil, nil }
func (stubServiceService) Create(_ context.Context, s domainservice.Service) (domainservice.Service, error) {
	return s, nil
}
func (stubServiceService) Update(_ context.Context, s domainservice.Service) (domainservice.Service, error) {
	return s, nil
}
func (stubServiceService) Delete(context.Context, uuid.UUID) error { return nil }

// newRouterWithServices mirrors newRouterWithCatalog exactly, one
// resource over: it proves /api/v1/services is wired up behind
// auth.Middleware and authz.Middleware in the real production router,
// using its own dedicated RequireServiceRead/RequireServiceWrite (see
// authz.CanReadServices's doc comment for why Service does not share
// Catalog's/Location's capability pair). See
// internal/service/httpapi/authenticated_test.go for a far more thorough
// version of the same checks, scoped to that package.
func newRouterWithServices(tokens *auth.TokenIssuer, role auth.Role) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return api.NewRouter(api.Dependencies{
		Logger:         logger,
		Version:        "test",
		Commit:         "test",
		ServiceHandler: servicehttpapi.NewServiceHandler(stubServiceService{}),
		Tokens:         tokens,
		Authz:          authz.NewMiddleware(stubUserRepository{role: role}),
	})
}

const validServiceBody = `{"location_id":"11111111-1111-1111-1111-111111111111","product_id":"22222222-2222-2222-2222-222222222222","service_profile_id":"33333333-3333-3333-3333-333333333333","status":"Pending"}`

func TestRouterRejectsUnauthenticatedServiceRequests(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithServices(tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/services/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

// TestRouterViewerCanReadServices is "apply the standard RBAC matrix",
// proven through the real, fully wired router.
func TestRouterViewerCanReadServices(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithServices(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/services/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// TestRouterViewerCannotWriteServices is "apply the standard RBAC
// matrix", proven through the real, fully wired router.
func TestRouterViewerCannotWriteServices(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithServices(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/services/", strings.NewReader(validServiceBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

// TestRouterOperatorCanWriteServices is "apply the standard RBAC
// matrix", proven through the real, fully wired router.
func TestRouterOperatorCanWriteServices(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithServices(tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/services/", strings.NewReader(validServiceBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// TestRouterAdministratorCanWriteServices is "apply the standard RBAC
// matrix", proven through the real, fully wired router.
func TestRouterAdministratorCanWriteServices(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithServices(tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/services/", strings.NewReader(validServiceBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// stubServiceEquipmentService satisfies whatever interface
// serviceequipmenthttpapi.ServiceEquipmentHandler needs structurally, the
// same technique stubSiteService and every other stub above uses.
type stubServiceEquipmentService struct{}

func (stubServiceEquipmentService) Get(context.Context, uuid.UUID) (serviceequipment.ServiceEquipment, error) {
	return serviceequipment.ServiceEquipment{}, apperror.NotFound("service equipment not found")
}
func (stubServiceEquipmentService) List(context.Context) ([]serviceequipment.ServiceEquipment, error) {
	return nil, nil
}
func (stubServiceEquipmentService) Create(_ context.Context, e serviceequipment.ServiceEquipment) (serviceequipment.ServiceEquipment, error) {
	return e, nil
}
func (stubServiceEquipmentService) Update(_ context.Context, e serviceequipment.ServiceEquipment) (serviceequipment.ServiceEquipment, error) {
	return e, nil
}
func (stubServiceEquipmentService) Delete(context.Context, uuid.UUID) error { return nil }

// newRouterWithServiceEquipment mirrors newRouterWithServices exactly,
// one resource over: it proves /api/v1/service-equipment is wired up
// behind auth.Middleware and authz.Middleware in the real production
// router, using its own dedicated
// RequireServiceEquipmentRead/RequireServiceEquipmentWrite (see
// authz.CanReadServiceEquipment's doc comment for why Service Equipment
// does not share Service's capability pair). See
// internal/serviceequipment/httpapi/authenticated_test.go for a far more
// thorough version of the same checks, scoped to that package.
func newRouterWithServiceEquipment(tokens *auth.TokenIssuer, role auth.Role) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return api.NewRouter(api.Dependencies{
		Logger:                  logger,
		Version:                 "test",
		Commit:                  "test",
		ServiceEquipmentHandler: serviceequipmenthttpapi.NewServiceEquipmentHandler(stubServiceEquipmentService{}),
		Tokens:                  tokens,
		Authz:                   authz.NewMiddleware(stubUserRepository{role: role}),
	})
}

const validServiceEquipmentBody = `{"service_id":"11111111-1111-1111-1111-111111111111","device_id":"22222222-2222-2222-2222-222222222222","role":"ONU"}`

func TestRouterRejectsUnauthenticatedServiceEquipmentRequests(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithServiceEquipment(tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/service-equipment/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

// TestRouterViewerCanReadServiceEquipment is "apply the standard RBAC
// matrix", proven through the real, fully wired router.
func TestRouterViewerCanReadServiceEquipment(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithServiceEquipment(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/service-equipment/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// TestRouterViewerCannotWriteServiceEquipment is "apply the standard RBAC
// matrix", proven through the real, fully wired router.
func TestRouterViewerCannotWriteServiceEquipment(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithServiceEquipment(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/service-equipment/", strings.NewReader(validServiceEquipmentBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

// TestRouterOperatorCanWriteServiceEquipment is "apply the standard RBAC
// matrix", proven through the real, fully wired router.
func TestRouterOperatorCanWriteServiceEquipment(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithServiceEquipment(tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/service-equipment/", strings.NewReader(validServiceEquipmentBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// TestRouterAdministratorCanWriteServiceEquipment is "apply the standard
// RBAC matrix", proven through the real, fully wired router.
func TestRouterAdministratorCanWriteServiceEquipment(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithServiceEquipment(tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/service-equipment/", strings.NewReader(validServiceEquipmentBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// stubWorkflowService satisfies whatever interface
// workflowhttpapi.WorkflowHandler needs structurally, the same technique
// stubSiteService and every other stub above uses.
type stubWorkflowService struct{}

// Get returns a real (if empty) instance for any id, unlike most other
// stubs' NotFound default: WorkflowHandler.Execute calls Get again after
// a successful Engine.Execute to return the up-to-date instance, so this
// stub must satisfy that second call, not just the initial lookup.
func (stubWorkflowService) Get(_ context.Context, id uuid.UUID) (workflow.Instance, error) {
	return workflow.Instance{ID: id, Status: workflow.StatusSucceeded}, nil
}
func (stubWorkflowService) List(context.Context) ([]workflow.Instance, error) {
	return nil, nil
}
func (stubWorkflowService) ListByServiceID(context.Context, uuid.UUID) ([]workflow.Instance, error) {
	return nil, nil
}
func (stubWorkflowService) Create(_ context.Context, i workflow.Instance) (workflow.Instance, error) {
	return i, nil
}
func (stubWorkflowService) Delete(context.Context, uuid.UUID) error { return nil }
func (stubWorkflowService) Cancel(_ context.Context, id uuid.UUID) (workflow.Instance, error) {
	return workflow.Instance{ID: id, Status: workflow.StatusCancelled}, nil
}
func (stubWorkflowService) Retry(_ context.Context, id uuid.UUID) (workflow.Instance, error) {
	return workflow.Instance{ID: id, Status: workflow.StatusPending}, nil
}

// stubWorkflowEngine satisfies workflowhttpapi.WorkflowHandler's engine
// dependency, always succeeding without touching a Service, equipment,
// or a real Plugin.
type stubWorkflowEngine struct{}

func (stubWorkflowEngine) Execute(context.Context, uuid.UUID) error { return nil }

// newRouterWithWorkflow mirrors newRouterWithServiceEquipment exactly,
// one resource over: it proves /api/v1/workflow-instances (including its
// action sub-routes) is wired up behind auth.Middleware and
// authz.Middleware in the real production router, using its own
// dedicated RequireWorkflowRead/RequireWorkflowWrite.
func newRouterWithWorkflow(tokens *auth.TokenIssuer, role auth.Role) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return api.NewRouter(api.Dependencies{
		Logger:          logger,
		Version:         "test",
		Commit:          "test",
		WorkflowHandler: workflowhttpapi.NewWorkflowHandler(stubWorkflowService{}, stubWorkflowEngine{}),
		Tokens:          tokens,
		Authz:           authz.NewMiddleware(stubUserRepository{role: role}),
	})
}

const validWorkflowBody = `{"service_id":"11111111-1111-1111-1111-111111111111","definition_name":"provision-service"}`

func TestRouterRejectsUnauthenticatedWorkflowRequests(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithWorkflow(tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/workflow-instances/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

// TestRouterViewerCanReadWorkflow is "apply the standard RBAC matrix",
// proven through the real, fully wired router.
func TestRouterViewerCanReadWorkflow(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithWorkflow(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflow-instances/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// TestRouterViewerCannotWriteWorkflow is "apply the standard RBAC
// matrix", proven through the real, fully wired router.
func TestRouterViewerCannotWriteWorkflow(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithWorkflow(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow-instances/", strings.NewReader(validWorkflowBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

// TestRouterViewerCannotDriveWorkflowStateTransitions proves the action
// sub-routes (execute/cancel/retry) are covered by the same write
// capability as create/delete, not left unguarded.
func TestRouterViewerCannotDriveWorkflowStateTransitions(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithWorkflow(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	instanceID := uuid.New()
	for _, action := range []string{"execute", "cancel", "retry"} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow-instances/"+instanceID.String()+"/"+action, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("action %q: status = %d, want %d; body: %s", action, rec.Code, http.StatusForbidden, rec.Body.String())
		}
	}
}

// TestRouterOperatorCanWriteWorkflow is "apply the standard RBAC
// matrix", proven through the real, fully wired router.
func TestRouterOperatorCanWriteWorkflow(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithWorkflow(tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow-instances/", strings.NewReader(validWorkflowBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// TestRouterAdministratorCanWriteWorkflow is "apply the standard RBAC
// matrix", proven through the real, fully wired router.
func TestRouterAdministratorCanWriteWorkflow(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithWorkflow(tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow-instances/", strings.NewReader(validWorkflowBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// TestRouterAdministratorCanDriveWorkflowStateTransitions proves the
// action sub-routes are reachable at all through the real router for a
// role that has the write capability.
func TestRouterAdministratorCanDriveWorkflowStateTransitions(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithWorkflow(tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	instanceID := uuid.New()
	for _, action := range []string{"execute", "cancel", "retry"} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow-instances/"+instanceID.String()+"/"+action, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("action %q: status = %d, want %d; body: %s", action, rec.Code, http.StatusOK, rec.Body.String())
		}
	}
}

// stubAccessNetworkService, stubOLTService, and stubPONPortService each
// satisfy whatever interface their respective httpapi.*Handler needs
// structurally, the same technique stubSiteService and every other stub
// above uses.
type stubAccessNetworkService struct{}

func (stubAccessNetworkService) Get(context.Context, uuid.UUID) (accessnetwork.AccessNetwork, error) {
	return accessnetwork.AccessNetwork{}, apperror.NotFound("access network not found")
}
func (stubAccessNetworkService) List(context.Context) ([]accessnetwork.AccessNetwork, error) {
	return nil, nil
}
func (stubAccessNetworkService) Create(_ context.Context, a accessnetwork.AccessNetwork) (accessnetwork.AccessNetwork, error) {
	return a, nil
}
func (stubAccessNetworkService) Update(_ context.Context, a accessnetwork.AccessNetwork) (accessnetwork.AccessNetwork, error) {
	return a, nil
}
func (stubAccessNetworkService) Delete(context.Context, uuid.UUID) error { return nil }

type stubOLTService struct{}

func (stubOLTService) Get(context.Context, uuid.UUID) (olt.OLT, error) {
	return olt.OLT{}, apperror.NotFound("olt not found")
}
func (stubOLTService) List(context.Context) ([]olt.OLT, error) { return nil, nil }
func (stubOLTService) Create(_ context.Context, o olt.OLT) (olt.OLT, error) {
	return o, nil
}
func (stubOLTService) Update(_ context.Context, o olt.OLT) (olt.OLT, error) {
	return o, nil
}
func (stubOLTService) Delete(context.Context, uuid.UUID) error { return nil }

type stubPONPortService struct{}

func (stubPONPortService) Get(context.Context, uuid.UUID) (ponport.PONPort, error) {
	return ponport.PONPort{}, apperror.NotFound("pon port not found")
}
func (stubPONPortService) List(context.Context) ([]ponport.PONPort, error) { return nil, nil }
func (stubPONPortService) Create(_ context.Context, p ponport.PONPort) (ponport.PONPort, error) {
	return p, nil
}
func (stubPONPortService) Update(_ context.Context, p ponport.PONPort) (ponport.PONPort, error) {
	return p, nil
}
func (stubPONPortService) Delete(context.Context, uuid.UUID) error { return nil }

// newRouterWithAccessNetwork mirrors newRouterWithCatalog exactly, one
// domain over: it proves /api/v1/access-networks, /api/v1/olts, and
// /api/v1/pon-ports are all wired up behind auth.Middleware and
// authz.Middleware in the real production router, sharing
// RequireAccessNetworkRead/RequireAccessNetworkWrite (see
// authz.CanReadAccessNetwork's doc comment for why one capability pair
// guards all three resources). See each domain's own
// httpapi/authenticated_test.go for far more thorough versions of the
// same checks, scoped to that package.
func newRouterWithAccessNetwork(tokens *auth.TokenIssuer, role auth.Role) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return api.NewRouter(api.Dependencies{
		Logger:               logger,
		Version:              "test",
		Commit:               "test",
		AccessNetworkHandler: accessnetworkhttpapi.NewAccessNetworkHandler(stubAccessNetworkService{}),
		OLTHandler:           olthttpapi.NewOLTHandler(stubOLTService{}),
		PONPortHandler:       ponporthttpapi.NewPONPortHandler(stubPONPortService{}),
		Tokens:               tokens,
		Authz:                authz.NewMiddleware(stubUserRepository{role: role}),
	})
}

const validAccessNetworkBody = `{"name":"Test Access Network","status":"Active"}`
const validOLTBody = `{"access_network_id":"11111111-1111-1111-1111-111111111111","name":"Test OLT","vendor":"Kontron"}`
const validPONPortBody = `{"olt_id":"11111111-1111-1111-1111-111111111111","port_number":1}`

func TestRouterRejectsUnauthenticatedAccessNetworkRequests(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithAccessNetwork(tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/access-networks/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestRouterRejectsUnauthenticatedOLTRequests(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithAccessNetwork(tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/olts/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestRouterRejectsUnauthenticatedPONPortRequests(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithAccessNetwork(tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/pon-ports/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

// TestRouterViewerCanReadAccessNetworkOLTsAndPONPorts is "apply the
// standard RBAC matrix", proven through the real, fully wired router,
// for all three resources at once.
func TestRouterViewerCanReadAccessNetworkOLTsAndPONPorts(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithAccessNetwork(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	for _, path := range []string{"/api/v1/access-networks/", "/api/v1/olts/", "/api/v1/pon-ports/"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("GET %s: status = %d, want %d; body: %s", path, rec.Code, http.StatusOK, rec.Body.String())
		}
	}
}

// TestRouterViewerCannotWriteAccessNetworkOLTsOrPONPorts is "apply the
// standard RBAC matrix", proven through the real, fully wired router,
// for all three resources at once.
func TestRouterViewerCannotWriteAccessNetworkOLTsOrPONPorts(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithAccessNetwork(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	cases := []struct {
		path string
		body string
	}{
		{"/api/v1/access-networks/", validAccessNetworkBody},
		{"/api/v1/olts/", validOLTBody},
		{"/api/v1/pon-ports/", validPONPortBody},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodPost, c.path, strings.NewReader(c.body))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("POST %s: status = %d, want %d; body: %s", c.path, rec.Code, http.StatusForbidden, rec.Body.String())
		}
	}
}

// TestRouterOperatorCanWriteAccessNetworkOLTsAndPONPorts is "apply the
// standard RBAC matrix", proven through the real, fully wired router,
// for all three resources at once.
func TestRouterOperatorCanWriteAccessNetworkOLTsAndPONPorts(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithAccessNetwork(tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	cases := []struct {
		path string
		body string
	}{
		{"/api/v1/access-networks/", validAccessNetworkBody},
		{"/api/v1/olts/", validOLTBody},
		{"/api/v1/pon-ports/", validPONPortBody},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodPost, c.path, strings.NewReader(c.body))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("POST %s: status = %d, want %d; body: %s", c.path, rec.Code, http.StatusCreated, rec.Body.String())
		}
	}
}

// TestRouterAdministratorCanWriteAccessNetworkOLTsAndPONPorts is "apply
// the standard RBAC matrix", proven through the real, fully wired
// router, for all three resources at once.
func TestRouterAdministratorCanWriteAccessNetworkOLTsAndPONPorts(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithAccessNetwork(tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	cases := []struct {
		path string
		body string
	}{
		{"/api/v1/access-networks/", validAccessNetworkBody},
		{"/api/v1/olts/", validOLTBody},
		{"/api/v1/pon-ports/", validPONPortBody},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodPost, c.path, strings.NewReader(c.body))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("POST %s: status = %d, want %d; body: %s", c.path, rec.Code, http.StatusCreated, rec.Body.String())
		}
	}
}

// stubAccessInterfaceService and stubAccessAttachmentService each
// satisfy whatever interface their respective httpapi.*Handler needs
// structurally, the same technique stubAccessNetworkService and every
// other stub above uses.
type stubAccessInterfaceService struct{}

func (stubAccessInterfaceService) Get(context.Context, uuid.UUID) (accessinterface.AccessInterface, error) {
	return accessinterface.AccessInterface{}, apperror.NotFound("access interface not found")
}
func (stubAccessInterfaceService) List(context.Context) ([]accessinterface.AccessInterface, error) {
	return nil, nil
}
func (stubAccessInterfaceService) Create(_ context.Context, a accessinterface.AccessInterface) (accessinterface.AccessInterface, error) {
	return a, nil
}
func (stubAccessInterfaceService) Update(_ context.Context, a accessinterface.AccessInterface) (accessinterface.AccessInterface, error) {
	return a, nil
}
func (stubAccessInterfaceService) Delete(context.Context, uuid.UUID) error { return nil }

type stubAccessAttachmentService struct{}

func (stubAccessAttachmentService) Get(context.Context, uuid.UUID) (accessattachment.AccessAttachment, error) {
	return accessattachment.AccessAttachment{}, apperror.NotFound("access attachment not found")
}
func (stubAccessAttachmentService) List(context.Context) ([]accessattachment.AccessAttachment, error) {
	return nil, nil
}
func (stubAccessAttachmentService) Create(_ context.Context, a accessattachment.AccessAttachment) (accessattachment.AccessAttachment, error) {
	return a, nil
}
func (stubAccessAttachmentService) Update(_ context.Context, a accessattachment.AccessAttachment) (accessattachment.AccessAttachment, error) {
	return a, nil
}
func (stubAccessAttachmentService) Delete(context.Context, uuid.UUID) error { return nil }

// newRouterWithAccessTopology mirrors newRouterWithAccessNetwork exactly,
// one domain over: it proves /api/v1/access-interfaces and
// /api/v1/access-attachments are both wired up behind auth.Middleware and
// authz.Middleware in the real production router, sharing
// RequireAccessTopologyRead/RequireAccessTopologyWrite (see
// authz.CanReadAccessTopology's doc comment for why one capability pair
// guards both resources). See each domain's own
// httpapi/authenticated_test.go for far more thorough versions of the
// same checks, scoped to that package.
func newRouterWithAccessTopology(tokens *auth.TokenIssuer, role auth.Role) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return api.NewRouter(api.Dependencies{
		Logger:                  logger,
		Version:                 "test",
		Commit:                  "test",
		AccessInterfaceHandler:  accessinterfacehttpapi.NewAccessInterfaceHandler(stubAccessInterfaceService{}),
		AccessAttachmentHandler: accessattachmenthttpapi.NewAccessAttachmentHandler(stubAccessAttachmentService{}),
		Tokens:                  tokens,
		Authz:                   authz.NewMiddleware(stubUserRepository{role: role}),
	})
}

const validAccessInterfaceBody = `{"pon_port_id":"11111111-1111-1111-1111-111111111111","technology":"GPON","name":"gpon-0/1/1","status":"Active"}`
const validAccessAttachmentBody = `{"access_interface_id":"11111111-1111-1111-1111-111111111111","service_equipment_id":"22222222-2222-2222-2222-222222222222"}`

func TestRouterRejectsUnauthenticatedAccessInterfaceRequests(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithAccessTopology(tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/access-interfaces/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestRouterRejectsUnauthenticatedAccessAttachmentRequests(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithAccessTopology(tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/access-attachments/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

// TestRouterViewerCanReadAccessInterfacesAndAccessAttachments is "apply
// the standard RBAC matrix", proven through the real, fully wired
// router, for both resources at once.
func TestRouterViewerCanReadAccessInterfacesAndAccessAttachments(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithAccessTopology(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	for _, path := range []string{"/api/v1/access-interfaces/", "/api/v1/access-attachments/"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("GET %s: status = %d, want %d; body: %s", path, rec.Code, http.StatusOK, rec.Body.String())
		}
	}
}

// TestRouterViewerCannotWriteAccessInterfacesOrAccessAttachments is
// "apply the standard RBAC matrix", proven through the real, fully wired
// router, for both resources at once.
func TestRouterViewerCannotWriteAccessInterfacesOrAccessAttachments(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithAccessTopology(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	cases := []struct {
		path string
		body string
	}{
		{"/api/v1/access-interfaces/", validAccessInterfaceBody},
		{"/api/v1/access-attachments/", validAccessAttachmentBody},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodPost, c.path, strings.NewReader(c.body))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("POST %s: status = %d, want %d; body: %s", c.path, rec.Code, http.StatusForbidden, rec.Body.String())
		}
	}
}

// TestRouterOperatorCanWriteAccessInterfacesAndAccessAttachments is
// "apply the standard RBAC matrix", proven through the real, fully wired
// router, for both resources at once.
func TestRouterOperatorCanWriteAccessInterfacesAndAccessAttachments(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithAccessTopology(tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	cases := []struct {
		path string
		body string
	}{
		{"/api/v1/access-interfaces/", validAccessInterfaceBody},
		{"/api/v1/access-attachments/", validAccessAttachmentBody},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodPost, c.path, strings.NewReader(c.body))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("POST %s: status = %d, want %d; body: %s", c.path, rec.Code, http.StatusCreated, rec.Body.String())
		}
	}
}

// TestRouterAdministratorCanWriteAccessInterfacesAndAccessAttachments is
// "apply the standard RBAC matrix", proven through the real, fully wired
// router, for both resources at once.
func TestRouterAdministratorCanWriteAccessInterfacesAndAccessAttachments(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithAccessTopology(tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	cases := []struct {
		path string
		body string
	}{
		{"/api/v1/access-interfaces/", validAccessInterfaceBody},
		{"/api/v1/access-attachments/", validAccessAttachmentBody},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodPost, c.path, strings.NewReader(c.body))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("POST %s: status = %d, want %d; body: %s", c.path, rec.Code, http.StatusCreated, rec.Body.String())
		}
	}
}

// stubServiceProfileService satisfies whatever interface
// serviceprofilehttpapi.ServiceProfileHandler needs structurally, the
// same technique stubServiceService and every other stub above uses.
type stubServiceProfileService struct{}

func (stubServiceProfileService) Get(context.Context, uuid.UUID) (serviceprofile.ServiceProfile, error) {
	return serviceprofile.ServiceProfile{}, apperror.NotFound("service profile not found")
}
func (stubServiceProfileService) List(context.Context) ([]serviceprofile.ServiceProfile, error) {
	return nil, nil
}
func (stubServiceProfileService) Create(_ context.Context, p serviceprofile.ServiceProfile) (serviceprofile.ServiceProfile, error) {
	return p, nil
}
func (stubServiceProfileService) Update(_ context.Context, p serviceprofile.ServiceProfile) (serviceprofile.ServiceProfile, error) {
	return p, nil
}
func (stubServiceProfileService) Delete(context.Context, uuid.UUID) error { return nil }

// newRouterWithServiceProfiles mirrors newRouterWithServices exactly,
// one domain over: it proves /api/v1/service-profiles is wired up
// behind auth.Middleware and authz.Middleware in the real production
// router, using its own dedicated
// RequireServiceProfilesRead/RequireServiceProfilesWrite (see
// authz.CanReadServiceProfiles's doc comment for why Service Profile
// does not share Catalog's/Product's capability pair, per this
// milestone's explicit instruction). See
// internal/serviceprofile/httpapi/authenticated_test.go for a far more
// thorough version of the same checks, scoped to that package.
func newRouterWithServiceProfiles(tokens *auth.TokenIssuer, role auth.Role) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return api.NewRouter(api.Dependencies{
		Logger:                logger,
		Version:               "test",
		Commit:                "test",
		ServiceProfileHandler: serviceprofilehttpapi.NewServiceProfileHandler(stubServiceProfileService{}),
		Tokens:                tokens,
		Authz:                 authz.NewMiddleware(stubUserRepository{role: role}),
	})
}

const validServiceProfileBody = `{"name":"Residential Internet","status":"Active"}`

func TestRouterRejectsUnauthenticatedServiceProfileRequests(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithServiceProfiles(tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/service-profiles/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

// TestRouterViewerCanReadServiceProfiles is "apply the standard RBAC
// matrix", proven through the real, fully wired router.
func TestRouterViewerCanReadServiceProfiles(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithServiceProfiles(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/service-profiles/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// TestRouterViewerCannotWriteServiceProfiles is "apply the standard RBAC
// matrix", proven through the real, fully wired router.
func TestRouterViewerCannotWriteServiceProfiles(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithServiceProfiles(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/service-profiles/", strings.NewReader(validServiceProfileBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

// TestRouterOperatorCanWriteServiceProfiles is "apply the standard RBAC
// matrix", proven through the real, fully wired router.
func TestRouterOperatorCanWriteServiceProfiles(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithServiceProfiles(tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/service-profiles/", strings.NewReader(validServiceProfileBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// TestRouterAdministratorCanWriteServiceProfiles is "apply the standard
// RBAC matrix", proven through the real, fully wired router.
func TestRouterAdministratorCanWriteServiceProfiles(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithServiceProfiles(tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/service-profiles/", strings.NewReader(validServiceProfileBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// stubDiagnosticsService satisfies whatever interface
// diagnosticshttpapi.DiagnosticsHandler needs structurally, the same
// technique every other stub above uses. It always returns a fixed
// placeholder Result, mirroring diagnostics.BasicONUCheck's own
// behavior closely enough for a wiring test — this file proves the
// route reaches a handler behind the right middleware, not what a real
// diagnostic run produces (see
// internal/diagnostics/httpapi/diagnostics_handler_test.go and
// internal/diagnostics's own tests for that).
type stubDiagnosticsService struct{}

func (stubDiagnosticsService) Run(context.Context, string, diagnostics.Request) (*diagnostics.Result, error) {
	return &diagnostics.Result{Name: diagnostics.BasicONUCheckName}, nil
}

// newRouterWithDiagnostics mirrors newRouterWithServiceProfiles exactly,
// one domain over: it proves /api/v1/diagnostics/basic-onu-check is
// wired up behind auth.Middleware and authz.Middleware in the real
// production router, using RequireDiagnostics — the single capability
// guarding this route (see authz.CanRunDiagnostics's doc comment for
// why there is no separate read/write split here). See
// internal/diagnostics/httpapi/authenticated_test.go for a far more
// thorough version of the same checks, scoped to that package.
func newRouterWithDiagnostics(tokens *auth.TokenIssuer, role auth.Role) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return api.NewRouter(api.Dependencies{
		Logger:             logger,
		Version:            "test",
		Commit:             "test",
		DiagnosticsHandler: diagnosticshttpapi.NewDiagnosticsHandler(stubDiagnosticsService{}),
		Tokens:             tokens,
		Authz:              authz.NewMiddleware(stubUserRepository{role: role}),
	})
}

const validDiagnosticsBody = `{"onuId":"11111111-1111-1111-1111-111111111111"}`

func TestRouterRejectsUnauthenticatedDiagnosticsRequests(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithDiagnostics(tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/diagnostics/basic-onu-check", strings.NewReader(validDiagnosticsBody)))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

// TestRouterViewerCannotRunDiagnostics is "apply the standard RBAC
// matrix", proven through the real, fully wired router — Viewer cannot
// run a diagnostic (see authz.CanRunDiagnostics's doc comment for why
// this is treated as an action, not a read).
func TestRouterViewerCannotRunDiagnostics(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithDiagnostics(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/diagnostics/basic-onu-check", strings.NewReader(validDiagnosticsBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

// TestRouterOperatorCanRunDiagnostics is "apply the standard RBAC
// matrix", proven through the real, fully wired router.
func TestRouterOperatorCanRunDiagnostics(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithDiagnostics(tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/diagnostics/basic-onu-check", strings.NewReader(validDiagnosticsBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// TestRouterAdministratorCanRunDiagnostics is "apply the standard RBAC
// matrix", proven through the real, fully wired router.
func TestRouterAdministratorCanRunDiagnostics(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithDiagnostics(tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/diagnostics/basic-onu-check", strings.NewReader(validDiagnosticsBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// stubAuthenticationService satisfies whatever interface
// authenticationhttpapi.AuthenticationHandler needs structurally, the
// same technique every other stub above uses.
type stubAuthenticationService struct{}

func (stubAuthenticationService) Get(context.Context, uuid.UUID) (authentication.Authentication, error) {
	return authentication.Authentication{}, apperror.NotFound("authentication method not found")
}
func (stubAuthenticationService) List(context.Context) ([]authentication.Authentication, error) {
	return nil, nil
}
func (stubAuthenticationService) Create(_ context.Context, a authentication.Authentication) (authentication.Authentication, error) {
	return a, nil
}
func (stubAuthenticationService) Update(_ context.Context, a authentication.Authentication) (authentication.Authentication, error) {
	return a, nil
}
func (stubAuthenticationService) Delete(context.Context, uuid.UUID) error { return nil }

// newRouterWithAuthentication mirrors newRouterWithServiceProfiles
// exactly, one domain over: it proves /api/v1/authentication-methods is
// wired up behind auth.Middleware and authz.Middleware in the real
// production router, using its own dedicated
// RequireAuthenticationRead/RequireAuthenticationWrite (see
// authz.CanReadAuthentication's doc comment for why Authentication does
// not share any other domain's capability pair, per goal 5's explicit
// naming). See internal/authentication/httpapi/authenticated_test.go for
// a far more thorough version of the same checks, scoped to that
// package.
func newRouterWithAuthentication(tokens *auth.TokenIssuer, role auth.Role) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return api.NewRouter(api.Dependencies{
		Logger:                logger,
		Version:               "test",
		Commit:                "test",
		AuthenticationHandler: authenticationhttpapi.NewAuthenticationHandler(stubAuthenticationService{}),
		Tokens:                tokens,
		Authz:                 authz.NewMiddleware(stubUserRepository{role: role}),
	})
}

const validAuthenticationBody = `{"name":"OLT Admin Credentials","authentication_type":"Password","username":"admin","password":"hunter2"}`

func TestRouterRejectsUnauthenticatedAuthenticationRequests(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithAuthentication(tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/authentication-methods/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

// TestRouterViewerCanReadAuthentication is "apply the standard RBAC
// matrix", proven through the real, fully wired router.
func TestRouterViewerCanReadAuthentication(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithAuthentication(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/authentication-methods/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// TestRouterViewerCannotWriteAuthentication is "apply the standard RBAC
// matrix", proven through the real, fully wired router.
func TestRouterViewerCannotWriteAuthentication(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithAuthentication(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/authentication-methods/", strings.NewReader(validAuthenticationBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

// TestRouterOperatorCanWriteAuthentication is "apply the standard RBAC
// matrix", proven through the real, fully wired router.
func TestRouterOperatorCanWriteAuthentication(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithAuthentication(tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/authentication-methods/", strings.NewReader(validAuthenticationBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// TestRouterAdministratorCanWriteAuthentication is "apply the standard
// RBAC matrix", proven through the real, fully wired router.
func TestRouterAdministratorCanWriteAuthentication(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithAuthentication(tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/authentication-methods/", strings.NewReader(validAuthenticationBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// TestRouterAuthenticationResponseNeverEchoesSecrets proves the real,
// fully wired router applies internal/authentication/httpapi's
// never-echo-secrets response shape, not just the handler tested in
// isolation (see
// internal/authentication/httpapi/authentication_handler_test.go's
// TestAuthenticationHandlerCreateNeverEchoesSecrets for that narrower
// check) — a plaintext password submitted in the request body must never
// appear anywhere in the response body reachable through the production
// router.
func TestRouterAuthenticationResponseNeverEchoesSecrets(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithAuthentication(tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/authentication-methods/", strings.NewReader(validAuthenticationBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "hunter2") {
		t.Fatalf("response body contains the plaintext password: %s", rec.Body.String())
	}
}

// stubConnectionProfileService satisfies whatever interface
// connectionprofilehttpapi.ConnectionProfileHandler needs structurally,
// the same technique every other stub above uses.
type stubConnectionProfileService struct{}

func (stubConnectionProfileService) Get(context.Context, uuid.UUID) (connectionprofile.ConnectionProfile, error) {
	return connectionprofile.ConnectionProfile{}, apperror.NotFound("connection profile not found")
}
func (stubConnectionProfileService) List(context.Context) ([]connectionprofile.ConnectionProfile, error) {
	return nil, nil
}
func (stubConnectionProfileService) Create(_ context.Context, p connectionprofile.ConnectionProfile) (connectionprofile.ConnectionProfile, error) {
	return p, nil
}
func (stubConnectionProfileService) Update(_ context.Context, p connectionprofile.ConnectionProfile) (connectionprofile.ConnectionProfile, error) {
	return p, nil
}
func (stubConnectionProfileService) Delete(context.Context, uuid.UUID) error { return nil }

// newRouterWithConnectionProfiles mirrors newRouterWithAuthentication
// exactly, one domain over: it proves /api/v1/connection-profiles is
// wired up behind auth.Middleware and authz.Middleware in the real
// production router, using its own dedicated
// RequireConnectionProfilesRead/RequireConnectionProfilesWrite (see
// authz.CanReadConnectionProfiles's doc comment for why Connection
// Profile does not share Authentication's capability pair, per goal 5's
// explicit naming). See
// internal/connectionprofile/httpapi/authenticated_test.go for a far
// more thorough version of the same checks, scoped to that package.
func newRouterWithConnectionProfiles(tokens *auth.TokenIssuer, role auth.Role) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return api.NewRouter(api.Dependencies{
		Logger:                   logger,
		Version:                  "test",
		Commit:                   "test",
		ConnectionProfileHandler: connectionprofilehttpapi.NewConnectionProfileHandler(stubConnectionProfileService{}),
		Tokens:                   tokens,
		Authz:                    authz.NewMiddleware(stubUserRepository{role: role}),
	})
}

const validConnectionProfileBody = `{"name":"Default SSH Profile","protocol":"SSH","port":22,"timeout":"30s","host_key_policy":"Strict"}`

func TestRouterRejectsUnauthenticatedConnectionProfileRequests(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithConnectionProfiles(tokens, auth.RoleAdministrator)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/connection-profiles/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

// TestRouterViewerCanReadConnectionProfiles is "apply the standard RBAC
// matrix", proven through the real, fully wired router.
func TestRouterViewerCanReadConnectionProfiles(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithConnectionProfiles(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/connection-profiles/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// TestRouterViewerCannotWriteConnectionProfiles is "apply the standard
// RBAC matrix", proven through the real, fully wired router.
func TestRouterViewerCannotWriteConnectionProfiles(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithConnectionProfiles(tokens, auth.RoleViewer)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/connection-profiles/", strings.NewReader(validConnectionProfileBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

// TestRouterOperatorCanWriteConnectionProfiles is "apply the standard
// RBAC matrix", proven through the real, fully wired router.
func TestRouterOperatorCanWriteConnectionProfiles(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithConnectionProfiles(tokens, auth.RoleOperator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/connection-profiles/", strings.NewReader(validConnectionProfileBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// TestRouterAdministratorCanWriteConnectionProfiles is "apply the
// standard RBAC matrix", proven through the real, fully wired router.
func TestRouterAdministratorCanWriteConnectionProfiles(t *testing.T) {
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	router := newRouterWithConnectionProfiles(tokens, auth.RoleAdministrator)
	token := mustIssueToken(t, tokens)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/connection-profiles/", strings.NewReader(validConnectionProfileBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// stubAuthService satisfies authhttpapi's unexported authService
// interface structurally, the same technique stubSiteService above uses.
type stubAuthService struct{}

func (stubAuthService) Authenticate(context.Context, string, string) (string, error) {
	return "stub.jwt.token", nil
}

// TestRouterLoginEndpointIsReachableWithoutAuthentication proves
// /api/v1/auth/login is wired up outside the auth.Middleware group in the
// real production router — a caller has no token yet at the point they
// are trying to obtain one, so this route must never require one. See
// internal/auth/httpapi's own tests for thorough coverage of the login
// handler's behavior in isolation; this test only proves the wiring.
func TestRouterLoginEndpointIsReachableWithoutAuthentication(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := api.NewRouter(api.Dependencies{
		Logger:       logger,
		Version:      "test",
		Commit:       "test",
		LoginHandler: authhttpapi.NewLoginHandler(stubAuthService{}, 30*time.Minute),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{"email":"admin@example.com","password":"whatever"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}
