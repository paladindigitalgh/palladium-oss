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
// ListByLocationID is this package's one addition beyond that standard
// shape, added for internal/accesstopology, one hop past
// location.LocationRepository.ListByCustomerID: once a Customer's
// Locations are known, finding which Services were purchased at each
// one is the next step in resolving where that Customer's equipment
// sits on the access network, and no existing method answers that
// directly. It follows serviceequipment.ServiceEquipmentRepository.
// ListActiveByServiceID's own precedent — a query shaped around one
// specific, already-real need. Unlike that method, this one has no
// "active" filter of its own: it returns every Service at a Location
// regardless of ServiceStatus, the same reasoning
// location.LocationRepository.ListByCustomerID gives for not filtering
// by Location.Status — a future caller filtering by Status is free to,
// this method does not decide that for them.
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
	ListByLocationID(ctx context.Context, locationID uuid.UUID) ([]Service, error)
}
