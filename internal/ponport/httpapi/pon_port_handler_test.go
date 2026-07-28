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
	"github.com/paladindigitalgh/palladium-oss/internal/ponport"
	"github.com/paladindigitalgh/palladium-oss/internal/ponport/httpapi"
)

// fakePONPortService is the seam httpapi.PONPortHandler depends on (see
// its unexported ponPortService interface in pon_port_handler.go). It
// lets these tests exercise HTTP-only concerns — status codes, JSON
// shapes, routing, error translation — without a real service,
// repository, or database; internal/ponport/service and
// internal/ponport/postgres each have their own tests for the layers
// below this one.
type fakePONPortService struct {
	ports map[uuid.UUID]ponport.PONPort
	err   error // if set, every method returns this error instead
}

func newFakePONPortService(ports ...ponport.PONPort) *fakePONPortService {
	f := &fakePONPortService{ports: make(map[uuid.UUID]ponport.PONPort)}
	for _, p := range ports {
		f.ports[p.ID] = p
	}
	return f
}

func (f *fakePONPortService) Get(_ context.Context, id uuid.UUID) (ponport.PONPort, error) {
	if f.err != nil {
		return ponport.PONPort{}, f.err
	}
	p, ok := f.ports[id]
	if !ok {
		return ponport.PONPort{}, apperror.NotFound("pon port not found")
	}
	return p, nil
}

func (f *fakePONPortService) List(context.Context) ([]ponport.PONPort, error) {
	if f.err != nil {
		return nil, f.err
	}
	ports := make([]ponport.PONPort, 0, len(f.ports))
	for _, p := range f.ports {
		ports = append(ports, p)
	}
	return ports, nil
}

func (f *fakePONPortService) Create(_ context.Context, p ponport.PONPort) (ponport.PONPort, error) {
	if f.err != nil {
		return ponport.PONPort{}, f.err
	}
	p.ID = uuid.New()
	p.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p.UpdatedAt = p.CreatedAt
	f.ports[p.ID] = p
	return p, nil
}

func (f *fakePONPortService) Update(_ context.Context, p ponport.PONPort) (ponport.PONPort, error) {
	if f.err != nil {
		return ponport.PONPort{}, f.err
	}
	if _, ok := f.ports[p.ID]; !ok {
		return ponport.PONPort{}, apperror.NotFound("pon port not found")
	}
	f.ports[p.ID] = p
	return p, nil
}

func (f *fakePONPortService) Delete(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.ports[id]; !ok {
		return apperror.NotFound("pon port not found")
	}
	delete(f.ports, id)
	return nil
}

// newTestRouter mounts a PONPortHandler backed by svc on a real
// chi.Router, so tests that need a URL path parameter (Get/Update/
// Delete's {id}) get one populated the same way production code does,
// rather than faking chi's route context by hand.
func newTestRouter(svc *fakePONPortService) http.Handler {
	handler := httpapi.NewPONPortHandler(svc)

	r := chi.NewRouter()
	r.Post("/pon-ports", handler.Create)
	r.Get("/pon-ports", handler.List)
	r.Get("/pon-ports/{id}", handler.Get)
	r.Put("/pon-ports/{id}", handler.Update)
	r.Delete("/pon-ports/{id}", handler.Delete)
	return r
}

const validBody = `{"olt_id":"11111111-1111-1111-1111-111111111111","port_number":1}`

func TestPONPortHandlerCreate(t *testing.T) {
	router := newTestRouter(newFakePONPortService())

	req := httptest.NewRequest(http.MethodPost, "/pon-ports", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		ID         string `json:"id"`
		OLTID      string `json:"olt_id"`
		PortNumber int    `json:"port_number"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID == "" {
		t.Error("response did not include an id")
	}
	if body.OLTID != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("olt_id = %q, want %q", body.OLTID, "11111111-1111-1111-1111-111111111111")
	}
	if body.PortNumber != 1 {
		t.Errorf("port_number = %d, want %d", body.PortNumber, 1)
	}
}

func TestPONPortHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newTestRouter(newFakePONPortService())

	req := httptest.NewRequest(http.MethodPost, "/pon-ports", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestPONPortHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := newFakePONPortService()
	svc.err = apperror.Invalid("port_number: is required")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/pon-ports", strings.NewReader(`{"port_number":0}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestPONPortHandlerCreatePropagatesConflictOnUnknownOLT(t *testing.T) {
	svc := newFakePONPortService()
	svc.err = apperror.Conflict("create pon port: violates a foreign key relationship")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/pon-ports", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestPONPortHandlerList(t *testing.T) {
	a := ponport.PONPort{ID: uuid.New(), OLTID: uuid.New(), PortNumber: 1}
	b := ponport.PONPort{ID: uuid.New(), OLTID: uuid.New(), PortNumber: 2}
	router := newTestRouter(newFakePONPortService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/pon-ports", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		PONPorts []struct {
			ID string `json:"id"`
		} `json:"pon_ports"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.PONPorts) != 2 {
		t.Fatalf("len(pon_ports) = %d, want 2", len(body.PONPorts))
	}
}

func TestPONPortHandlerGet(t *testing.T) {
	p := ponport.PONPort{ID: uuid.New(), OLTID: uuid.New(), PortNumber: 1}
	router := newTestRouter(newFakePONPortService(p))

	req := httptest.NewRequest(http.MethodGet, "/pon-ports/"+p.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestPONPortHandlerGetNotFound(t *testing.T) {
	router := newTestRouter(newFakePONPortService())

	req := httptest.NewRequest(http.MethodGet, "/pon-ports/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestPONPortHandlerGetRejectsMalformedID(t *testing.T) {
	router := newTestRouter(newFakePONPortService())

	req := httptest.NewRequest(http.MethodGet, "/pon-ports/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestPONPortHandlerUpdate(t *testing.T) {
	p := ponport.PONPort{ID: uuid.New(), OLTID: uuid.New(), PortNumber: 1}
	router := newTestRouter(newFakePONPortService(p))

	req := httptest.NewRequest(http.MethodPut, "/pon-ports/"+p.ID.String(),
		strings.NewReader(`{"olt_id":"`+p.OLTID.String()+`","port_number":2}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		PortNumber int `json:"port_number"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.PortNumber != 2 {
		t.Errorf("port_number = %d, want %d", body.PortNumber, 2)
	}
}

func TestPONPortHandlerUpdateNotFound(t *testing.T) {
	router := newTestRouter(newFakePONPortService())

	req := httptest.NewRequest(http.MethodPut, "/pon-ports/"+uuid.New().String(), strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestPONPortHandlerDelete(t *testing.T) {
	p := ponport.PONPort{ID: uuid.New(), OLTID: uuid.New(), PortNumber: 1}
	router := newTestRouter(newFakePONPortService(p))

	req := httptest.NewRequest(http.MethodDelete, "/pon-ports/"+p.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204 No Content", rec.Body.String())
	}
}

func TestPONPortHandlerDeleteNotFound(t *testing.T) {
	router := newTestRouter(newFakePONPortService())

	req := httptest.NewRequest(http.MethodDelete, "/pon-ports/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
