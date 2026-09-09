package devicemodel

import (
	"context"

	"github.com/google/uuid"
)

// DeviceModelRepository persists DeviceModels. It follows the exact
// shape of every other repository in this codebase (see e.g.
// internal/oltmodel.OLTModelRepository): Get, List, Create, Update,
// Delete, with Create and Update returning the persisted entity so a
// caller sees anything the store sets (e.g. timestamps) without a
// second read.
//
// List returns every DeviceModel regardless of ManufacturerID, the same
// "no server-side filtering" shape internal/oltmodel.OLTModelRepository
// and most other catalogs in this codebase use — a caller narrowing to
// one Manufacturer's Models (e.g. a cascading picker) filters
// client-side, the same pattern this codebase's frontend already uses
// throughout (see e.g. oltRepository.ts's listOLTsByAccessNetworkId).
//
// Nothing in this package implements it — no SQL, no migrations — so the
// domain has zero dependency on any storage technology. A concrete
// implementation (internal/devicemodel/postgres) satisfies it.
type DeviceModelRepository interface {
	Get(ctx context.Context, id uuid.UUID) (DeviceModel, error)
	List(ctx context.Context) ([]DeviceModel, error)
	Create(ctx context.Context, m DeviceModel) (DeviceModel, error)
	Update(ctx context.Context, m DeviceModel) (DeviceModel, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
