package inventory

import (
	"context"

	"github.com/google/uuid"
)

// Repository interfaces are the boundary future persistence code depends
// on. Nothing in this package implements them yet — no SQL, no
// migrations — so the domain has zero dependency on any storage
// technology. A concrete implementation (e.g. a future
// internal/inventory/postgres package built on internal/database.Querier)
// satisfies one of these per entity.
//
// Every repository follows the same shape: Get, List, Create, Update,
// Delete. Create and Update return the persisted entity so a caller sees
// anything the store sets (e.g. timestamps) without a second read.

// SiteRepository persists Sites.
type SiteRepository interface {
	Get(ctx context.Context, id uuid.UUID) (Site, error)
	List(ctx context.Context) ([]Site, error)
	Create(ctx context.Context, site Site) (Site, error)
	Update(ctx context.Context, site Site) (Site, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// BuildingRepository persists Buildings.
type BuildingRepository interface {
	Get(ctx context.Context, id uuid.UUID) (Building, error)
	List(ctx context.Context) ([]Building, error)
	Create(ctx context.Context, building Building) (Building, error)
	Update(ctx context.Context, building Building) (Building, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// RoomRepository persists Rooms.
type RoomRepository interface {
	Get(ctx context.Context, id uuid.UUID) (Room, error)
	List(ctx context.Context) ([]Room, error)
	Create(ctx context.Context, room Room) (Room, error)
	Update(ctx context.Context, room Room) (Room, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// RackRepository persists Racks.
type RackRepository interface {
	Get(ctx context.Context, id uuid.UUID) (Rack, error)
	List(ctx context.Context) ([]Rack, error)
	Create(ctx context.Context, rack Rack) (Rack, error)
	Update(ctx context.Context, rack Rack) (Rack, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// DeviceRepository persists Devices.
type DeviceRepository interface {
	Get(ctx context.Context, id uuid.UUID) (Device, error)
	List(ctx context.Context) ([]Device, error)
	Create(ctx context.Context, device Device) (Device, error)
	Update(ctx context.Context, device Device) (Device, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
