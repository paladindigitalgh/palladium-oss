package removal

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessattachment"
	"github.com/paladindigitalgh/palladium-oss/internal/customer"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/location"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/service"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// fakeCustomers is an in-memory customerRepository.
type fakeCustomers struct {
	byID        map[uuid.UUID]customer.Customer
	updateCalls []customer.Customer
}

func (f *fakeCustomers) Get(_ context.Context, id uuid.UUID) (customer.Customer, error) {
	c, ok := f.byID[id]
	if !ok {
		return customer.Customer{}, errors.New("not found")
	}
	return c, nil
}

func (f *fakeCustomers) Update(_ context.Context, c customer.Customer) (customer.Customer, error) {
	f.updateCalls = append(f.updateCalls, c)
	f.byID[c.ID] = c
	return c, nil
}

// fakeLocations is an in-memory locationRepository.
type fakeLocations struct {
	byCustomer  map[uuid.UUID][]location.Location
	updateCalls []location.Location
}

func (f *fakeLocations) ListByCustomerID(_ context.Context, customerID uuid.UUID) ([]location.Location, error) {
	return f.byCustomer[customerID], nil
}

func (f *fakeLocations) Update(_ context.Context, l location.Location) (location.Location, error) {
	f.updateCalls = append(f.updateCalls, l)
	for cid, locs := range f.byCustomer {
		for i, existing := range locs {
			if existing.ID == l.ID {
				f.byCustomer[cid][i] = l
			}
		}
	}
	return l, nil
}

// fakeServices is an in-memory serviceRepository.
type fakeServices struct {
	byLocation  map[uuid.UUID][]service.Service
	updateCalls []service.Service
}

func (f *fakeServices) ListByLocationID(_ context.Context, locationID uuid.UUID) ([]service.Service, error) {
	return f.byLocation[locationID], nil
}

func (f *fakeServices) Update(_ context.Context, svc service.Service) (service.Service, error) {
	f.updateCalls = append(f.updateCalls, svc)
	for lid, svcs := range f.byLocation {
		for i, existing := range svcs {
			if existing.ID == svc.ID {
				f.byLocation[lid][i] = svc
			}
		}
	}
	return svc, nil
}

// fakeDevices is an in-memory deviceGetter.
type fakeDevices struct {
	byID map[uuid.UUID]inventory.Device
}

func (f *fakeDevices) Get(_ context.Context, id uuid.UUID) (inventory.Device, error) {
	d, ok := f.byID[id]
	if !ok {
		return inventory.Device{}, errors.New("not found")
	}
	return d, nil
}

// fakeEquipment is an in-memory equipmentLister + equipmentUpdater.
type fakeEquipment struct {
	byService   map[uuid.UUID][]serviceequipment.ServiceEquipment
	updateCalls []serviceequipment.ServiceEquipment
}

func (f *fakeEquipment) ListActiveByServiceID(_ context.Context, serviceID uuid.UUID) ([]serviceequipment.ServiceEquipment, error) {
	var active []serviceequipment.ServiceEquipment
	for _, eq := range f.byService[serviceID] {
		if eq.Active() {
			active = append(active, eq)
		}
	}
	return active, nil
}

func (f *fakeEquipment) Update(_ context.Context, e serviceequipment.ServiceEquipment) (serviceequipment.ServiceEquipment, error) {
	f.updateCalls = append(f.updateCalls, e)
	for sid, eqs := range f.byService {
		for i, existing := range eqs {
			if existing.ID == e.ID {
				f.byService[sid][i] = e
			}
		}
	}
	return e, nil
}

// fakeAttachments is an in-memory attachmentGetter + attachmentUpdater.
type fakeAttachments struct {
	byEquipment map[uuid.UUID]accessattachment.AccessAttachment
	updateCalls []accessattachment.AccessAttachment
}

func (f *fakeAttachments) GetActiveByServiceEquipmentID(_ context.Context, serviceEquipmentID uuid.UUID) (accessattachment.AccessAttachment, error) {
	a, ok := f.byEquipment[serviceEquipmentID]
	if !ok || !a.Active() {
		return accessattachment.AccessAttachment{}, errors.New("not found")
	}
	return a, nil
}

func (f *fakeAttachments) Update(_ context.Context, a accessattachment.AccessAttachment) (accessattachment.AccessAttachment, error) {
	f.updateCalls = append(f.updateCalls, a)
	f.byEquipment[a.ServiceEquipmentID] = a
	return a, nil
}

// fakeProfiles is an in-memory serviceProfileRemover.
type fakeProfiles struct {
	err   error
	calls []serviceequipment.ServiceEquipment
}

func (f *fakeProfiles) Remove(_ context.Context, _ service.Service, equipment serviceequipment.ServiceEquipment) (string, error) {
	f.calls = append(f.calls, equipment)
	if f.err != nil {
		return "", f.err
	}
	return "xgs/6/3", nil
}

var fixedTime = time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)

// testFixture bundles every fake plus a ready-to-use RemovalService, and
// the IDs of a single Customer -> Location -> Service -> ServiceEquipment
// -> AccessAttachment chain, for tests that build on the common case.
type testFixture struct {
	customers   *fakeCustomers
	locations   *fakeLocations
	services    *fakeServices
	devices     *fakeDevices
	equipment   *fakeEquipment
	attachments *fakeAttachments
	profiles    *fakeProfiles
	svc         *RemovalService

	customerID uuid.UUID
	locationID uuid.UUID
	serviceID  uuid.UUID
	equipID    uuid.UUID
}

func newFixture(equipmentRole serviceequipment.EquipmentRole, serviceStatus service.ServiceStatus, withAttachment bool) *testFixture {
	customerID, locationID, serviceID, equipID, deviceID, attachmentID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()

	f := &testFixture{
		customers: &fakeCustomers{byID: map[uuid.UUID]customer.Customer{
			customerID: {ID: customerID, Status: customer.CustomerStatusActive},
		}},
		locations: &fakeLocations{byCustomer: map[uuid.UUID][]location.Location{
			customerID: {{ID: locationID, CustomerID: customerID, Name: "Main St", Status: location.LocationStatusActive}},
		}},
		services: &fakeServices{byLocation: map[uuid.UUID][]service.Service{
			locationID: {{ID: serviceID, LocationID: locationID, Status: serviceStatus}},
		}},
		devices: &fakeDevices{byID: map[uuid.UUID]inventory.Device{
			deviceID: {Metadata: inventory.Metadata{ID: deviceID, Name: "ONT-1"}},
		}},
		equipment: &fakeEquipment{byService: map[uuid.UUID][]serviceequipment.ServiceEquipment{
			serviceID: {{ID: equipID, ServiceID: serviceID, DeviceID: deviceID, Role: equipmentRole}},
		}},
		attachments: &fakeAttachments{byEquipment: map[uuid.UUID]accessattachment.AccessAttachment{}},
		profiles:    &fakeProfiles{},

		customerID: customerID, locationID: locationID, serviceID: serviceID, equipID: equipID,
	}
	if withAttachment {
		f.attachments.byEquipment[equipID] = accessattachment.AccessAttachment{ID: attachmentID, ServiceEquipmentID: equipID}
	}
	f.svc = NewRemovalService(f.customers, f.locations, f.services, f.devices, f.equipment, f.equipment, f.attachments, f.attachments, f.profiles, clock.NewFrozen(fixedTime))
	return f
}

func TestExecuteHappyPathTransitionsEverythingAndRunsOLTTeardown(t *testing.T) {
	f := newFixture(serviceequipment.EquipmentRoleONU, service.ServiceStatusActive, true)

	if err := f.svc.Execute(context.Background(), f.customerID); err != nil {
		t.Fatalf("Execute() = %v", err)
	}

	if len(f.profiles.calls) != 1 {
		t.Fatalf("OLT teardown calls = %d, want 1", len(f.profiles.calls))
	}

	if got := f.attachments.byEquipment[f.equipID]; got.RemovedAt == nil || !got.RemovedAt.Equal(fixedTime) {
		t.Errorf("attachment RemovedAt = %v, want %v", got.RemovedAt, fixedTime)
	}

	updatedEquip := f.equipment.byService[f.serviceID][0]
	if updatedEquip.RemovedAt == nil || !updatedEquip.RemovedAt.Equal(fixedTime) {
		t.Errorf("equipment RemovedAt = %v, want %v", updatedEquip.RemovedAt, fixedTime)
	}

	updatedService := f.services.byLocation[f.locationID][0]
	if updatedService.Status != service.ServiceStatusDisconnected {
		t.Errorf("service status = %s, want Disconnected", updatedService.Status)
	}
	if updatedService.DisconnectedAt == nil || !updatedService.DisconnectedAt.Equal(fixedTime) {
		t.Errorf("service DisconnectedAt = %v, want %v", updatedService.DisconnectedAt, fixedTime)
	}

	updatedLocation := f.locations.byCustomer[f.customerID][0]
	if updatedLocation.Status != location.LocationStatusInactive {
		t.Errorf("location status = %s, want Inactive", updatedLocation.Status)
	}

	updatedCustomer := f.customers.byID[f.customerID]
	if updatedCustomer.Status != customer.CustomerStatusArchived {
		t.Errorf("customer status = %s, want Archived", updatedCustomer.Status)
	}
}

func TestExecuteSkipsOLTTeardownForNonONURole(t *testing.T) {
	f := newFixture(serviceequipment.EquipmentRoleRouter, service.ServiceStatusActive, true)

	if err := f.svc.Execute(context.Background(), f.customerID); err != nil {
		t.Fatalf("Execute() = %v", err)
	}

	if len(f.profiles.calls) != 0 {
		t.Errorf("OLT teardown calls = %d, want 0 for a Router", len(f.profiles.calls))
	}
	if f.equipment.byService[f.serviceID][0].RemovedAt == nil {
		t.Error("equipment was not marked removed")
	}
}

func TestExecuteSkipsOLTTeardownWhenNoActiveAttachment(t *testing.T) {
	f := newFixture(serviceequipment.EquipmentRoleONU, service.ServiceStatusActive, false)

	if err := f.svc.Execute(context.Background(), f.customerID); err != nil {
		t.Fatalf("Execute() = %v", err)
	}

	if len(f.profiles.calls) != 0 {
		t.Errorf("OLT teardown calls = %d, want 0 with no active attachment", len(f.profiles.calls))
	}
}

func TestExecuteSkipsOLTTeardownForAlreadyDisconnectedService(t *testing.T) {
	f := newFixture(serviceequipment.EquipmentRoleONU, service.ServiceStatusDisconnected, true)

	if err := f.svc.Execute(context.Background(), f.customerID); err != nil {
		t.Fatalf("Execute() = %v", err)
	}

	if len(f.profiles.calls) != 0 {
		t.Errorf("OLT teardown calls = %d, want 0 for an already-Disconnected service", len(f.profiles.calls))
	}
	// Equipment still gets unassigned even though the service was already
	// Disconnected before this cascade ran (Disconnect alone never
	// unassigns equipment).
	if f.equipment.byService[f.serviceID][0].RemovedAt == nil {
		t.Error("equipment was not marked removed for an already-Disconnected service")
	}
}

// TestExecuteSkipsOLTTeardownForAlreadySuspendedService proves
// ServiceStatus.HasAppliedProfile's own point: Suspend and Disconnect
// are, at the Kontron config level, the identical action (see that
// method's doc comment) — a Suspended Service's profile is already gone
// from the real ONU, so running Remove against it a second time here
// would hit the real device's "profile not assigned to interface"
// rejection instead of a harmless no-op. Confirmed live (2026-09-10)
// against the real Kontron/Iskratel C16 after an earlier version of this
// logic got this wrong and treated Suspended the same as Active.
func TestExecuteSkipsOLTTeardownForAlreadySuspendedService(t *testing.T) {
	f := newFixture(serviceequipment.EquipmentRoleONU, service.ServiceStatusSuspended, true)

	if err := f.svc.Execute(context.Background(), f.customerID); err != nil {
		t.Fatalf("Execute() = %v", err)
	}

	if len(f.profiles.calls) != 0 {
		t.Errorf("OLT teardown calls = %d, want 0 for an already-Suspended service", len(f.profiles.calls))
	}
	if f.equipment.byService[f.serviceID][0].RemovedAt == nil {
		t.Error("equipment was not marked removed for an already-Suspended service")
	}
}

func TestExecuteHandlesServiceWithNoActiveEquipment(t *testing.T) {
	f := newFixture(serviceequipment.EquipmentRoleONU, service.ServiceStatusPending, true)
	f.equipment.byService[f.serviceID] = nil // no equipment at all

	if err := f.svc.Execute(context.Background(), f.customerID); err != nil {
		t.Fatalf("Execute() = %v", err)
	}

	if f.services.byLocation[f.locationID][0].Status != service.ServiceStatusDisconnected {
		t.Error("service with no equipment did not transition to Disconnected")
	}
}

func TestExecuteStopsOnOLTTeardownFailureAndPreservesPriorProgress(t *testing.T) {
	// Two independent locations under the same customer: the first has no
	// equipment needing OLT teardown and should fully complete; the
	// second's teardown fails and should stop the cascade there.
	customerID, loc1ID, loc2ID := uuid.New(), uuid.New(), uuid.New()
	svc1ID, svc2ID := uuid.New(), uuid.New()
	equip2ID, device2ID, attachment2ID := uuid.New(), uuid.New(), uuid.New()

	customers := &fakeCustomers{byID: map[uuid.UUID]customer.Customer{customerID: {ID: customerID, Status: customer.CustomerStatusActive}}}
	locations := &fakeLocations{byCustomer: map[uuid.UUID][]location.Location{
		customerID: {
			{ID: loc1ID, CustomerID: customerID, Status: location.LocationStatusActive},
			{ID: loc2ID, CustomerID: customerID, Status: location.LocationStatusActive},
		},
	}}
	services := &fakeServices{byLocation: map[uuid.UUID][]service.Service{
		loc1ID: {{ID: svc1ID, LocationID: loc1ID, Status: service.ServiceStatusPending}},
		loc2ID: {{ID: svc2ID, LocationID: loc2ID, Status: service.ServiceStatusActive}},
	}}
	devices := &fakeDevices{byID: map[uuid.UUID]inventory.Device{device2ID: {Metadata: inventory.Metadata{ID: device2ID, Name: "ONT-2"}}}}
	equipment := &fakeEquipment{byService: map[uuid.UUID][]serviceequipment.ServiceEquipment{
		svc2ID: {{ID: equip2ID, ServiceID: svc2ID, DeviceID: device2ID, Role: serviceequipment.EquipmentRoleONU}},
	}}
	attachments := &fakeAttachments{byEquipment: map[uuid.UUID]accessattachment.AccessAttachment{
		equip2ID: {ID: attachment2ID, ServiceEquipmentID: equip2ID},
	}}
	profiles := &fakeProfiles{err: errors.New("connection refused")}

	svc := NewRemovalService(customers, locations, services, devices, equipment, equipment, attachments, attachments, profiles, clock.NewFrozen(fixedTime))

	err := svc.Execute(context.Background(), customerID)
	if err == nil {
		t.Fatal("Execute() error = nil, want the OLT teardown failure")
	}

	if locations.byCustomer[customerID][0].Status != location.LocationStatusInactive {
		t.Error("first location should have completed before the failure")
	}
	if locations.byCustomer[customerID][1].Status == location.LocationStatusInactive {
		t.Error("second location should not have completed")
	}
	if customers.byID[customerID].Status == customer.CustomerStatusArchived {
		t.Error("customer must not be archived when a location failed")
	}
}

func TestExecuteIsIdempotentOnRetry(t *testing.T) {
	f := newFixture(serviceequipment.EquipmentRoleONU, service.ServiceStatusActive, true)

	if err := f.svc.Execute(context.Background(), f.customerID); err != nil {
		t.Fatalf("first Execute() = %v", err)
	}
	firstTeardownCalls := len(f.profiles.calls)

	if err := f.svc.Execute(context.Background(), f.customerID); err != nil {
		t.Fatalf("second Execute() = %v", err)
	}

	if len(f.profiles.calls) != firstTeardownCalls {
		t.Errorf("second Execute() ran %d more OLT teardown(s); an already-removed customer must be a no-op", len(f.profiles.calls)-firstTeardownCalls)
	}
}

func TestPreviewReportsWithoutChangingAnything(t *testing.T) {
	f := newFixture(serviceequipment.EquipmentRoleONU, service.ServiceStatusActive, true)

	preview, err := f.svc.Preview(context.Background(), f.customerID)
	if err != nil {
		t.Fatalf("Preview() = %v", err)
	}

	if len(preview.Locations) != 1 || len(preview.Locations[0].Services) != 1 || len(preview.Locations[0].Services[0].Equipment) != 1 {
		t.Fatalf("Preview() = %+v, want one location/service/equipment", preview)
	}
	eq := preview.Locations[0].Services[0].Equipment[0]
	if !eq.WillRunOLTTeardown {
		t.Error("WillRunOLTTeardown = false, want true for an Active ONU with an active attachment")
	}
	if eq.DeviceName != "ONT-1" {
		t.Errorf("DeviceName = %q, want %q", eq.DeviceName, "ONT-1")
	}

	if len(f.profiles.calls) != 0 {
		t.Error("Preview() must never call the OLT teardown")
	}
	if f.customers.byID[f.customerID].Status != customer.CustomerStatusActive {
		t.Error("Preview() must never change the customer's status")
	}
}
