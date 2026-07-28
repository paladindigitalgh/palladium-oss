package service

import (
	"context"

	"github.com/google/uuid"
)

// ServiceRepository persists Services. It follows the exact shape of
// every other repository in this codebase (see e.g.
// internal/location.LocationRepository): Get, List, Create, Update,
// Delete, with Create and Update returning the persisted entity so a
// caller sees anything the store sets (e.g. timestamps) without a second
// read.
//
// Nothing in this package implements it — no SQL, no migrations — so the
// domain has zero dependency on any storage technology. A concrete
// implementation (internal/service/postgres) satisfies it.
type ServiceRepository interface {
	Get(ctx context.Context, id uuid.UUID) (Service, error)
	List(ctx context.Context) ([]Service, error)
	Create(ctx context.Context, service Service) (Service, error)
	Update(ctx context.Context, service Service) (Service, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
