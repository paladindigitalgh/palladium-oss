package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/olt"
	"github.com/paladindigitalgh/palladium-oss/internal/olt/service"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeOLTRepository is an in-memory olt.OLTRepository. Like
// internal/accessnetwork/service/access_network_service_test.go's
// fakeAccessNetworkRepository, it exists so OLTService's business logic
// — validate, then delegate — is tested without a real database;
// internal/olt/postgres/olt_test.go already covers the repository itself
// against real PostgreSQL. It tracks whether Create/Update were actually
// invoked, which is what lets
// TestOLTServiceCreateRejectsInvalidOLTWithoutPersisting prove
// validation happens before any repository call.
type fakeOLTRepository struct {
	byID         map[uuid.UUID]olt.OLT
	createCalled bool
	updateCalled bool
}

func newFakeOLTRepository(olts ...olt.OLT) *fakeOLTRepository {
	f := &fakeOLTRepository{byID: make(map[uuid.UUID]olt.OLT)}
	for _, o := range olts {
		f.byID[o.ID] = o
	}
	return f
}

func (f *fakeOLTRepository) Get(_ context.Context, id uuid.UUID) (olt.OLT, error) {
	o, ok := f.byID[id]
	if !ok {
		return olt.OLT{}, apperror.NotFound("olt not found")
	}
	return o, nil
}

func (f *fakeOLTRepository) List(_ context.Context) ([]olt.OLT, error) {
	olts := make([]olt.OLT, 0, len(f.byID))
	for _, o := range f.byID {
		olts = append(olts, o)
	}
	return olts, nil
}

func (f *fakeOLTRepository) Create(_ context.Context, o olt.OLT) (olt.OLT, error) {
	f.createCalled = true
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	f.byID[o.ID] = o
	return o, nil
}

func (f *fakeOLTRepository) Update(_ context.Context, o olt.OLT) (olt.OLT, error) {
	f.updateCalled = true
	if _, ok := f.byID[o.ID]; !ok {
		return olt.OLT{}, apperror.NotFound("olt not found")
	}
	f.byID[o.ID] = o
	return o, nil
}

func (f *fakeOLTRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apperror.NotFound("olt not found")
	}
	delete(f.byID, id)
	return nil
}

var _ olt.OLTRepository = (*fakeOLTRepository)(nil)

func validOLT() olt.OLT {
	return olt.OLT{
		AccessNetworkID: uuid.New(),
		Name:            "OLT-01",
		Vendor:          olt.VendorKontron,
	}
}

func TestOLTServiceCreateSucceeds(t *testing.T) {
	repo := newFakeOLTRepository()
	svc := service.NewOLTService(repo)

	created, err := svc.Create(context.Background(), validOLT())
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

func TestOLTServiceCreateRejectsInvalidOLTWithoutPersisting(t *testing.T) {
	repo := newFakeOLTRepository()
	svc := service.NewOLTService(repo)

	_, err := svc.Create(context.Background(), olt.OLT{}) // no AccessNetworkID, Name, Vendor

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestOLTServiceUpdateSucceeds(t *testing.T) {
	existing := validOLT()
	existing.ID = uuid.New()
	repo := newFakeOLTRepository(existing)
	svc := service.NewOLTService(repo)

	toUpdate := existing
	toUpdate.Name = "New Name"
	toUpdate.Vendor = olt.VendorNokia

	updated, err := svc.Update(context.Background(), toUpdate)
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.Vendor != olt.VendorNokia {
		t.Errorf("Vendor = %q, want %q", updated.Vendor, olt.VendorNokia)
	}
	if !repo.updateCalled {
		t.Error("repository Update() was never called")
	}
}

func TestOLTServiceUpdateRejectsInvalidOLTWithoutPersisting(t *testing.T) {
	existing := validOLT()
	existing.ID = uuid.New()
	repo := newFakeOLTRepository(existing)
	svc := service.NewOLTService(repo)

	invalid := existing
	invalid.Name = "" // invalid

	_, err := svc.Update(context.Background(), invalid)

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.updateCalled {
		t.Error("repository Update() was called despite invalid input; validation must happen first")
	}
}

func TestOLTServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakeOLTRepository()
	svc := service.NewOLTService(repo)

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestOLTServiceListDelegatesToRepository(t *testing.T) {
	a := validOLT()
	a.ID = uuid.New()
	b := validOLT()
	b.ID = uuid.New()
	repo := newFakeOLTRepository(a, b)
	svc := service.NewOLTService(repo)

	olts, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(olts) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(olts))
	}
}

func TestOLTServiceDeleteSucceeds(t *testing.T) {
	existing := validOLT()
	existing.ID = uuid.New()
	repo := newFakeOLTRepository(existing)
	svc := service.NewOLTService(repo)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestOLTServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakeOLTRepository()
	svc := service.NewOLTService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}
