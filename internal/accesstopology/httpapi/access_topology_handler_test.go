package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accesstopology"
	"github.com/paladindigitalgh/palladium-oss/internal/accesstopology/httpapi"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeCustomerResolver is the seam httpapi.AccessTopologyHandler depends
// on (see its unexported customerResolver interface). It lets these
// tests exercise HTTP-only concerns — routing, status codes, JSON
// shapes, error translation — without a real resolver or repositories
// involved; internal/accesstopology has its own tests for the resolution
// logic itself.
type fakeCustomerResolver struct {
	locations []accesstopology.CustomerLocation
	err       error
	gotID     uuid.UUID
}

func (f *fakeCustomerResolver) LocateForCustomer(_ context.Context, customerID uuid.UUID) ([]accesstopology.CustomerLocation, error) {
	f.gotID = customerID
	if f.err != nil {
		return nil, f.err
	}
	return f.locations, nil
}

func newTestRouter(resolver *fakeCustomerResolver) http.Handler {
	handler := httpapi.NewAccessTopologyHandler(resolver)

	r := chi.NewRouter()
	r.Get("/diagnostics/customers/{customerId}/equipment-locations", handler.ListCustomerEquipmentLocations)
	return r
}

func TestListCustomerEquipmentLocationsReturnsResolvedLocations(t *testing.T) {
	customerID := uuid.New()
	serviceEquipmentID := uuid.New()
	oltID := uuid.New()

	resolver := &fakeCustomerResolver{locations: []accesstopology.CustomerLocation{
		{ServiceEquipmentID: serviceEquipmentID, Location: accesstopology.Location{OLTID: oltID, Interface: "xgs/1/1"}},
	}}
	router := newTestRouter(resolver)

	req := httptest.NewRequest(http.MethodGet, "/diagnostics/customers/"+customerID.String()+"/equipment-locations", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if resolver.gotID != customerID {
		t.Errorf("customerID passed to resolver = %v, want %v", resolver.gotID, customerID)
	}

	var body struct {
		Locations []struct {
			ServiceEquipmentID uuid.UUID `json:"service_equipment_id"`
			OLTID              uuid.UUID `json:"olt_id"`
			Interface          string    `json:"interface"`
		} `json:"locations"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Locations) != 1 {
		t.Fatalf("len(locations) = %d, want 1", len(body.Locations))
	}
	if body.Locations[0].ServiceEquipmentID != serviceEquipmentID {
		t.Errorf("service_equipment_id = %v, want %v", body.Locations[0].ServiceEquipmentID, serviceEquipmentID)
	}
	if body.Locations[0].OLTID != oltID {
		t.Errorf("olt_id = %v, want %v", body.Locations[0].OLTID, oltID)
	}
	if body.Locations[0].Interface != "xgs/1/1" {
		t.Errorf("interface = %q, want %q", body.Locations[0].Interface, "xgs/1/1")
	}
}

func TestListCustomerEquipmentLocationsReturnsEmptyListNotError(t *testing.T) {
	resolver := &fakeCustomerResolver{locations: nil}
	router := newTestRouter(resolver)

	req := httptest.NewRequest(http.MethodGet, "/diagnostics/customers/"+uuid.New().String()+"/equipment-locations", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Locations []any `json:"locations"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Locations) != 0 {
		t.Errorf("locations = %v, want empty", body.Locations)
	}
}

func TestListCustomerEquipmentLocationsRejectsInvalidCustomerID(t *testing.T) {
	resolver := &fakeCustomerResolver{}
	router := newTestRouter(resolver)

	req := httptest.NewRequest(http.MethodGet, "/diagnostics/customers/not-a-uuid/equipment-locations", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestListCustomerEquipmentLocationsPropagatesResolverError(t *testing.T) {
	resolver := &fakeCustomerResolver{err: apperror.NotFound("customer not found")}
	router := newTestRouter(resolver)

	req := httptest.NewRequest(http.MethodGet, "/diagnostics/customers/"+uuid.New().String()+"/equipment-locations", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}
