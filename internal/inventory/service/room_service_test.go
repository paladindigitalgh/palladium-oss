package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory/service"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeRoomRepository is an in-memory inventory.RoomRepository. See
// fakeSiteRepository's doc comment in site_service_test.go for why this
// exists (and why it tracks createCalled/updateCalled).
type fakeRoomRepository struct {
	byID         map[uuid.UUID]inventory.Room
	createCalled bool
	updateCalled bool
}

func newFakeRoomRepository(rooms ...inventory.Room) *fakeRoomRepository {
	f := &fakeRoomRepository{byID: make(map[uuid.UUID]inventory.Room)}
	for _, r := range rooms {
		f.byID[r.ID] = r
	}
	return f
}

func (f *fakeRoomRepository) Get(_ context.Context, id uuid.UUID) (inventory.Room, error) {
	r, ok := f.byID[id]
	if !ok {
		return inventory.Room{}, apperror.NotFound("room not found")
	}
	return r, nil
}

func (f *fakeRoomRepository) List(_ context.Context) ([]inventory.Room, error) {
	rooms := make([]inventory.Room, 0, len(f.byID))
	for _, r := range f.byID {
		rooms = append(rooms, r)
	}
	return rooms, nil
}

func (f *fakeRoomRepository) Create(_ context.Context, room inventory.Room) (inventory.Room, error) {
	f.createCalled = true
	if room.ID == uuid.Nil {
		room.ID = uuid.New()
	}
	f.byID[room.ID] = room
	return room, nil
}

func (f *fakeRoomRepository) Update(_ context.Context, room inventory.Room) (inventory.Room, error) {
	f.updateCalled = true
	if _, ok := f.byID[room.ID]; !ok {
		return inventory.Room{}, apperror.NotFound("room not found")
	}
	f.byID[room.ID] = room
	return room, nil
}

func (f *fakeRoomRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apperror.NotFound("room not found")
	}
	delete(f.byID, id)
	return nil
}

var _ inventory.RoomRepository = (*fakeRoomRepository)(nil)

func TestRoomServiceCreateSucceeds(t *testing.T) {
	repo := newFakeRoomRepository()
	svc := service.NewRoomService(repo)

	room := inventory.Room{Metadata: inventory.Metadata{Name: "Server Room"}, BuildingID: uuid.New()}

	created, err := svc.Create(context.Background(), room)
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

func TestRoomServiceCreateRejectsInvalidRoomWithoutPersisting(t *testing.T) {
	repo := newFakeRoomRepository()
	svc := service.NewRoomService(repo)

	_, err := svc.Create(context.Background(), inventory.Room{}) // no Name, no BuildingID

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestRoomServiceUpdateSucceeds(t *testing.T) {
	buildingID := uuid.New()
	existing := inventory.Room{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Old Name"}, BuildingID: buildingID}
	repo := newFakeRoomRepository(existing)
	svc := service.NewRoomService(repo)

	updated, err := svc.Update(context.Background(), inventory.Room{
		Metadata:   inventory.Metadata{ID: existing.ID, Name: "New Name"},
		BuildingID: buildingID,
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

func TestRoomServiceUpdateRejectsInvalidRoomWithoutPersisting(t *testing.T) {
	existing := inventory.Room{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Old Name"}, BuildingID: uuid.New()}
	repo := newFakeRoomRepository(existing)
	svc := service.NewRoomService(repo)

	_, err := svc.Update(context.Background(), inventory.Room{
		Metadata:   inventory.Metadata{ID: existing.ID, Name: ""}, // invalid
		BuildingID: existing.BuildingID,
	})

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.updateCalled {
		t.Error("repository Update() was called despite invalid input; validation must happen first")
	}
}

func TestRoomServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakeRoomRepository()
	svc := service.NewRoomService(repo)

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestRoomServiceListDelegatesToRepository(t *testing.T) {
	a := inventory.Room{Metadata: inventory.Metadata{ID: uuid.New(), Name: "A"}, BuildingID: uuid.New()}
	b := inventory.Room{Metadata: inventory.Metadata{ID: uuid.New(), Name: "B"}, BuildingID: uuid.New()}
	repo := newFakeRoomRepository(a, b)
	svc := service.NewRoomService(repo)

	rooms, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(rooms) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(rooms))
	}
}

func TestRoomServiceDeleteSucceeds(t *testing.T) {
	existing := inventory.Room{Metadata: inventory.Metadata{ID: uuid.New(), Name: "Temporary"}, BuildingID: uuid.New()}
	repo := newFakeRoomRepository(existing)
	svc := service.NewRoomService(repo)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestRoomServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakeRoomRepository()
	svc := service.NewRoomService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}
