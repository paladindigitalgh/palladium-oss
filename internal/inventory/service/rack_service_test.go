package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory/service"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeRackRepository is an in-memory inventory.RackRepository. See
// fakeSiteRepository's doc comment in site_service_test.go for why this
// exists (and why it tracks createCalled/updateCalled).
type fakeRackRepository struct {
	byID         map[uuid.UUID]inventory.Rack
	createCalled bool
	updateCalled bool
}

func newFakeRackRepository(racks ...inventory.Rack) *fakeRackRepository {
	f := &fakeRackRepository{byID: make(map[uuid.UUID]inventory.Rack)}
	for _, r := range racks {
		f.byID[r.ID] = r
	}
	return f
}

func (f *fakeRackRepository) Get(_ context.Context, id uuid.UUID) (inventory.Rack, error) {
	r, ok := f.byID[id]
	if !ok {
		return inventory.Rack{}, apperror.NotFound("rack not found")
	}
	return r, nil
}

func (f *fakeRackRepository) List(_ context.Context) ([]inventory.Rack, error) {
	racks := make([]inventory.Rack, 0, len(f.byID))
	for _, r := range f.byID {
		racks = append(racks, r)
	}
	return racks, nil
}

func (f *fakeRackRepository) Create(_ context.Context, rack inventory.Rack) (inventory.Rack, error) {
	f.createCalled = true
	if rack.ID == uuid.Nil {
		rack.ID = uuid.New()
	}
	f.byID[rack.ID] = rack
	return rack, nil
}

func (f *fakeRackRepository) Update(_ context.Context, rack inventory.Rack) (inventory.Rack, error) {
	f.updateCalled = true
	if _, ok := f.byID[rack.ID]; !ok {
		return inventory.Rack{}, apperror.NotFound("rack not found")
	}
	f.byID[rack.ID] = rack
	return rack, nil
}

func (f *fakeRackRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apperror.NotFound("rack not found")
	}
	delete(f.byID, id)
	return nil
}

var _ inventory.RackRepository = (*fakeRackRepository)(nil)

func TestRackServiceCreateSucceeds(t *testing.T) {
	repo := newFakeRackRepository()
	svc := service.NewRackService(repo)

	rack := inventory.Rack{Metadata: inventory.Metadata{Name: "Rack A1"}}

	created, err := svc.Create(context.Background(), rack)
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

// TestRackServiceCreateSucceedsWithNilRoomID proves RoomID being nil does
// not fail validation -- see Rack.Validate's own doc comment for why.
func TestRackServiceCreateSucceedsWithNilRoomID(t *testing.T) {
	repo := newFakeRackRepository()
	svc := service.NewRackService(repo)

	created, err := svc.Create(context.Background(), inventory.Rack{Metadata: inventory.Metadata{Name: "Warehouse Rack"}, RoomID: nil})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	if created.RoomID != nil {
		t.Errorf("RoomID = %v, want nil", created.RoomID)
	}
}

func TestRackServiceCreateRejectsInvalidRackWithoutPersisting(t *testing.T) {
	repo := newFakeRackRepository()
	svc := service.NewRackService(repo)

	_, err := svc.Create(context.Background(), inventory.Rack{}) // no Name

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestRackServiceUpdateSucceeds(t *testing.T) {
	existing := inventory.Rack{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Old Name"}}
	repo := newFakeRackRepository(existing)
	svc := service.NewRackService(repo)

	updated, err := svc.Update(context.Background(), inventory.Rack{
		Metadata: inventory.Metadata{ID: existing.ID, Name: "New Name"},
	})
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if !repo.updateCalled {
		t.Error("repository Update() was never called")
	}
}

func TestRackServiceUpdateRejectsInvalidRackWithoutPersisting(t *testing.T) {
	existing := inventory.Rack{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Old Name"}}
	repo := newFakeRackRepository(existing)
	svc := service.NewRackService(repo)

	_, err := svc.Update(context.Background(), inventory.Rack{
		Metadata: inventory.Metadata{ID: existing.ID, Name: ""}, // invalid
	})

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.updateCalled {
		t.Error("repository Update() was called despite invalid input; validation must happen first")
	}
}

func TestRackServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakeRackRepository()
	svc := service.NewRackService(repo)

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestRackServiceListDelegatesToRepository(t *testing.T) {
	a := inventory.Rack{Metadata: inventory.Metadata{ID: uuid.New(), Name: "A"}}
	b := inventory.Rack{Metadata: inventory.Metadata{ID: uuid.New(), Name: "B"}}
	repo := newFakeRackRepository(a, b)
	svc := service.NewRackService(repo)

	racks, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(racks) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(racks))
	}
}

func TestRackServiceDeleteSucceeds(t *testing.T) {
	existing := inventory.Rack{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Temporary"}}
	repo := newFakeRackRepository(existing)
	svc := service.NewRackService(repo)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestRackServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakeRackRepository()
	svc := service.NewRackService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}
