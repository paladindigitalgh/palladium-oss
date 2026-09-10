// Package service is the Service domain's business logic layer. It sits
// between the HTTP layer and the repository layer: HTTP handlers never
// call a repository directly (see internal/service/httpapi), and
// repositories never validate or otherwise reason about business rules
// (see internal/service/postgres, which trusts its caller) — this is
// where those two responsibilities meet. It mirrors
// internal/location/service exactly.
//
// This package's import path, internal/service/service, nests a package
// named "service" inside a domain package also named "service"
// (internal/service, home to the Service struct itself). That is not a
// naming mistake: it is the exact mechanical extension of the same
// convention every other domain in this codebase already follows — the
// business logic layer for internal/location lives at
// internal/location/service, for internal/catalog at
// internal/catalog/service, and so on. The Service domain's business
// logic layer landing at internal/service/service is what applying that
// convention consistently produces; special-casing it to avoid the
// repeated word would be the actual inconsistency.
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	domainservice "github.com/paladindigitalgh/palladium-oss/internal/service"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// activeEquipmentLister is the seam ServiceService depends on instead of
// the full serviceequipment.ServiceEquipmentRepository — used only by
// Delete, to give a clear, specific error when a Service still has
// equipment attached, rather than surfacing the database's own generic
// foreign-key-violation message (see internal/service/postgres/errors.go's
// translateError). serviceequipment.ServiceEquipmentRepository and
// *serviceequipmentservice.ServiceEquipmentService both already satisfy
// this exactly.
type activeEquipmentLister interface {
	ListActiveByServiceID(ctx context.Context, serviceID uuid.UUID) ([]serviceequipment.ServiceEquipment, error)
}

// ServiceService is the Service domain's business logic.
//
// It depends on domainservice.ServiceRepository and, for Delete's two
// preconditions (see that method's own doc comment), activeEquipmentLister
// — not clock.Clock, for the same reason internal/location/service.LocationService
// does not: timestamps are already the repository's responsibility, and
// this service has no business rule that needs to reason about "now".
type ServiceService struct {
	services  domainservice.ServiceRepository
	equipment activeEquipmentLister
}

// NewServiceService builds a ServiceService.
func NewServiceService(services domainservice.ServiceRepository, equipment activeEquipmentLister) *ServiceService {
	return &ServiceService{services: services, equipment: equipment}
}

// Get retrieves a Service by ID.
func (s *ServiceService) Get(ctx context.Context, id uuid.UUID) (domainservice.Service, error) {
	return s.services.Get(ctx, id)
}

// List returns every Service.
func (s *ServiceService) List(ctx context.Context) ([]domainservice.Service, error) {
	return s.services.List(ctx)
}

// Create validates svc and, if valid, persists it.
//
// Validation happens here — not in the repository, which trusts its
// caller, and not in the HTTP handler, which would then need to duplicate
// this for every other future caller of ServiceService — so every caller
// gets the same guarantee for free, and invalid input never costs a
// database round trip. See internal/location/service.LocationService.Create
// for the identical reasoning applied to Locations.
func (s *ServiceService) Create(ctx context.Context, svc domainservice.Service) (domainservice.Service, error) {
	if err := svc.Validate(); err != nil {
		return domainservice.Service{}, err
	}
	return s.services.Create(ctx, svc)
}

// Update validates svc and, if valid, persists the change. See Create for
// why validation happens here rather than elsewhere.
func (s *ServiceService) Update(ctx context.Context, svc domainservice.Service) (domainservice.Service, error) {
	if err := svc.Validate(); err != nil {
		return domainservice.Service{}, err
	}
	return s.services.Update(ctx, svc)
}

// Delete removes the Service identified by id.
//
// Delete is a hard delete for disposable records — real operational
// history is meant to go through Suspend/Disconnect (the real Workflow-
// driven OLT teardown, internal/provisioning/kontron/plugin) or Remove
// Customer's cascade (internal/customer/removal.RemovalService), neither
// of which deletes anything. Two preconditions are checked explicitly
// here, before ever touching the repository, so the operator gets a
// clear reason rather than a generic database error:
//
//   - The Service must not still imply an applied Kontron service-
//     profile (svc.Status.HasAppliedProfile(), i.e. Active — see that
//     method's own doc comment on why Suspended does not count too).
//     Delete never runs the real OLT command Suspend/Disconnect/Remove
//     Customer's cascade would — deleting an Active Service's records
//     here would silently leave that service-profile applied on the
//     real ONU with no Palladium record of it left to ever clean it up.
//     This is exactly the gap a live OLT test surfaced this session: an
//     operator using this action as a substitute for Disconnect Service
//     left a stale service-profile behind.
//   - The Service must have no active ServiceEquipment. The database
//     already enforces this (service_equipment.service_id references
//     services(id) ON DELETE RESTRICT), but checking it here first turns
//     that into a specific "still has equipment attached" message
//     instead of the generic foreign-key-violation one translateError
//     would otherwise produce.
func (s *ServiceService) Delete(ctx context.Context, id uuid.UUID) error {
	svc, err := s.services.Get(ctx, id)
	if err != nil {
		return err
	}
	if svc.Status.HasAppliedProfile() {
		return apperror.Conflict(fmt.Sprintf(
			"this service is still %s -- suspend or disconnect it first so its configuration is removed from the OLT before deleting the record",
			svc.Status))
	}

	equipment, err := s.equipment.ListActiveByServiceID(ctx, id)
	if err != nil {
		return err
	}
	if len(equipment) > 0 {
		return apperror.Conflict("this service still has equipment attached -- remove it first")
	}

	return s.services.Delete(ctx, id)
}
