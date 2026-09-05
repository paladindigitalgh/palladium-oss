package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/product"
	"github.com/paladindigitalgh/palladium-oss/internal/product/service"
)

// fakeProductRepository is an in-memory product.ProductRepository. Like
// internal/catalog/service/catalog_service_test.go's
// fakeCatalogRepository, it exists so ProductService's business logic —
// validate, then delegate — is tested without a real database;
// internal/product/postgres/product_test.go already covers the repository
// itself against real PostgreSQL. It tracks whether Create/Update were
// actually invoked, which is what lets
// TestProductServiceCreateRejectsInvalidProductWithoutPersisting prove
// validation happens before any repository call.
type fakeProductRepository struct {
	byID         map[uuid.UUID]product.Product
	createCalled bool
	updateCalled bool
}

func newFakeProductRepository(products ...product.Product) *fakeProductRepository {
	f := &fakeProductRepository{byID: make(map[uuid.UUID]product.Product)}
	for _, p := range products {
		f.byID[p.ID] = p
	}
	return f
}

func (f *fakeProductRepository) Get(_ context.Context, id uuid.UUID) (product.Product, error) {
	p, ok := f.byID[id]
	if !ok {
		return product.Product{}, apperror.NotFound("product not found")
	}
	return p, nil
}

func (f *fakeProductRepository) List(_ context.Context) ([]product.Product, error) {
	products := make([]product.Product, 0, len(f.byID))
	for _, p := range f.byID {
		products = append(products, p)
	}
	return products, nil
}

func (f *fakeProductRepository) Create(_ context.Context, p product.Product) (product.Product, error) {
	f.createCalled = true
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	f.byID[p.ID] = p
	return p, nil
}

func (f *fakeProductRepository) Update(_ context.Context, p product.Product) (product.Product, error) {
	f.updateCalled = true
	if _, ok := f.byID[p.ID]; !ok {
		return product.Product{}, apperror.NotFound("product not found")
	}
	f.byID[p.ID] = p
	return p, nil
}

func (f *fakeProductRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apperror.NotFound("product not found")
	}
	delete(f.byID, id)
	return nil
}

var _ product.ProductRepository = (*fakeProductRepository)(nil)

func validProduct() product.Product {
	return product.Product{
		CatalogID:  uuid.New(),
		ProviderID: uuid.New(),
		Name:       "Residential Internet 100/20",
		Category:   product.ProductCategoryInternet,
		Status:     product.ProductStatusActive,
	}
}

func TestProductServiceCreateSucceeds(t *testing.T) {
	repo := newFakeProductRepository()
	svc := service.NewProductService(repo)

	created, err := svc.Create(context.Background(), validProduct())
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

func TestProductServiceCreateRejectsInvalidProductWithoutPersisting(t *testing.T) {
	repo := newFakeProductRepository()
	svc := service.NewProductService(repo)

	_, err := svc.Create(context.Background(), product.Product{}) // no CatalogID, Name, Category, Status

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if repo.createCalled {
		t.Error("repository Create() was called despite invalid input; validation must happen first")
	}
}

func TestProductServiceUpdateSucceeds(t *testing.T) {
	existing := validProduct()
	existing.ID = uuid.New()
	repo := newFakeProductRepository(existing)
	svc := service.NewProductService(repo)

	toUpdate := existing
	toUpdate.Name = "New Name"
	toUpdate.Status = product.ProductStatusRetired

	updated, err := svc.Update(context.Background(), toUpdate)
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.Status != product.ProductStatusRetired {
		t.Errorf("Status = %q, want %q", updated.Status, product.ProductStatusRetired)
	}
	if !repo.updateCalled {
		t.Error("repository Update() was never called")
	}
}

func TestProductServiceUpdateRejectsInvalidProductWithoutPersisting(t *testing.T) {
	existing := validProduct()
	existing.ID = uuid.New()
	repo := newFakeProductRepository(existing)
	svc := service.NewProductService(repo)

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

func TestProductServiceGetPropagatesNotFound(t *testing.T) {
	repo := newFakeProductRepository()
	svc := service.NewProductService(repo)

	_, err := svc.Get(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestProductServiceListDelegatesToRepository(t *testing.T) {
	a := validProduct()
	a.ID = uuid.New()
	b := validProduct()
	b.ID = uuid.New()
	repo := newFakeProductRepository(a, b)
	svc := service.NewProductService(repo)

	products, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(products) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(products))
	}
}

func TestProductServiceDeleteSucceeds(t *testing.T) {
	existing := validProduct()
	existing.ID = uuid.New()
	repo := newFakeProductRepository(existing)
	svc := service.NewProductService(repo)

	if err := svc.Delete(context.Background(), existing.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err := svc.Get(context.Background(), existing.ID)
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Get() after Delete() Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}

func TestProductServiceDeletePropagatesNotFound(t *testing.T) {
	repo := newFakeProductRepository()
	svc := service.NewProductService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindNotFound)
	}
}
