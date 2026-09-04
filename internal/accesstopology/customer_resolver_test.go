package accesstopology

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/location"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/service"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

type fakeCustomerLocationsGetter struct {
	locations []location.Location
	err       error
	gotID     uuid.UUID
}

func (f *fakeCustomerLocationsGetter) ListByCustomerID(_ context.Context, customerID uuid.UUID) ([]location.Location, error) {
	f.gotID = customerID
	if f.err != nil {
		return nil, f.err
	}
	return f.locations, nil
}

type fakeLocationServicesGetter struct {
	byLocationID map[uuid.UUID][]service.Service
	err          error
	gotIDs       []uuid.UUID
}

func (f *fakeLocationServicesGetter) ListByLocationID(_ context.Context, locationID uuid.UUID) ([]service.Service, error) {
	f.gotIDs = append(f.gotIDs, locationID)
	if f.err != nil {
		return nil, f.err
	}
	return f.byLocationID[locationID], nil
}

type fakeActiveServiceEquipmentGetter struct {
	byServiceID map[uuid.UUID][]serviceequipment.ServiceEquipment
	err         error
	gotIDs      []uuid.UUID
}

func (f *fakeActiveServiceEquipmentGetter) ListActiveByServiceID(_ context.Context, serviceID uuid.UUID) ([]serviceequipment.ServiceEquipment, error) {
	f.gotIDs = append(f.gotIDs, serviceID)
	if f.err != nil {
		return nil, f.err
	}
	return f.byServiceID[serviceID], nil
}

// fakeEquipmentLocator delegates to locateFunc so each test can decide,
// per call, whether a given ServiceEquipmentID resolves, is not found
// (the expected "nothing attached yet" case), or fails outright.
type fakeEquipmentLocator struct {
	locateFunc func(serviceEquipmentID uuid.UUID) (Location, error)
	calls      []uuid.UUID
}

func (f *fakeEquipmentLocator) Locate(_ context.Context, serviceEquipmentID uuid.UUID) (Location, error) {
	f.calls = append(f.calls, serviceEquipmentID)
	return f.locateFunc(serviceEquipmentID)
}

func TestLocateForCustomerResolvesFullFanOut(t *testing.T) {
	customerID := uuid.New()
	loc := location.Location{ID: uuid.New(), CustomerID: customerID}
	svc := service.Service{ID: uuid.New(), LocationID: loc.ID}
	eq := serviceequipment.ServiceEquipment{ID: uuid.New(), ServiceID: svc.ID}
	want := Location{OLTID: uuid.New(), Interface: "xgs/1/1"}

	locations := &fakeCustomerLocationsGetter{locations: []location.Location{loc}}
	services := &fakeLocationServicesGetter{byLocationID: map[uuid.UUID][]service.Service{loc.ID: {svc}}}
	equipment := &fakeActiveServiceEquipmentGetter{byServiceID: map[uuid.UUID][]serviceequipment.ServiceEquipment{svc.ID: {eq}}}
	locator := &fakeEquipmentLocator{locateFunc: func(uuid.UUID) (Location, error) { return want, nil }}

	r := NewCustomerResolver(locations, services, equipment, locator)

	got, err := r.LocateForCustomer(context.Background(), customerID)
	if err != nil {
		t.Fatalf("LocateForCustomer() = %v", err)
	}

	if locations.gotID != customerID {
		t.Errorf("locations.ListByCustomerID called with %v, want %v", locations.gotID, customerID)
	}
	if len(got) != 1 {
		t.Fatalf("len(LocateForCustomer()) = %d, want 1; got %+v", len(got), got)
	}
	if got[0].ServiceEquipmentID != eq.ID || got[0].Location != want {
		t.Errorf("LocateForCustomer()[0] = %+v, want {ServiceEquipmentID:%v Location:%+v}", got[0], eq.ID, want)
	}
}

// TestLocateForCustomerSkipsEquipmentWithNoActiveAttachment proves
// equipment the locator reports as not-found is silently omitted from
// the result rather than aborting the whole resolution — the expected,
// common case for equipment that exists but has nothing attached yet.
func TestLocateForCustomerSkipsEquipmentWithNoActiveAttachment(t *testing.T) {
	customerID := uuid.New()
	loc := location.Location{ID: uuid.New(), CustomerID: customerID}
	svc := service.Service{ID: uuid.New(), LocationID: loc.ID}
	unattached := serviceequipment.ServiceEquipment{ID: uuid.New(), ServiceID: svc.ID}
	attached := serviceequipment.ServiceEquipment{ID: uuid.New(), ServiceID: svc.ID}
	want := Location{OLTID: uuid.New(), Interface: "xgs/1/2"}

	locations := &fakeCustomerLocationsGetter{locations: []location.Location{loc}}
	services := &fakeLocationServicesGetter{byLocationID: map[uuid.UUID][]service.Service{loc.ID: {svc}}}
	equipment := &fakeActiveServiceEquipmentGetter{
		byServiceID: map[uuid.UUID][]serviceequipment.ServiceEquipment{svc.ID: {unattached, attached}},
	}
	locator := &fakeEquipmentLocator{locateFunc: func(id uuid.UUID) (Location, error) {
		if id == unattached.ID {
			return Location{}, apperror.NotFound("no active attachment")
		}
		return want, nil
	}}

	r := NewCustomerResolver(locations, services, equipment, locator)

	got, err := r.LocateForCustomer(context.Background(), customerID)
	if err != nil {
		t.Fatalf("LocateForCustomer() = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(LocateForCustomer()) = %d, want 1 (the unattached equipment must be skipped); got %+v", len(got), got)
	}
	if got[0].ServiceEquipmentID != attached.ID {
		t.Errorf("ServiceEquipmentID = %v, want %v (the attached one)", got[0].ServiceEquipmentID, attached.ID)
	}
}

// TestLocateForCustomerAbortsOnGenuineLocatorError proves a non-not-found
// error from the locator aborts the whole resolution rather than being
// silently skipped like the not-found case above.
func TestLocateForCustomerAbortsOnGenuineLocatorError(t *testing.T) {
	customerID := uuid.New()
	loc := location.Location{ID: uuid.New(), CustomerID: customerID}
	svc := service.Service{ID: uuid.New(), LocationID: loc.ID}
	eq := serviceequipment.ServiceEquipment{ID: uuid.New(), ServiceID: svc.ID}
	locatorErr := errors.New("database unavailable")

	locations := &fakeCustomerLocationsGetter{locations: []location.Location{loc}}
	services := &fakeLocationServicesGetter{byLocationID: map[uuid.UUID][]service.Service{loc.ID: {svc}}}
	equipment := &fakeActiveServiceEquipmentGetter{byServiceID: map[uuid.UUID][]serviceequipment.ServiceEquipment{svc.ID: {eq}}}
	locator := &fakeEquipmentLocator{locateFunc: func(uuid.UUID) (Location, error) { return Location{}, locatorErr }}

	r := NewCustomerResolver(locations, services, equipment, locator)

	_, err := r.LocateForCustomer(context.Background(), customerID)
	if !errors.Is(err, locatorErr) {
		t.Fatalf("LocateForCustomer() error = %v, want it to wrap %v", err, locatorErr)
	}
}

func TestLocateForCustomerReturnsEmptyWhenNoLocations(t *testing.T) {
	locations := &fakeCustomerLocationsGetter{locations: nil}
	services := &fakeLocationServicesGetter{}
	equipment := &fakeActiveServiceEquipmentGetter{}
	locator := &fakeEquipmentLocator{locateFunc: func(uuid.UUID) (Location, error) {
		t.Fatal("locator.Locate was called; it must never be reached for a customer with no locations")
		return Location{}, nil
	}}

	r := NewCustomerResolver(locations, services, equipment, locator)

	got, err := r.LocateForCustomer(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("LocateForCustomer() = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("LocateForCustomer() = %+v, want empty", got)
	}
	if len(services.gotIDs) != 0 {
		t.Error("services.ListByLocationID was called; it must never be reached for a customer with no locations")
	}
}

func TestLocateForCustomerPropagatesLocationsError(t *testing.T) {
	wantErr := errors.New("boom")
	locations := &fakeCustomerLocationsGetter{err: wantErr}
	services := &fakeLocationServicesGetter{}
	equipment := &fakeActiveServiceEquipmentGetter{}
	locator := &fakeEquipmentLocator{locateFunc: func(uuid.UUID) (Location, error) { return Location{}, nil }}

	r := NewCustomerResolver(locations, services, equipment, locator)

	_, err := r.LocateForCustomer(context.Background(), uuid.New())
	if !errors.Is(err, wantErr) {
		t.Fatalf("LocateForCustomer() error = %v, want it to wrap %v", err, wantErr)
	}
}

func TestLocateForCustomerPropagatesServicesError(t *testing.T) {
	loc := location.Location{ID: uuid.New()}
	wantErr := errors.New("boom")

	locations := &fakeCustomerLocationsGetter{locations: []location.Location{loc}}
	services := &fakeLocationServicesGetter{err: wantErr}
	equipment := &fakeActiveServiceEquipmentGetter{}
	locator := &fakeEquipmentLocator{locateFunc: func(uuid.UUID) (Location, error) { return Location{}, nil }}

	r := NewCustomerResolver(locations, services, equipment, locator)

	_, err := r.LocateForCustomer(context.Background(), uuid.New())
	if !errors.Is(err, wantErr) {
		t.Fatalf("LocateForCustomer() error = %v, want it to wrap %v", err, wantErr)
	}
}

func TestLocateForCustomerPropagatesEquipmentError(t *testing.T) {
	loc := location.Location{ID: uuid.New()}
	svc := service.Service{ID: uuid.New(), LocationID: loc.ID}
	wantErr := errors.New("boom")

	locations := &fakeCustomerLocationsGetter{locations: []location.Location{loc}}
	services := &fakeLocationServicesGetter{byLocationID: map[uuid.UUID][]service.Service{loc.ID: {svc}}}
	equipment := &fakeActiveServiceEquipmentGetter{err: wantErr}
	locator := &fakeEquipmentLocator{locateFunc: func(uuid.UUID) (Location, error) { return Location{}, nil }}

	r := NewCustomerResolver(locations, services, equipment, locator)

	_, err := r.LocateForCustomer(context.Background(), uuid.New())
	if !errors.Is(err, wantErr) {
		t.Fatalf("LocateForCustomer() error = %v, want it to wrap %v", err, wantErr)
	}
}

// TestLocateForCustomerHandlesMultipleLocationsAndServices proves the
// fan-out actually fans out across more than one Location and Service,
// not just the single-item happy path every other test here uses.
func TestLocateForCustomerHandlesMultipleLocationsAndServices(t *testing.T) {
	customerID := uuid.New()
	locA := location.Location{ID: uuid.New(), CustomerID: customerID}
	locB := location.Location{ID: uuid.New(), CustomerID: customerID}
	svcA := service.Service{ID: uuid.New(), LocationID: locA.ID}
	svcB := service.Service{ID: uuid.New(), LocationID: locB.ID}
	eqA := serviceequipment.ServiceEquipment{ID: uuid.New(), ServiceID: svcA.ID}
	eqB := serviceequipment.ServiceEquipment{ID: uuid.New(), ServiceID: svcB.ID}

	locations := &fakeCustomerLocationsGetter{locations: []location.Location{locA, locB}}
	services := &fakeLocationServicesGetter{byLocationID: map[uuid.UUID][]service.Service{
		locA.ID: {svcA}, locB.ID: {svcB},
	}}
	equipment := &fakeActiveServiceEquipmentGetter{byServiceID: map[uuid.UUID][]serviceequipment.ServiceEquipment{
		svcA.ID: {eqA}, svcB.ID: {eqB},
	}}
	locator := &fakeEquipmentLocator{locateFunc: func(id uuid.UUID) (Location, error) {
		return Location{OLTID: uuid.New(), Interface: id.String()}, nil
	}}

	r := NewCustomerResolver(locations, services, equipment, locator)

	got, err := r.LocateForCustomer(context.Background(), customerID)
	if err != nil {
		t.Fatalf("LocateForCustomer() = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(LocateForCustomer()) = %d, want 2; got %+v", len(got), got)
	}

	gotIDs := map[uuid.UUID]bool{got[0].ServiceEquipmentID: true, got[1].ServiceEquipmentID: true}
	if !gotIDs[eqA.ID] || !gotIDs[eqB.ID] {
		t.Errorf("LocateForCustomer() = %+v, want entries for both %v and %v", got, eqA.ID, eqB.ID)
	}
}
