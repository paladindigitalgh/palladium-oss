package serviceprofile_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/serviceprofile"
)

// stubServiceProfileRepository has no SQL implementation to test yet —
// that is internal/serviceprofile/postgres's job. It exists solely to
// prove ServiceProfileRepository is satisfiable with a sane, consistent
// method shape, mirroring internal/catalog/repository_test.go's stub for
// the same reason: the var block's compile-time assertion is the actual
// check — this file fails to build if the interface and this stub ever
// drift apart.
type stubServiceProfileRepository struct{}

func (stubServiceProfileRepository) Get(context.Context, uuid.UUID) (serviceprofile.ServiceProfile, error) {
	return serviceprofile.ServiceProfile{}, nil
}
func (stubServiceProfileRepository) List(context.Context) ([]serviceprofile.ServiceProfile, error) {
	return nil, nil
}
func (stubServiceProfileRepository) Create(_ context.Context, p serviceprofile.ServiceProfile) (serviceprofile.ServiceProfile, error) {
	return p, nil
}
func (stubServiceProfileRepository) Update(_ context.Context, p serviceprofile.ServiceProfile) (serviceprofile.ServiceProfile, error) {
	return p, nil
}
func (stubServiceProfileRepository) Delete(context.Context, uuid.UUID) error { return nil }

var _ serviceprofile.ServiceProfileRepository = (*stubServiceProfileRepository)(nil)

func TestServiceProfileRepositoryInterfaceIsSatisfiable(t *testing.T) {
	// The compile-time assertion above is the real test: if this package
	// builds, ServiceProfileRepository has the intended
	// Get/List/Create/Update/Delete shape. This test exists so `go test`
	// reports that check explicitly instead of the file silently
	// containing no tests.
}
