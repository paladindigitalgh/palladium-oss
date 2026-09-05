package provider

import (
	"context"

	"github.com/google/uuid"
)

// ProviderRepository persists Providers. It follows the exact shape of
// every other repository in this codebase (see e.g.
// internal/serviceprofile.ServiceProfileRepository): Get, List, Create,
// Update, Delete, with Create and Update returning the persisted entity
// so a caller sees anything the store sets (e.g. timestamps) without a
// second read.
//
// Nothing in this package implements it — no SQL, no migrations — so the
// domain has zero dependency on any storage technology. A concrete
// implementation (internal/provider/postgres) satisfies it.
type ProviderRepository interface {
	Get(ctx context.Context, id uuid.UUID) (Provider, error)
	List(ctx context.Context) ([]Provider, error)
	Create(ctx context.Context, p Provider) (Provider, error)
	Update(ctx context.Context, p Provider) (Provider, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
