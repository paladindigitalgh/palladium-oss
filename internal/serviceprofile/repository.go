package serviceprofile

import (
	"context"

	"github.com/google/uuid"
)

// ServiceProfileRepository persists ServiceProfiles. It follows the
// exact shape of every other repository in this codebase (see e.g.
// internal/catalog.CatalogRepository): Get, List, Create, Update,
// Delete, with Create and Update returning the persisted entity so a
// caller sees anything the store sets (e.g. timestamps) without a second
// read.
//
// Nothing in this package implements it — no SQL, no migrations — so the
// domain has zero dependency on any storage technology. A concrete
// implementation (internal/serviceprofile/postgres) satisfies it.
type ServiceProfileRepository interface {
	Get(ctx context.Context, id uuid.UUID) (ServiceProfile, error)
	List(ctx context.Context) ([]ServiceProfile, error)
	Create(ctx context.Context, profile ServiceProfile) (ServiceProfile, error)
	Update(ctx context.Context, profile ServiceProfile) (ServiceProfile, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
