package provisioning

import (
	"context"

	"github.com/google/uuid"
)

// ProvisioningProfileRepository persists ProvisioningProfiles. It follows
// the exact shape of every other repository in this codebase (see e.g.
// internal/product.ProductRepository): Get, List, Create, Update,
// Delete, with Create and Update returning the persisted entity so a
// caller sees anything the store sets (e.g. timestamps) without a second
// read. ListByProductID additionally supports the one real query this
// domain needs beyond a flat list: "which vendor profiles does this
// Product already have?"
//
// Nothing in this package implements it — no SQL, no migrations — so the
// domain has zero dependency on any storage technology. A concrete
// implementation (internal/provisioning/postgres) satisfies it.
//
// There is no ListByProductID: GET /provisioning-profiles has no
// server-side filtering, the same pattern
// serviceEquipmentRepository.ts's own doc comment documents for
// ServiceEquipment — a caller that needs "this Product's profiles" fetches
// the full list once and filters client-side, which is cheap at this
// domain's expected size (one row per Product per vendor).
type ProvisioningProfileRepository interface {
	Get(ctx context.Context, id uuid.UUID) (ProvisioningProfile, error)
	List(ctx context.Context) ([]ProvisioningProfile, error)
	Create(ctx context.Context, profile ProvisioningProfile) (ProvisioningProfile, error)
	Update(ctx context.Context, profile ProvisioningProfile) (ProvisioningProfile, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
