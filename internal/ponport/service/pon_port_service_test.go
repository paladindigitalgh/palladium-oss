package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/ponport"
	"github.com/paladindigitalgh/palladium-oss/internal/ponport/service"
)

// fakePONPortRepository is an in-memory ponport.PONPortRepository. Like
// internal/olt/service/olt_service_test.go's fakeOLTRepository, it
// exists so PONPortService's business logic — validate, then delegate —
// is tested without a real database;
// internal/ponport/postgres/pon_port_test.go already covers the
// repository itself against real PostgreSQL. It tracks whether
// Create/Update were actually invoked, which is what lets
// TestPONPortServiceCreateRejectsInvalidPONPortWithoutPersisting prove
// validation happens before any repository call.
type fakePONPortRepository struct {
	byID         map[uuid.UUID]ponport.PONPort
	createCalled bool
	updateCalled bool
}

func newFakePONPortRepository(ports ...ponport.PONPort) *fakePONPortRepository {
	f := &fakePONPortRepository{byID: make(map[uuid.UUID]ponport.PONPort)}
	for _, p := range ports {
		f.byID[p.ID] = p
	}
	return f
}

func (f *fakePONPortRepository) Get(_ context.Context, id uuid.UUID) (ponport.PONPort, error) {
	p, ok := f.byID[id]
	if !ok {
		return ponport.PONPort{}, apperror.NotFound("pon port not found")
	}
	return p, nil
}

func (f *fakePONPortRepository) List(_ context.Context) ([]ponport.PONPort, error) {
	ports := make([]ponport.PONPort, 0, len(f.byID))
	for _, p := range f.byID {
		ports = append(ports, p)
	}
	return ports, nil
}

func (f *fakePONPortRepository) Create(_ context.Context, p ponport.PONPort) (ponport.PONPort, error) {
	f.createCalled = true
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	f.byID[p.ID] = p
	return p, nil
}

func (f *fakePONPortRepository) Update(_ context.Context, p ponport.PONPort) (ponport.PONPort, error) {
	f.updateCalled = true
	if _, ok := f.byID[p.ID]; !ok {
		return ponport.PONPort{}, apperror.NotFound("pon port not found")
	}
	f.byID[p.ID] = p
	return p, nil
}

func (f *fakePONPortRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apperror.NotFound("pon port not found")
	}
	delete(f.byID, id)
	return nil
}

var _ ponport.PONPortRepository = (*fakePONPortRepository)(nil)

func validPONPort() ponport.PONPort {
	return ponport.PONPort{
		OLTID:      uuid.New(),
		PortNumber: 1,
	}
}

func TestPONPortServiceCreateSucceeds(t *testing.T) {
	repo := newFakePONPortRepository()
	svc := service.NewPONPortService(repo)

	created, err := svc.Create(context.Background(), validPONPort())
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

func TestPONPortServiceCreateRejectsInvalidPONPortWithoutPersisting(t *testing.T) {
	repo := newFakePONPortRepository()
	svc := service.NewPONPortService(repo)

	_, err := svc.Create(context.Background(), ponport.PONPort{}) // no OLTID, PortNumber

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestPONPortServiceUpdateSucceeds(t *testing.T) {
	existing := validPONPort()
	existing.ID = uuid.New()
	repo := newFakePONPortRepository(existing)
	svc := service.NewPONPortService(repo)

	toUpdate := existing
	toUpdate.PortNumber = 2

	updated, err := svc.Update(context.Background(), toUpdate)
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if updated.PortNumber != 2 {
		t.Errorf("PortNumber = %d, want %d", updated.PortNumber, 2)
	}
	if !repo.updateCalled {
		t.Error("repository Update() was never called")
	}
}

func TestPONPortServiceUpdateRejectsInvalidPONPortWithoutPersisting(t *testing.T) {
	existing := validPONPort()
	existing.ID = uuid.New()
	repo := newFakePONPortRepository(existing)
	svc := service.NewPONPortService(repo)

	invalid := existing
	invalid.PortNumber = 0 // invalid

	_, err := svc.Update(context.Background(), invalid)

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.updateCalled {
		t.Error("repository Update() was called despite invalid input; validation must happen first")
	}
}

func TestPONPortServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakePONPortRepository()
	svc := service.NewPONPortService(repo)

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestPONPortServiceListDelegatesToRepository(t *testing.T) {
	a := validPONPort()
	a.ID = uuid.New()
	b := validPONPort()
	b.ID = uuid.New()
	repo := newFakePONPortRepository(a, b)
	svc := service.NewPONPortService(repo)

	ports, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(ports) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(ports))
	}
}

func TestPONPortServiceDeleteSucceeds(t *testing.T) {
	existing := validPONPort()
	existing.ID = uuid.New()
	repo := newFakePONPortRepository(existing)
	svc := service.NewPONPortService(repo)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestPONPortServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakePONPortRepository()
	svc := service.NewPONPortService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}
