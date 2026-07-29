package connectionprofile_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/connectionprofile"
)

// stubConnectionProfileRepository has no SQL implementation to test yet
// — that is internal/connectionprofile/postgres's job. It exists solely
// to prove ConnectionProfileRepository is satisfiable with a sane,
// consistent method shape, mirroring
// internal/catalog/repository_test.go's stub for the same reason: the
// var block's compile-time assertion is the actual check — this file
// fails to build if the interface and this stub ever drift apart.
type stubConnectionProfileRepository struct{}

func (stubConnectionProfileRepository) Get(context.Context, uuid.UUID) (connectionprofile.ConnectionProfile, error) {
	return connectionprofile.ConnectionProfile{}, nil
}
func (stubConnectionProfileRepository) List(context.Context) ([]connectionprofile.ConnectionProfile, error) {
	return nil, nil
}
func (stubConnectionProfileRepository) Create(_ context.Context, p connectionprofile.ConnectionProfile) (connectionprofile.ConnectionProfile, error) {
	return p, nil
}
func (stubConnectionProfileRepository) Update(_ context.Context, p connectionprofile.ConnectionProfile) (connectionprofile.ConnectionProfile, error) {
	return p, nil
}
func (stubConnectionProfileRepository) Delete(context.Context, uuid.UUID) error { return nil }

var _ connectionprofile.ConnectionProfileRepository = (*stubConnectionProfileRepository)(nil)

func TestConnectionProfileRepositoryInterfaceIsSatisfiable(t *testing.T) {
	// The compile-time assertion above is the real test: if this package
	// builds, ConnectionProfileRepository has the intended
	// Get/List/Create/Update/Delete shape. This test exists so `go test`
	// reports that check explicitly instead of the file silently
	// containing no tests.
}
