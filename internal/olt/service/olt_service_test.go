package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/olt"
	"github.com/paladindigitalgh/palladium-oss/internal/olt/service"
	"github.com/paladindigitalgh/palladium-oss/internal/oltmodel"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/ponport"
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

// fakeOLTModelRepository is an in-memory oltmodel.OLTModelRepository,
// used here only for its Get method — OLTService never lists, creates,
// updates, or deletes OLTModels itself (see internal/oltmodel/service
// for that), it only looks one up to read PONPortCount when
// auto-creating PON ports.
type fakeOLTModelRepository struct {
	byID map[uuid.UUID]oltmodel.OLTModel
}

func newFakeOLTModelRepository(models ...oltmodel.OLTModel) *fakeOLTModelRepository {
	f := &fakeOLTModelRepository{byID: make(map[uuid.UUID]oltmodel.OLTModel)}
	for _, m := range models {
		f.byID[m.ID] = m
	}
	return f
}

func (f *fakeOLTModelRepository) Get(_ context.Context, id uuid.UUID) (oltmodel.OLTModel, error) {
	m, ok := f.byID[id]
	if !ok {
		return oltmodel.OLTModel{}, apperror.NotFound("olt model not found")
	}
	return m, nil
}

func (f *fakeOLTModelRepository) List(context.Context) ([]oltmodel.OLTModel, error) {
	panic("not used by OLTService")
}

func (f *fakeOLTModelRepository) Create(context.Context, oltmodel.OLTModel) (oltmodel.OLTModel, error) {
	panic("not used by OLTService")
}

func (f *fakeOLTModelRepository) Update(context.Context, oltmodel.OLTModel) (oltmodel.OLTModel, error) {
	panic("not used by OLTService")
}

func (f *fakeOLTModelRepository) Delete(context.Context, uuid.UUID) error {
	panic("not used by OLTService")
}

var _ oltmodel.OLTModelRepository = (*fakeOLTModelRepository)(nil)

// fakePONPortRepository is an in-memory ponport.PONPortRepository.
// OLTService uses Create (auto-provisioning, see Create's doc comment),
// List and Delete (the Delete cascade, see Delete's doc comment) — Get
// and Update are never called by OLTService and still panic if invoked.
type fakePONPortRepository struct {
	created      []ponport.PONPort
	failAfter    int // if > 0, Create fails starting on the (failAfter+1)th call
	createCalled int
	failDeleteID uuid.UUID // if set, Delete for this one ID returns a conflict
	deleted      []uuid.UUID
}

func (f *fakePONPortRepository) Get(context.Context, uuid.UUID) (ponport.PONPort, error) {
	panic("not used by OLTService")
}

// List returns a copy of created, not the field itself: a real
// repository's List hits Postgres and returns an independent slice each
// call, so a caller that ranges over the result while also calling
// Delete (as OLTService.Delete does) must not observe this fake mutating
// its own backing array mid-range.
func (f *fakePONPortRepository) List(context.Context) ([]ponport.PONPort, error) {
	ports := make([]ponport.PONPort, len(f.created))
	copy(ports, f.created)
	return ports, nil
}

func (f *fakePONPortRepository) Create(_ context.Context, p ponport.PONPort) (ponport.PONPort, error) {
	f.createCalled++
	if f.failAfter > 0 && f.createCalled > f.failAfter {
		return ponport.PONPort{}, apperror.Internal("create pon port", nil)
	}
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	f.created = append(f.created, p)
	return p, nil
}

func (f *fakePONPortRepository) Update(context.Context, ponport.PONPort) (ponport.PONPort, error) {
	panic("not used by OLTService")
}

// Delete removes id from created, simulating a real delete, unless id
// matches failDeleteID — simulating the RESTRICT conflict a PON port
// with a real AccessInterface attached would return.
func (f *fakePONPortRepository) Delete(_ context.Context, id uuid.UUID) error {
	if f.failDeleteID != uuid.Nil && id == f.failDeleteID {
		return apperror.Conflict("pon port has an access interface attached")
	}
	for i, p := range f.created {
		if p.ID == id {
			f.created = append(f.created[:i], f.created[i+1:]...)
			break
		}
	}
	f.deleted = append(f.deleted, id)
	return nil
}

var _ ponport.PONPortRepository = (*fakePONPortRepository)(nil)

func validOLT() olt.OLT {
	return olt.OLT{
		AccessNetworkID: uuid.New(),
		Name:            "OLT-01",
		OLTModelID:      uuid.New(),
	}
}

// newOLTServiceWithModel builds an OLTService wired to a fake OLTModel
// (with the given PONPortCount) that validOLT()'s OLTModelID resolves
// to, and a fresh fakePONPortRepository — the setup every auto-port-
// creation test below shares.
func newOLTServiceWithModel(oltModelID uuid.UUID, ponPortCount int) (*service.OLTService, *fakeOLTRepository, *fakePONPortRepository) {
	oltRepo := newFakeOLTRepository()
	modelRepo := newFakeOLTModelRepository(oltmodel.OLTModel{
		ID:           oltModelID,
		Vendor:       oltmodel.VendorKontron,
		Name:         "C16",
		PONPortCount: ponPortCount,
	})
	portRepo := &fakePONPortRepository{}
	return service.NewOLTService(oltRepo, modelRepo, portRepo), oltRepo, portRepo
}

func TestOLTServiceCreateSucceeds(t *testing.T) {
	o := validOLT()
	svc, oltRepo, _ := newOLTServiceWithModel(o.OLTModelID, 16)

	created, err := svc.Create(context.Background(), o)
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if !oltRepo.createCalled {
		t.Error("repository Create() was never called")
	}
}

func TestOLTServiceCreateAutoCreatesPONPortsMatchingModelCount(t *testing.T) {
	o := validOLT()
	svc, _, portRepo := newOLTServiceWithModel(o.OLTModelID, 16)

	created, err := svc.Create(context.Background(), o)
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if len(portRepo.created) != 16 {
		t.Fatalf("len(created ports) = %d, want 16", len(portRepo.created))
	}
	for i, port := range portRepo.created {
		wantPortNumber := i + 1
		if port.PortNumber != wantPortNumber {
			t.Errorf("created ports[%d].PortNumber = %d, want %d", i, port.PortNumber, wantPortNumber)
		}
		if port.OLTID != created.ID {
			t.Errorf("created ports[%d].OLTID = %v, want %v", i, port.OLTID, created.ID)
		}
	}
}

func TestOLTServiceCreateAutoCreatesPONPortsForSmallerModel(t *testing.T) {
	o := validOLT()
	svc, _, portRepo := newOLTServiceWithModel(o.OLTModelID, 8)

	if _, err := svc.Create(context.Background(), o); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if len(portRepo.created) != 8 {
		t.Fatalf("len(created ports) = %d, want 8", len(portRepo.created))
	}
}

func TestOLTServiceCreateFailsWhenOLTModelCannotBeLoaded(t *testing.T) {
	o := validOLT() // OLTModelID does not resolve in modelRepo below
	oltRepo := newFakeOLTRepository()
	modelRepo := newFakeOLTModelRepository() // empty: Get always NotFound
	portRepo := &fakePONPortRepository{}
	svc := service.NewOLTService(oltRepo, modelRepo, portRepo)

	_, err := svc.Create(context.Background(), o)

	if err == nil {
		t.Fatal("Create() = nil error, want an error when the referenced OLTModel cannot be loaded")
	}
	if len(portRepo.created) != 0 {
		t.Errorf("len(created ports) = %d, want 0 when the OLTModel lookup fails", len(portRepo.created))
	}
}

func TestOLTServiceCreateReturnsErrorWhenPortCreationFailsPartway(t *testing.T) {
	o := validOLT()
	oltRepo := newFakeOLTRepository()
	modelRepo := newFakeOLTModelRepository(oltmodel.OLTModel{
		ID:           o.OLTModelID,
		Vendor:       oltmodel.VendorKontron,
		Name:         "C16",
		PONPortCount: 16,
	})
	portRepo := &fakePONPortRepository{failAfter: 3} // ports 1-3 succeed, port 4 fails
	svc := service.NewOLTService(oltRepo, modelRepo, portRepo)

	_, err := svc.Create(context.Background(), o)

	if err == nil {
		t.Fatal("Create() = nil error, want an error when a PON port fails to create")
	}
	if len(portRepo.created) != 3 {
		t.Errorf("len(created ports) = %d, want 3 (this is a best-effort cascade, not atomic)", len(portRepo.created))
	}
}

func TestOLTServiceCreateRejectsInvalidOLTWithoutPersisting(t *testing.T) {
	svc, oltRepo, _ := newOLTServiceWithModel(uuid.New(), 16)

	_, err := svc.Create(context.Background(), olt.OLT{}) // no AccessNetworkID, Name, OLTModelID

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if oltRepo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestOLTServiceUpdateSucceeds(t *testing.T) {
	existing := validOLT()
	existing.ID = uuid.New()
	oltRepo := newFakeOLTRepository(existing)
	svc := service.NewOLTService(oltRepo, newFakeOLTModelRepository(), &fakePONPortRepository{})

	toUpdate := existing
	toUpdate.Name = "New Name"
	toUpdate.OLTModelID = uuid.New()

	updated, err := svc.Update(context.Background(), toUpdate)
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.OLTModelID != toUpdate.OLTModelID {
		t.Errorf("OLTModelID = %v, want %v", updated.OLTModelID, toUpdate.OLTModelID)
	}
	if !oltRepo.updateCalled {
		t.Error("repository Update() was never called")
	}
}

func TestOLTServiceUpdateNeverCreatesPONPorts(t *testing.T) {
	existing := validOLT()
	existing.ID = uuid.New()
	oltRepo := newFakeOLTRepository(existing)
	portRepo := &fakePONPortRepository{}
	svc := service.NewOLTService(oltRepo, newFakeOLTModelRepository(), portRepo)

	toUpdate := existing
	toUpdate.OLTModelID = uuid.New()
	if _, err := svc.Update(context.Background(), toUpdate); err != nil {
		t.Fatalf("Update() = %v", err)
	}

	if len(portRepo.created) != 0 {
		t.Errorf("len(created ports) = %d, want 0 -- Update must never auto-create ports", len(portRepo.created))
	}
}

func TestOLTServiceUpdateRejectsInvalidOLTWithoutPersisting(t *testing.T) {
	existing := validOLT()
	existing.ID = uuid.New()
	oltRepo := newFakeOLTRepository(existing)
	svc := service.NewOLTService(oltRepo, newFakeOLTModelRepository(), &fakePONPortRepository{})

	invalid := existing
	invalid.Name = "" // invalid

	_, err := svc.Update(context.Background(), invalid)

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if oltRepo.updateCalled {
		t.Error("repository Update() was called despite invalid input; validation must happen first")
	}
}

func TestOLTServiceGetPropagatesNotFound(t *testing.T) {
	svc, _, _ := newOLTServiceWithModel(uuid.New(), 16)

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
	oltRepo := newFakeOLTRepository(a, b)
	svc := service.NewOLTService(oltRepo, newFakeOLTModelRepository(), &fakePONPortRepository{})

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
	oltRepo := newFakeOLTRepository(existing)
	svc := service.NewOLTService(oltRepo, newFakeOLTModelRepository(), &fakePONPortRepository{})

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestOLTServiceDeletePropagatesNotFound(t *testing.T) {
	svc, _, _ := newOLTServiceWithModel(uuid.New(), 16)

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

// TestOLTServiceDeleteRemovesItsPONPorts proves the cascade Delete's own
// doc comment describes: an OLT created through Create (and so already
// carrying auto-created PON ports) can be deleted in one call, with every
// one of its ports removed first.
func TestOLTServiceDeleteRemovesItsPONPorts(t *testing.T) {
	o := validOLT()
	svc, oltRepo, portRepo := newOLTServiceWithModel(o.OLTModelID, 16)

	created, err := svc.Create(context.Background(), o)
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	if len(portRepo.created) != 16 {
		t.Fatalf("ports created = %d, want 16", len(portRepo.created))
	}

	if err := svc.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	if len(portRepo.created) != 0 {
		t.Errorf("ports remaining after Delete() = %d, want 0", len(portRepo.created))
	}
	if len(portRepo.deleted) != 16 {
		t.Errorf("ports deleted = %d, want 16", len(portRepo.deleted))
	}
	if _, err := oltRepo.Get(context.Background(), created.ID); !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

// TestOLTServiceDeleteFailsWhenAPONPortIsStillInUse proves the safety
// half of the cascade: a PON port that cannot itself be deleted (a real
// AccessInterface RESTRICTs it) aborts the whole OLT delete instead of
// silently skipping that port or deleting the OLT anyway.
func TestOLTServiceDeleteFailsWhenAPONPortIsStillInUse(t *testing.T) {
	o := validOLT()
	svc, oltRepo, portRepo := newOLTServiceWithModel(o.OLTModelID, 4)

	created, err := svc.Create(context.Background(), o)
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	portRepo.failDeleteID = portRepo.created[2].ID

	err = svc.Delete(context.Background(), created.ID)

	if !apperror.Is(err, apperror.KindConflict) {
		t.Fatalf("Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindConflict)
	}
	if _, err := oltRepo.Get(context.Background(), created.ID); err != nil {
		t.Errorf("OLT should still exist after a failed Delete(), Get() = %v", err)
	}
}
