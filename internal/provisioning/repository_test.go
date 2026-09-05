package provisioning_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/provisioning"
)

// stubProvisioningProfileRepository has no SQL implementation to test yet
// — that is internal/provisioning/postgres's job. It exists solely to
// prove ProvisioningProfileRepository is satisfiable with a sane,
// consistent method shape, mirroring internal/product/repository_test.go's
// stub for the same reason: the var block's compile-time assertion is the
// actual check — this file fails to build if the interface and this stub
// ever drift apart.
type stubProvisioningProfileRepository struct{}

func (stubProvisioningProfileRepository) Get(context.Context, uuid.UUID) (provisioning.ProvisioningProfile, error) {
	return provisioning.ProvisioningProfile{}, nil
}
func (stubProvisioningProfileRepository) List(context.Context) ([]provisioning.ProvisioningProfile, error) {
	return nil, nil
}
func (stubProvisioningProfileRepository) Create(_ context.Context, p provisioning.ProvisioningProfile) (provisioning.ProvisioningProfile, error) {
	return p, nil
}
func (stubProvisioningProfileRepository) Update(_ context.Context, p provisioning.ProvisioningProfile) (provisioning.ProvisioningProfile, error) {
	return p, nil
}
func (stubProvisioningProfileRepository) Delete(context.Context, uuid.UUID) error { return nil }

var _ provisioning.ProvisioningProfileRepository = (*stubProvisioningProfileRepository)(nil)

func TestProvisioningProfileRepositoryInterfaceIsSatisfiable(t *testing.T) {
	// The compile-time assertion above is the real test: if this package
	// builds, ProvisioningProfileRepository has the intended Get/List/
	// Create/Update/Delete shape. This test exists so `go test` reports
	// that check explicitly instead of the file silently containing no
	// tests.
}
