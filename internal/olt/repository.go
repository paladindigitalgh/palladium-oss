package olt

import (
	"context"

	"github.com/google/uuid"
)

// OLTRepository persists OLTs. It follows the exact shape of every other
// repository in this codebase (see e.g. product.ProductRepository): Get,
// List, Create, Update, Delete, with Create and Update returning the
// persisted entity so a caller sees anything the store sets (e.g.
// timestamps) without a second read.
//
// Nothing in this package implements it — no SQL, no migrations — so the
// domain has zero dependency on any storage technology. A concrete
// implementation (internal/olt/postgres) satisfies it.
type OLTRepository interface {
	Get(ctx context.Context, id uuid.UUID) (OLT, error)
	List(ctx context.Context) ([]OLT, error)
	Create(ctx context.Context, olt OLT) (OLT, error)
	Update(ctx context.Context, olt OLT) (OLT, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
