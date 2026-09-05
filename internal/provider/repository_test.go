package provider_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/provider"
)

// stubProviderRepository has no SQL implementation to test yet — that is
// internal/provider/postgres's job. It exists solely to prove
// ProviderRepository is satisfiable with a sane, consistent method
// shape, mirroring internal/serviceprofile/repository_test.go's stub for
// the same reason: the var block's compile-time assertion is the actual
// check — this file fails to build if the interface and this stub ever
// drift apart.
type stubProviderRepository struct{}

func (stubProviderRepository) Get(context.Context, uuid.UUID) (provider.Provider, error) {
	return provider.Provider{}, nil
}
func (stubProviderRepository) List(context.Context) ([]provider.Provider, error) { return nil, nil }
func (stubProviderRepository) Create(_ context.Context, p provider.Provider) (provider.Provider, error) {
	return p, nil
}
func (stubProviderRepository) Update(_ context.Context, p provider.Provider) (provider.Provider, error) {
	return p, nil
}
func (stubProviderRepository) Delete(context.Context, uuid.UUID) error { return nil }

var _ provider.ProviderRepository = (*stubProviderRepository)(nil)

func TestProviderRepositoryInterfaceIsSatisfiable(t *testing.T) {
	// The compile-time assertion above is the real test: if this package
	// builds, ProviderRepository has the intended Get/List/Create/Update/
	// Delete shape. This test exists so `go test` reports that check
	// explicitly instead of the file silently containing no tests.
}
