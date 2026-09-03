package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/contact"
	"github.com/paladindigitalgh/palladium-oss/internal/contact/service"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeContactRepository is an in-memory contact.ContactRepository. Like
// internal/location/service/location_service_test.go's
// fakeLocationRepository, it exists so ContactService's business logic —
// validate, then delegate — is tested without a real database;
// internal/contact/postgres/contact_test.go already covers the
// repository itself against real PostgreSQL. It tracks whether
// Create/Update were actually invoked, which is what lets
// TestContactServiceCreateRejectsInvalidContactWithoutPersisting prove
// validation happens before any repository call.
type fakeContactRepository struct {
	byID         map[uuid.UUID]contact.Contact
	createCalled bool
	updateCalled bool
}

func newFakeContactRepository(contacts ...contact.Contact) *fakeContactRepository {
	f := &fakeContactRepository{byID: make(map[uuid.UUID]contact.Contact)}
	for _, c := range contacts {
		f.byID[c.ID] = c
	}
	return f
}

func (f *fakeContactRepository) Get(_ context.Context, id uuid.UUID) (contact.Contact, error) {
	c, ok := f.byID[id]
	if !ok {
		return contact.Contact{}, apperror.NotFound("contact not found")
	}
	return c, nil
}

func (f *fakeContactRepository) List(_ context.Context) ([]contact.Contact, error) {
	contacts := make([]contact.Contact, 0, len(f.byID))
	for _, c := range f.byID {
		contacts = append(contacts, c)
	}
	return contacts, nil
}

func (f *fakeContactRepository) Create(_ context.Context, c contact.Contact) (contact.Contact, error) {
	f.createCalled = true
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	f.byID[c.ID] = c
	return c, nil
}

func (f *fakeContactRepository) Update(_ context.Context, c contact.Contact) (contact.Contact, error) {
	f.updateCalled = true
	if _, ok := f.byID[c.ID]; !ok {
		return contact.Contact{}, apperror.NotFound("contact not found")
	}
	f.byID[c.ID] = c
	return c, nil
}

func (f *fakeContactRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apperror.NotFound("contact not found")
	}
	delete(f.byID, id)
	return nil
}

var _ contact.ContactRepository = (*fakeContactRepository)(nil)

func validContact() contact.Contact {
	return contact.Contact{
		CustomerID: uuid.New(),
		Name:       "Jane Doe",
		Role:       contact.ContactRolePrimary,
		Status:     contact.ContactStatusActive,
	}
}

func TestContactServiceCreateSucceeds(t *testing.T) {
	repo := newFakeContactRepository()
	svc := service.NewContactService(repo)

	created, err := svc.Create(context.Background(), validContact())
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

func TestContactServiceCreateRejectsInvalidContactWithoutPersisting(t *testing.T) {
	repo := newFakeContactRepository()
	svc := service.NewContactService(repo)

	_, err := svc.Create(context.Background(), contact.Contact{}) // no CustomerID, Name, Role, Status

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestContactServiceUpdateSucceeds(t *testing.T) {
	existing := validContact()
	existing.ID = uuid.New()
	repo := newFakeContactRepository(existing)
	svc := service.NewContactService(repo)

	toUpdate := existing
	toUpdate.Name = "New Name"
	toUpdate.Status = contact.ContactStatusInactive

	updated, err := svc.Update(context.Background(), toUpdate)
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.Status != contact.ContactStatusInactive {
		t.Errorf("Status = %q, want %q", updated.Status, contact.ContactStatusInactive)
	}
	if !repo.updateCalled {
		t.Error("repository Update() was never called")
	}
}

func TestContactServiceUpdateRejectsInvalidContactWithoutPersisting(t *testing.T) {
	existing := validContact()
	existing.ID = uuid.New()
	repo := newFakeContactRepository(existing)
	svc := service.NewContactService(repo)

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

func TestContactServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakeContactRepository()
	svc := service.NewContactService(repo)

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestContactServiceListDelegatesToRepository(t *testing.T) {
	a := validContact()
	a.ID = uuid.New()
	b := validContact()
	b.ID = uuid.New()
	repo := newFakeContactRepository(a, b)
	svc := service.NewContactService(repo)

	contacts, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(contacts) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(contacts))
	}
}

func TestContactServiceDeleteSucceeds(t *testing.T) {
	existing := validContact()
	existing.ID = uuid.New()
	repo := newFakeContactRepository(existing)
	svc := service.NewContactService(repo)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestContactServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakeContactRepository()
	svc := service.NewContactService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}
