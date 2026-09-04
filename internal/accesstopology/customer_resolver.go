package accesstopology

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/location"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/service"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// CustomerLocation pairs one of a Customer's currently-attached pieces
// of ServiceEquipment with where it resolves to on the access network.
type CustomerLocation struct {
	ServiceEquipmentID uuid.UUID
	Location           Location
}

// customerLocationsGetter, locationServicesGetter, and
// activeServiceEquipmentGetter are the seams CustomerResolver (and
// NewCustomerResolver itself) depend on instead of the three full
// repository interfaces — the same narrowing pattern resolver.go's own
// attachmentGetter/interfaceGetter/portGetter already establish in this
// package.
type customerLocationsGetter interface {
	ListByCustomerID(ctx context.Context, customerID uuid.UUID) ([]location.Location, error)
}

type locationServicesGetter interface {
	ListByLocationID(ctx context.Context, locationID uuid.UUID) ([]service.Service, error)
}

type activeServiceEquipmentGetter interface {
	ListActiveByServiceID(ctx context.Context, serviceID uuid.UUID) ([]serviceequipment.ServiceEquipment, error)
}

// equipmentLocator is the seam CustomerResolver depends on for the
// final, per-equipment hop — satisfied by *Resolver's own Locate method,
// declared as an interface here so this file's own tests can substitute
// a fake and focus purely on the Customer -> Location -> Service ->
// ServiceEquipment fan-out this type adds, without re-exercising
// Resolver's three-hop chain a second time (already covered by
// resolver_test.go).
type equipmentLocator interface {
	Locate(ctx context.Context, serviceEquipmentID uuid.UUID) (Location, error)
}

// CustomerResolver locates every one of a Customer's currently-attached
// pieces of equipment on the access network.
type CustomerResolver struct {
	locations customerLocationsGetter
	services  locationServicesGetter
	equipment activeServiceEquipmentGetter
	locator   equipmentLocator
}

// NewCustomerResolver builds a CustomerResolver. Real callers pass their
// full location.LocationRepository / service.ServiceRepository /
// serviceequipment.ServiceEquipmentRepository implementations directly
// (each already satisfies the narrower interface actually declared here,
// per this file's own doc comment) and a *Resolver (built via
// NewResolver) as locator.
func NewCustomerResolver(
	locations customerLocationsGetter,
	services locationServicesGetter,
	equipment activeServiceEquipmentGetter,
	locator equipmentLocator,
) *CustomerResolver {
	return &CustomerResolver{locations: locations, services: services, equipment: equipment, locator: locator}
}

// LocateForCustomer resolves every currently-locatable piece of
// equipment belonging to the Customer identified by customerID: every
// Location that Customer has, every Service purchased at each, every
// currently-active ServiceEquipment record for each Service, and — for
// each one that currently has an active AccessAttachment — where it sits
// on the access network.
//
// A ServiceEquipment record with no active AccessAttachment (Locate
// returning an apperror.KindNotFound error) is silently skipped, not
// treated as a failure: per Resolver.Locate's own doc comment, that is
// the expected, common state for equipment not yet installed anywhere.
// Any other error — a genuine repository failure at any hop — aborts
// immediately and is returned to the caller, the same fail-fast behavior
// as every other resolver in this codebase.
//
// A Customer with no Locations, no Services, or no active
// ServiceEquipment returns an empty slice and a nil error — this is not
// an error case, it is the ordinary state of a Customer nothing has been
// attached to yet (e.g. a brand-new Customer with no ONU assigned).
func (r *CustomerResolver) LocateForCustomer(ctx context.Context, customerID uuid.UUID) ([]CustomerLocation, error) {
	locations, err := r.locations.ListByCustomerID(ctx, customerID)
	if err != nil {
		return nil, err
	}

	var results []CustomerLocation
	for _, loc := range locations {
		services, err := r.services.ListByLocationID(ctx, loc.ID)
		if err != nil {
			return nil, err
		}

		for _, svc := range services {
			equipment, err := r.equipment.ListActiveByServiceID(ctx, svc.ID)
			if err != nil {
				return nil, err
			}

			for _, eq := range equipment {
				resolved, err := r.locator.Locate(ctx, eq.ID)
				if err != nil {
					if apperror.Is(err, apperror.KindNotFound) {
						continue
					}
					return nil, err
				}
				results = append(results, CustomerLocation{ServiceEquipmentID: eq.ID, Location: resolved})
			}
		}
	}

	return results, nil
}
