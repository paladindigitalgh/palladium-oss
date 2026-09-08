package oltmodel

import (
	"context"

	"github.com/google/uuid"
)

// OLTModelRepository persists OLTModels. It follows the exact shape of
// every other repository in this codebase (see e.g.
// provider.ProviderRepository): Get, List, Create, Update, Delete, with
// Create and Update returning the persisted entity so a caller sees
// anything the store sets (e.g. timestamps) without a second read.
//
// Nothing in this package implements it — no SQL, no migrations — so the
// domain has zero dependency on any storage technology. A concrete
// implementation (internal/oltmodel/postgres) satisfies it.
//
// internal/olt/service.OLTService depends on this interface directly
// (not just internal/oltmodel/httpapi) to look up a referenced
// OLTModel's PONPortCount when auto-creating PON ports — see that
// package's doc comment for why that cross-domain dependency exists.
type OLTModelRepository interface {
	Get(ctx context.Context, id uuid.UUID) (OLTModel, error)
	List(ctx context.Context) ([]OLTModel, error)
	Create(ctx context.Context, m OLTModel) (OLTModel, error)
	Update(ctx context.Context, m OLTModel) (OLTModel, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
