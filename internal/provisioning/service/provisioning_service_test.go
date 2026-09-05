package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning/service"
)

// fakeProvisioningProfileRepository is an in-memory
// provisioning.ProvisioningProfileRepository. Like
// internal/product/service/product_service_test.go's
// fakeProductRepository, it exists so ProvisioningProfileService's
// business logic — validate, then delegate — is tested without a real
// database; internal/provisioning/postgres/provisioning_profile_test.go
// already covers the repository itself against real PostgreSQL.
type fakeProvisioningProfileRepository struct {
	byID         map[uuid.UUID]provisioning.ProvisioningProfile
	createCalled bool
	updateCalled bool
}

func newFakeProvisioningProfileRepository(profiles ...provisioning.ProvisioningProfile) *fakeProvisioningProfileRepository {
	f := &fakeProvisioningProfileRepository{byID: make(map[uuid.UUID]provisioning.ProvisioningProfile)}
	for _, p := range profiles {
		f.byID[p.ID] = p
	}
	return f
}

func (f *fakeProvisioningProfileRepository) Get(_ context.Context, id uuid.UUID) (provisioning.ProvisioningProfile, error) {
	p, ok := f.byID[id]
	if !ok {
		return provisioning.ProvisioningProfile{}, apperror.NotFound("provisioning profile not found")
	}
	return p, nil
}

func (f *fakeProvisioningProfileRepository) List(_ context.Context) ([]provisioning.ProvisioningProfile, error) {
	profiles := make([]provisioning.ProvisioningProfile, 0, len(f.byID))
	for _, p := range f.byID {
		profiles = append(profiles, p)
	}
	return profiles, nil
}

func (f *fakeProvisioningProfileRepository) Create(_ context.Context, p provisioning.ProvisioningProfile) (provisioning.ProvisioningProfile, error) {
	f.createCalled = true
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	f.byID[p.ID] = p
	return p, nil
}

func (f *fakeProvisioningProfileRepository) Update(_ context.Context, p provisioning.ProvisioningProfile) (provisioning.ProvisioningProfile, error) {
	f.updateCalled = true
	if _, ok := f.byID[p.ID]; !ok {
		return provisioning.ProvisioningProfile{}, apperror.NotFound("provisioning profile not found")
	}
	f.byID[p.ID] = p
	return p, nil
}

func (f *fakeProvisioningProfileRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apperror.NotFound("provisioning profile not found")
	}
	delete(f.byID, id)
	return nil
}

var _ provisioning.ProvisioningProfileRepository = (*fakeProvisioningProfileRepository)(nil)

func validProvisioningProfile() provisioning.ProvisioningProfile {
	return provisioning.ProvisioningProfile{
		ProductID:   uuid.New(),
		Vendor:      "Kontron",
		ProfileName: "RES-500M",
	}
}

func TestProvisioningProfileServiceCreateSucceeds(t *testing.T) {
	repo := newFakeProvisioningProfileRepository()
	svc := service.NewProvisioningProfileService(repo)

	created, err := svc.Create(context.Background(), validProvisioningProfile())
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

func TestProvisioningProfileServiceCreateRejectsInvalidProfileWithoutPersisting(t *testing.T) {
	repo := newFakeProvisioningProfileRepository()
	svc := service.NewProvisioningProfileService(repo)

	_, err := svc.Create(context.Background(), provisioning.ProvisioningProfile{}) // no ProductID, Vendor, ProfileName

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestProvisioningProfileServiceUpdateSucceeds(t *testing.T) {
	existing := validProvisioningProfile()
	existing.ID = uuid.New()
	repo := newFakeProvisioningProfileRepository(existing)
	svc := service.NewProvisioningProfileService(repo)

	toUpdate := existing
	toUpdate.ProfileName = "RES-1000M"

	updated, err := svc.Update(context.Background(), toUpdate)
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if updated.ProfileName != "RES-1000M" {
		t.Errorf("ProfileName = %q, want %q", updated.ProfileName, "RES-1000M")
	}
	if !repo.updateCalled {
		t.Error("repository Update() was never called")
	}
}

func TestProvisioningProfileServiceUpdateRejectsInvalidProfileWithoutPersisting(t *testing.T) {
	existing := validProvisioningProfile()
	existing.ID = uuid.New()
	repo := newFakeProvisioningProfileRepository(existing)
	svc := service.NewProvisioningProfileService(repo)

	invalid := existing
	invalid.ProfileName = "" // invalid

	_, err := svc.Update(context.Background(), invalid)

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.updateCalled {
		t.Error("repository Update() was called despite invalid input; validation must happen first")
	}
}

func TestProvisioningProfileServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakeProvisioningProfileRepository()
	svc := service.NewProvisioningProfileService(repo)

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestProvisioningProfileServiceListDelegatesToRepository(t *testing.T) {
	a := validProvisioningProfile()
	a.ID = uuid.New()
	b := validProvisioningProfile()
	b.ID = uuid.New()
	repo := newFakeProvisioningProfileRepository(a, b)
	svc := service.NewProvisioningProfileService(repo)

	profiles, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(profiles))
	}
}

func TestProvisioningProfileServiceDeleteSucceeds(t *testing.T) {
	existing := validProvisioningProfile()
	existing.ID = uuid.New()
	repo := newFakeProvisioningProfileRepository(existing)
	svc := service.NewProvisioningProfileService(repo)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestProvisioningProfileServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakeProvisioningProfileRepository()
	svc := service.NewProvisioningProfileService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}
