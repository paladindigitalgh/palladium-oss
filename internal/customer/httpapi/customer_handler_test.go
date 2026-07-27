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

	"github.com/paladindigitalgh/palladium-oss/internal/customer"
	"github.com/paladindigitalgh/palladium-oss/internal/customer/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeCustomerService is the seam httpapi.CustomerHandler depends on (see
// its unexported customerService interface in customer_handler.go). It
// lets these tests exercise HTTP-only concerns — status codes, JSON
// shapes, routing, error translation — without a real service,
// repository, or database; internal/customer/service and
// internal/customer/postgres each have their own tests for the layers
// below this one.
type fakeCustomerService struct {
	customers map[uuid.UUID]customer.Customer
	err       error // if set, every method returns this error instead
}

func newFakeCustomerService(customers ...customer.Customer) *fakeCustomerService {
	f := &fakeCustomerService{customers: make(map[uuid.UUID]customer.Customer)}
	for _, c := range customers {
		f.customers[c.ID] = c
	}
	return f
}

func (f *fakeCustomerService) Get(_ context.Context, id uuid.UUID) (customer.Customer, error) {
	if f.err != nil {
		return customer.Customer{}, f.err
	}
	c, ok := f.customers[id]
	if !ok {
		return customer.Customer{}, apperror.NotFound("customer not found")
	}
	return c, nil
}

func (f *fakeCustomerService) List(context.Context) ([]customer.Customer, error) {
	if f.err != nil {
		return nil, f.err
	}
	customers := make([]customer.Customer, 0, len(f.customers))
	for _, c := range f.customers {
		customers = append(customers, c)
	}
	return customers, nil
}

func (f *fakeCustomerService) Create(_ context.Context, c customer.Customer) (customer.Customer, error) {
	if f.err != nil {
		return customer.Customer{}, f.err
	}
	c.ID = uuid.New()
	c.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c.UpdatedAt = c.CreatedAt
	f.customers[c.ID] = c
	return c, nil
}

func (f *fakeCustomerService) Update(_ context.Context, c customer.Customer) (customer.Customer, error) {
	if f.err != nil {
		return customer.Customer{}, f.err
	}
	if _, ok := f.customers[c.ID]; !ok {
		return customer.Customer{}, apperror.NotFound("customer not found")
	}
	f.customers[c.ID] = c
	return c, nil
}

func (f *fakeCustomerService) Delete(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.customers[id]; !ok {
		return apperror.NotFound("customer not found")
	}
	delete(f.customers, id)
	return nil
}

// newTestRouter mounts a CustomerHandler backed by svc on a real
// chi.Router, so tests that need a URL path parameter (Get/Update/
// Delete's {id}) get one populated the same way production code does,
// rather than faking chi's route context by hand.
func newTestRouter(svc *fakeCustomerService) http.Handler {
	handler := httpapi.NewCustomerHandler(svc)

	r := chi.NewRouter()
	r.Post("/customers", handler.Create)
	r.Get("/customers", handler.List)
	r.Get("/customers/{id}", handler.Get)
	r.Put("/customers/{id}", handler.Update)
	r.Delete("/customers/{id}", handler.Delete)
	return r
}

const validBody = `{"name":"Jane Doe","customer_type":"Residential","status":"Active"}`

func TestCustomerHandlerCreate(t *testing.T) {
	router := newTestRouter(newFakeCustomerService())

	req := httptest.NewRequest(http.MethodPost, "/customers", strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		CustomerType string `json:"customer_type"`
		Status       string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID == "" {
		t.Error("response did not include an id")
	}
	if body.Name != "Jane Doe" || body.CustomerType != "Residential" || body.Status != "Active" {
		t.Errorf("body = %+v, want Name=Jane Doe CustomerType=Residential Status=Active", body)
	}
}

func TestCustomerHandlerCreateRejectsMalformedJSON(t *testing.T) {
	router := newTestRouter(newFakeCustomerService())

	req := httptest.NewRequest(http.MethodPost, "/customers", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCustomerHandlerCreatePropagatesServiceValidationError(t *testing.T) {
	svc := newFakeCustomerService()
	svc.err = apperror.Invalid("name: is required")
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/customers", strings.NewReader(`{"name":""}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestCustomerHandlerList(t *testing.T) {
	a := customer.Customer{ID: uuid.New(), Name: "A", CustomerType: customer.CustomerTypeResidential, Status: customer.CustomerStatusActive}
	b := customer.Customer{ID: uuid.New(), Name: "B", CustomerType: customer.CustomerTypeBusiness, Status: customer.CustomerStatusActive}
	router := newTestRouter(newFakeCustomerService(a, b))

	req := httptest.NewRequest(http.MethodGet, "/customers", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		Customers []struct {
			ID string `json:"id"`
		} `json:"customers"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Customers) != 2 {
		t.Fatalf("len(customers) = %d, want 2", len(body.Customers))
	}
}

func TestCustomerHandlerGet(t *testing.T) {
	c := customer.Customer{ID: uuid.New(), Name: "Jane Doe", CustomerType: customer.CustomerTypeResidential, Status: customer.CustomerStatusActive}
	router := newTestRouter(newFakeCustomerService(c))

	req := httptest.NewRequest(http.MethodGet, "/customers/"+c.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestCustomerHandlerGetNotFound(t *testing.T) {
	router := newTestRouter(newFakeCustomerService())

	req := httptest.NewRequest(http.MethodGet, "/customers/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestCustomerHandlerGetRejectsMalformedID(t *testing.T) {
	router := newTestRouter(newFakeCustomerService())

	req := httptest.NewRequest(http.MethodGet, "/customers/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCustomerHandlerUpdate(t *testing.T) {
	c := customer.Customer{ID: uuid.New(), Name: "Old Name", CustomerType: customer.CustomerTypeResidential, Status: customer.CustomerStatusActive}
	router := newTestRouter(newFakeCustomerService(c))

	req := httptest.NewRequest(http.MethodPut, "/customers/"+c.ID.String(),
		strings.NewReader(`{"name":"New Name","customer_type":"Business","status":"Inactive"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Name         string `json:"name"`
		CustomerType string `json:"customer_type"`
		Status       string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Name != "New Name" || body.CustomerType != "Business" || body.Status != "Inactive" {
		t.Errorf("body = %+v, want Name=New Name CustomerType=Business Status=Inactive", body)
	}
}

func TestCustomerHandlerUpdateNotFound(t *testing.T) {
	router := newTestRouter(newFakeCustomerService())

	req := httptest.NewRequest(http.MethodPut, "/customers/"+uuid.New().String(), strings.NewReader(validBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestCustomerHandlerDelete(t *testing.T) {
	c := customer.Customer{ID: uuid.New(), Name: "Temporary", CustomerType: customer.CustomerTypeResidential, Status: customer.CustomerStatusActive}
	router := newTestRouter(newFakeCustomerService(c))

	req := httptest.NewRequest(http.MethodDelete, "/customers/"+c.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204 No Content", rec.Body.String())
	}
}

func TestCustomerHandlerDeleteNotFound(t *testing.T) {
	router := newTestRouter(newFakeCustomerService())

	req := httptest.NewRequest(http.MethodDelete, "/customers/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
