package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/service"
)

// stubServiceRepository has no SQL implementation to test yet — that is
// internal/service/postgres's job. It exists solely to prove
// ServiceRepository is satisfiable with a sane, consistent method shape,
// mirroring internal/product/repository_test.go's stub for the same
// reason: the var block's compile-time assertion is the actual check —
// this file fails to build if the interface and this stub ever drift
// apart.
type stubServiceRepository struct{}

func (stubServiceRepository) Get(context.Context, uuid.UUID) (service.Service, error) {
	return service.Service{}, nil
}
func (stubServiceRepository) List(context.Context) ([]service.Service, error) { return nil, nil }
func (stubServiceRepository) Create(_ context.Context, s service.Service) (service.Service, error) {
	return s, nil
}
func (stubServiceRepository) Update(_ context.Context, s service.Service) (service.Service, error) {
	return s, nil
}
func (stubServiceRepository) Delete(context.Context, uuid.UUID) error { return nil }

var _ service.ServiceRepository = (*stubServiceRepository)(nil)

func TestServiceRepositoryInterfaceIsSatisfiable(t *testing.T) {
	// The compile-time assertion above is the real test: if this package
	// builds, ServiceRepository has the intended Get/List/Create/Update/
	// Delete shape. This test exists so `go test` reports that check
	// explicitly instead of the file silently containing no tests.
}
