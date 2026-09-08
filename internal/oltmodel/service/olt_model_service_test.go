package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/oltmodel"
	"github.com/paladindigitalgh/palladium-oss/internal/oltmodel/service"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeOLTModelRepository is an in-memory oltmodel.OLTModelRepository.
// Like internal/provider/service/provider_service_test.go's
// fakeProviderRepository, it exists so OLTModelService's business logic
// — validate, then delegate — is tested without a real database;
// internal/oltmodel/postgres/oltmodel_test.go already covers the
// repository itself against real PostgreSQL.
type fakeOLTModelRepository struct {
	byID         map[uuid.UUID]oltmodel.OLTModel
	createCalled bool
	updateCalled bool
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

func (f *fakeOLTModelRepository) List(_ context.Context) ([]oltmodel.OLTModel, error) {
	models := make([]oltmodel.OLTModel, 0, len(f.byID))
	for _, m := range f.byID {
		models = append(models, m)
	}
	return models, nil
}

func (f *fakeOLTModelRepository) Create(_ context.Context, m oltmodel.OLTModel) (oltmodel.OLTModel, error) {
	f.createCalled = true
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	f.byID[m.ID] = m
	return m, nil
}

func (f *fakeOLTModelRepository) Update(_ context.Context, m oltmodel.OLTModel) (oltmodel.OLTModel, error) {
	f.updateCalled = true
	if _, ok := f.byID[m.ID]; !ok {
		return oltmodel.OLTModel{}, apperror.NotFound("olt model not found")
	}
	f.byID[m.ID] = m
	return m, nil
}

func (f *fakeOLTModelRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apperror.NotFound("olt model not found")
	}
	delete(f.byID, id)
	return nil
}

var _ oltmodel.OLTModelRepository = (*fakeOLTModelRepository)(nil)

func validOLTModel() oltmodel.OLTModel {
	return oltmodel.OLTModel{
		Vendor:       oltmodel.VendorKontron,
		Name:         "C16",
		PONPortCount: 16,
	}
}

func TestOLTModelServiceCreateSucceeds(t *testing.T) {
	repo := newFakeOLTModelRepository()
	svc := service.NewOLTModelService(repo)

	created, err := svc.Create(context.Background(), validOLTModel())
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

func TestOLTModelServiceCreateRejectsInvalidModelWithoutPersisting(t *testing.T) {
	repo := newFakeOLTModelRepository()
	svc := service.NewOLTModelService(repo)

	_, err := svc.Create(context.Background(), oltmodel.OLTModel{}) // no Name, Vendor, PONPortCount

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestOLTModelServiceUpdateSucceeds(t *testing.T) {
	existing := validOLTModel()
	existing.ID = uuid.New()
	repo := newFakeOLTModelRepository(existing)
	svc := service.NewOLTModelService(repo)

	toUpdate := existing
	toUpdate.Name = "C32"
	toUpdate.PONPortCount = 32

	updated, err := svc.Update(context.Background(), toUpdate)
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if updated.Name != "C32" {
		t.Errorf("Name = %q, want %q", updated.Name, "C32")
	}
	if updated.PONPortCount != 32 {
		t.Errorf("PONPortCount = %d, want %d", updated.PONPortCount, 32)
	}
	if !repo.updateCalled {
		t.Error("repository Update() was never called")
	}
}

func TestOLTModelServiceUpdateRejectsInvalidModelWithoutPersisting(t *testing.T) {
	existing := validOLTModel()
	existing.ID = uuid.New()
	repo := newFakeOLTModelRepository(existing)
	svc := service.NewOLTModelService(repo)

	invalid := existing
	invalid.PONPortCount = 0 // invalid

	_, err := svc.Update(context.Background(), invalid)

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.updateCalled {
		t.Error("repository Update() was called despite invalid input; validation must happen first")
	}
}

func TestOLTModelServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakeOLTModelRepository()
	svc := service.NewOLTModelService(repo)

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestOLTModelServiceListDelegatesToRepository(t *testing.T) {
	a := validOLTModel()
	a.ID = uuid.New()
	b := validOLTModel()
	b.ID = uuid.New()
	b.Name = "C8"
	repo := newFakeOLTModelRepository(a, b)
	svc := service.NewOLTModelService(repo)

	models, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(models))
	}
}

func TestOLTModelServiceDeleteSucceeds(t *testing.T) {
	existing := validOLTModel()
	existing.ID = uuid.New()
	repo := newFakeOLTModelRepository(existing)
	svc := service.NewOLTModelService(repo)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestOLTModelServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakeOLTModelRepository()
	svc := service.NewOLTModelService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}
