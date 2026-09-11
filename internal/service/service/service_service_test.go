package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	domainservice "github.com/paladindigitalgh/palladium-oss/internal/service"
	"github.com/paladindigitalgh/palladium-oss/internal/service/service"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// fakeServiceRepository is an in-memory domainservice.ServiceRepository.
// Like internal/product/service/product_service_test.go's
// fakeProductRepository, it exists so ServiceService's business logic —
// validate, then delegate — is tested without a real database;
// internal/service/postgres/service_test.go already covers the
// repository itself against real PostgreSQL. It tracks whether
// Create/Update were actually invoked, which is what lets
// TestServiceServiceCreateRejectsInvalidServiceWithoutPersisting prove
// validation happens before any repository call.
type fakeServiceRepository struct {
	byID         map[uuid.UUID]domainservice.Service
	createCalled bool
	updateCalled bool
}

func newFakeServiceRepository(services ...domainservice.Service) *fakeServiceRepository {
	f := &fakeServiceRepository{byID: make(map[uuid.UUID]domainservice.Service)}
	for _, s := range services {
		f.byID[s.ID] = s
	}
	return f
}

func (f *fakeServiceRepository) Get(_ context.Context, id uuid.UUID) (domainservice.Service, error) {
	s, ok := f.byID[id]
	if !ok {
		return domainservice.Service{}, apperror.NotFound("service not found")
	}
	return s, nil
}

func (f *fakeServiceRepository) List(_ context.Context) ([]domainservice.Service, error) {
	services := make([]domainservice.Service, 0, len(f.byID))
	for _, s := range f.byID {
		services = append(services, s)
	}
	return services, nil
}

func (f *fakeServiceRepository) Create(_ context.Context, s domainservice.Service) (domainservice.Service, error) {
	f.createCalled = true
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	f.byID[s.ID] = s
	return s, nil
}

func (f *fakeServiceRepository) Update(_ context.Context, s domainservice.Service) (domainservice.Service, error) {
	f.updateCalled = true
	if _, ok := f.byID[s.ID]; !ok {
		return domainservice.Service{}, apperror.NotFound("service not found")
	}
	f.byID[s.ID] = s
	return s, nil
}

func (f *fakeServiceRepository) ListByLocationID(_ context.Context, locationID uuid.UUID) ([]domainservice.Service, error) {
	services := []domainservice.Service{}
	for _, s := range f.byID {
		if s.LocationID == locationID {
			services = append(services, s)
		}
	}
	return services, nil
}

func (f *fakeServiceRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apperror.NotFound("service not found")
	}
	delete(f.byID, id)
	return nil
}

var _ domainservice.ServiceRepository = (*fakeServiceRepository)(nil)

// fakeActiveEquipmentLister is an in-memory activeEquipmentLister.
// Defaults to "no active equipment" for any serviceID not explicitly
// seeded, the common case for every test in this file that is not about
// Delete's equipment-attached precondition specifically.
type fakeActiveEquipmentLister struct {
	byServiceID map[uuid.UUID][]serviceequipment.ServiceEquipment
}

func newFakeActiveEquipmentLister(equipment ...serviceequipment.ServiceEquipment) *fakeActiveEquipmentLister {
	f := &fakeActiveEquipmentLister{byServiceID: make(map[uuid.UUID][]serviceequipment.ServiceEquipment)}
	for _, e := range equipment {
		f.byServiceID[e.ServiceID] = append(f.byServiceID[e.ServiceID], e)
	}
	return f
}

func (f *fakeActiveEquipmentLister) ListActiveByServiceID(_ context.Context, serviceID uuid.UUID) ([]serviceequipment.ServiceEquipment, error) {
	return f.byServiceID[serviceID], nil
}

func validService() domainservice.Service {
	return domainservice.Service{
		LocationID: uuid.New(),
		ProductID:  uuid.New(),
		Status:     domainservice.ServiceStatusPending,
	}
}

func TestServiceServiceCreateSucceeds(t *testing.T) {
	repo := newFakeServiceRepository()
	svc := service.NewServiceService(repo, newFakeActiveEquipmentLister())

	created, err := svc.Create(context.Background(), validService())
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if !repo.createCalled {
		t.Error("repository Create() was never called")
	}
}

func TestServiceServiceCreateRejectsInvalidServiceWithoutPersisting(t *testing.T) {
	repo := newFakeServiceRepository()
	svc := service.NewServiceService(repo, newFakeActiveEquipmentLister())

	_, err := svc.Create(context.Background(), domainservice.Service{}) // no LocationID, ProductID, Status

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestServiceServiceUpdateSucceeds(t *testing.T) {
	existing := validService()
	existing.ID = uuid.New()
	repo := newFakeServiceRepository(existing)
	svc := service.NewServiceService(repo, newFakeActiveEquipmentLister())

	toUpdate := existing
	toUpdate.Status = domainservice.ServiceStatusActive

	updated, err := svc.Update(context.Background(), toUpdate)
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if updated.Status != domainservice.ServiceStatusActive {
		t.Errorf("Status = %q, want %q", updated.Status, domainservice.ServiceStatusActive)
	}
	if !repo.updateCalled {
		t.Error("repository Update() was never called")
	}
}

func TestServiceServiceUpdateRejectsInvalidServiceWithoutPersisting(t *testing.T) {
	existing := validService()
	existing.ID = uuid.New()
	repo := newFakeServiceRepository(existing)
	svc := service.NewServiceService(repo, newFakeActiveEquipmentLister())

	invalid := existing
	invalid.Status = "" // invalid

	_, err := svc.Update(context.Background(), invalid)

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.updateCalled {
		t.Error("repository Update() was called despite invalid input; validation must happen first")
	}
}

func TestServiceServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakeServiceRepository()
	svc := service.NewServiceService(repo, newFakeActiveEquipmentLister())

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestServiceServiceListDelegatesToRepository(t *testing.T) {
	a := validService()
	a.ID = uuid.New()
	b := validService()
	b.ID = uuid.New()
	repo := newFakeServiceRepository(a, b)
	svc := service.NewServiceService(repo, newFakeActiveEquipmentLister())

	services, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(services) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(services))
	}
}

func TestServiceServiceDeleteSucceeds(t *testing.T) {
	existing := validService()
	existing.ID = uuid.New()
	repo := newFakeServiceRepository(existing)
	svc := service.NewServiceService(repo, newFakeActiveEquipmentLister())

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestServiceServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakeServiceRepository()
	svc := service.NewServiceService(repo, newFakeActiveEquipmentLister())

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

// TestServiceServiceDeleteRejectsServiceWithAppliedProfile proves the
// precondition added after a live OLT test surfaced the gap: deleting an
// Active Service (the only status that implies its Kontron
// service-profile is still applied, see ServiceStatus.HasAppliedProfile)
// must be rejected rather than silently leaving that profile stranded on
// the real ONU with no Palladium record left to ever remove it.
func TestServiceServiceDeleteRejectsServiceWithAppliedProfile(t *testing.T) {
	existing := validService()
	existing.ID = uuid.New()
	existing.Status = domainservice.ServiceStatusActive
	repo := newFakeServiceRepository(existing)
	svc := service.NewServiceService(repo, newFakeActiveEquipmentLister())

	err := svc.Delete(context.Background(), existing.ID)

	if !apperror.Is(err, apperror.KindConflict) {
		t.Errorf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindConflict)
	}
	if repo.byID[existing.ID].ID == uuid.Nil {
		t.Error("service was deleted despite still having an applied profile")
	}
}

// TestServiceServiceDeleteAllowsServiceWithoutAppliedProfile proves the
// other half: Pending (never applied), Suspended, and Disconnected
// (Suspend and Disconnect both already remove the profile from the real
// ONU — see ServiceStatus.HasAppliedProfile's own doc comment on why
// Suspended is not treated as still-applied) are not blocked by the new
// precondition.
func TestServiceServiceDeleteAllowsServiceWithoutAppliedProfile(t *testing.T) {
	for _, status := range []domainservice.ServiceStatus{
		domainservice.ServiceStatusPending,
		domainservice.ServiceStatusSuspended,
		domainservice.ServiceStatusDisconnected,
	} {
		existing := validService()
		existing.ID = uuid.New()
		existing.Status = status
		repo := newFakeServiceRepository(existing)
		svc := service.NewServiceService(repo, newFakeActiveEquipmentLister())

		if err := svc.Delete(context.Background(), existing.ID); err != nil {
			t.Errorf("status %q: Delete() = %v, want success", status, err)
		}
	}
}

// TestServiceServiceDeleteRejectsServiceWithActiveEquipment proves
// Delete's other precondition: active ServiceEquipment still referencing
// the Service blocks deletion with a specific message, checked before
// ever calling the repository's own Delete (which would otherwise fail
// on the database's generic foreign-key-violation error).
func TestServiceServiceDeleteRejectsServiceWithActiveEquipment(t *testing.T) {
	existing := validService()
	existing.ID = uuid.New()
	repo := newFakeServiceRepository(existing)
	equipment := newFakeActiveEquipmentLister(serviceequipment.ServiceEquipment{
		ID: uuid.New(), ServiceID: existing.ID, DeviceID: uuid.New(), Role: serviceequipment.EquipmentRoleONU,
	})
	svc := service.NewServiceService(repo, equipment)

	err := svc.Delete(context.Background(), existing.ID)

	if !apperror.Is(err, apperror.KindConflict) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindConflict)
	}
	if repo.byID[existing.ID].ID == uuid.Nil {
		t.Error("service was deleted despite still having active equipment attached")
	}
}
