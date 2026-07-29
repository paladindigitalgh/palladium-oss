package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/authentication"
	"github.com/paladindigitalgh/palladium-oss/internal/authentication/service"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeAuthenticationRepository is an in-memory
// authentication.AuthenticationRepository. Like
// internal/catalog/service/catalog_service_test.go's
// fakeCatalogRepository, it exists so AuthenticationService's business
// logic — validate, then delegate — is tested without a real database;
// internal/authentication/postgres/authentication_test.go already
// covers the repository itself (including encryption) against real
// PostgreSQL. It tracks whether Create/Update were actually invoked,
// which is what lets
// TestAuthenticationServiceCreateRejectsInvalidAuthenticationWithoutPersisting
// prove validation happens before any repository call.
type fakeAuthenticationRepository struct {
	byID         map[uuid.UUID]authentication.Authentication
	createCalled bool
	updateCalled bool
}

func newFakeAuthenticationRepository(auths ...authentication.Authentication) *fakeAuthenticationRepository {
	f := &fakeAuthenticationRepository{byID: make(map[uuid.UUID]authentication.Authentication)}
	for _, a := range auths {
		f.byID[a.ID] = a
	}
	return f
}

func (f *fakeAuthenticationRepository) Get(_ context.Context, id uuid.UUID) (authentication.Authentication, error) {
	a, ok := f.byID[id]
	if !ok {
		return authentication.Authentication{}, apperror.NotFound("authentication not found")
	}
	return a, nil
}

func (f *fakeAuthenticationRepository) List(_ context.Context) ([]authentication.Authentication, error) {
	auths := make([]authentication.Authentication, 0, len(f.byID))
	for _, a := range f.byID {
		auths = append(auths, a)
	}
	return auths, nil
}

func (f *fakeAuthenticationRepository) Create(_ context.Context, a authentication.Authentication) (authentication.Authentication, error) {
	f.createCalled = true
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	f.byID[a.ID] = a
	return a, nil
}

func (f *fakeAuthenticationRepository) Update(_ context.Context, a authentication.Authentication) (authentication.Authentication, error) {
	f.updateCalled = true
	if _, ok := f.byID[a.ID]; !ok {
		return authentication.Authentication{}, apperror.NotFound("authentication not found")
	}
	f.byID[a.ID] = a
	return a, nil
}

func (f *fakeAuthenticationRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apperror.NotFound("authentication not found")
	}
	delete(f.byID, id)
	return nil
}

var _ authentication.AuthenticationRepository = (*fakeAuthenticationRepository)(nil)

func validAuthentication() authentication.Authentication {
	return authentication.Authentication{
		Name:               "Default Device Login",
		AuthenticationType: authentication.AuthenticationTypePassword,
		Username:           "admin",
		Password:           "hunter2",
	}
}

func TestAuthenticationServiceCreateSucceeds(t *testing.T) {
	repo := newFakeAuthenticationRepository()
	svc := service.NewAuthenticationService(repo)

	created, err := svc.Create(context.Background(), validAuthentication())
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

func TestAuthenticationServiceCreateRejectsInvalidAuthenticationWithoutPersisting(t *testing.T) {
	repo := newFakeAuthenticationRepository()
	svc := service.NewAuthenticationService(repo)

	_, err := svc.Create(context.Background(), authentication.Authentication{}) // no Name, Type, Username

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestAuthenticationServiceUpdateSucceeds(t *testing.T) {
	existing := validAuthentication()
	existing.ID = uuid.New()
	repo := newFakeAuthenticationRepository(existing)
	svc := service.NewAuthenticationService(repo)

	toUpdate := existing
	toUpdate.Username = "root"

	updated, err := svc.Update(context.Background(), toUpdate)
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if updated.Username != "root" {
		t.Errorf("Username = %q, want %q", updated.Username, "root")
	}
	if !repo.updateCalled {
		t.Error("repository Update() was never called")
	}
}

func TestAuthenticationServiceUpdateRejectsInvalidAuthenticationWithoutPersisting(t *testing.T) {
	existing := validAuthentication()
	existing.ID = uuid.New()
	repo := newFakeAuthenticationRepository(existing)
	svc := service.NewAuthenticationService(repo)

	invalid := existing
	invalid.Username = "" // invalid

	_, err := svc.Update(context.Background(), invalid)

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.updateCalled {
		t.Error("repository Update() was called despite invalid input; validation must happen first")
	}
}

func TestAuthenticationServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakeAuthenticationRepository()
	svc := service.NewAuthenticationService(repo)

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestAuthenticationServiceListDelegatesToRepository(t *testing.T) {
	a := validAuthentication()
	a.ID = uuid.New()
	b := validAuthentication()
	b.ID = uuid.New()
	repo := newFakeAuthenticationRepository(a, b)
	svc := service.NewAuthenticationService(repo)

	auths, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(auths) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(auths))
	}
}

func TestAuthenticationServiceDeleteSucceeds(t *testing.T) {
	existing := validAuthentication()
	existing.ID = uuid.New()
	repo := newFakeAuthenticationRepository(existing)
	svc := service.NewAuthenticationService(repo)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestAuthenticationServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakeAuthenticationRepository()
	svc := service.NewAuthenticationService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}
