package provisioning_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/provisioning"
)

// stubProvisioningRepository has no SQL implementation to test yet — that
// is internal/provisioning/postgres's job. It exists solely to prove
// ProvisioningRepository is satisfiable with a sane, consistent method
// shape, mirroring internal/serviceequipment/repository_test.go's stub
// for the same reason: the var block's compile-time assertion is the
// actual check — this file fails to build if the interface and this stub
// ever drift apart.
type stubProvisioningRepository struct{}

func (stubProvisioningRepository) Get(context.Context, uuid.UUID) (provisioning.ProvisioningJob, error) {
	return provisioning.ProvisioningJob{}, nil
}
func (stubProvisioningRepository) List(context.Context) ([]provisioning.ProvisioningJob, error) {
	return nil, nil
}
func (stubProvisioningRepository) ListByServiceID(context.Context, uuid.UUID) ([]provisioning.ProvisioningJob, error) {
	return nil, nil
}
func (stubProvisioningRepository) Create(_ context.Context, j provisioning.ProvisioningJob) (provisioning.ProvisioningJob, error) {
	return j, nil
}
func (stubProvisioningRepository) Update(_ context.Context, j provisioning.ProvisioningJob) (provisioning.ProvisioningJob, error) {
	return j, nil
}
func (stubProvisioningRepository) Delete(context.Context, uuid.UUID) error { return nil }

var _ provisioning.ProvisioningRepository = (*stubProvisioningRepository)(nil)

func TestProvisioningRepositoryInterfaceIsSatisfiable(t *testing.T) {
	// The compile-time assertion above is the real test: if this package
	// builds, ProvisioningRepository has the intended
	// Get/List/ListByServiceID/Create/Update/Delete shape. This test
	// exists so `go test` reports that check explicitly instead of the
	// file silently containing no tests.
}
