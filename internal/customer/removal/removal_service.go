// Package removal implements "Remove Customer": a standalone, one-shot
// admin action — like internal/provisioning/kontron's AuthorizeONU/
// DeauthorizeONU/Discover-ONU flows — not a Service-lifecycle Plugin/
// workflow action. It never creates a workflow.Instance or writes an
// event.Event, for the same reason those three don't: this is an
// operator-triggered, synchronous action needing immediate feedback, not
// something else in the system also triggers.
//
// "Remove" does not delete anything. customer.Customer, location.Location,
// and service.Service have no soft-delete concept of their own, but they
// each already have a status value built for exactly this: Customer's
// StatusArchived, Location's StatusInactive, and Service's
// StatusDisconnected (already the "permanently ended" terminal state —
// see internal/service/status.go). Using those, rather than inventing a
// new RemovedAt column, means workflow_instances.service_id's ON DELETE
// RESTRICT is never even in play: nothing is ever deleted, so nothing is
// ever blocked, and every Service's workflow history stays exactly where
// it is. ServiceEquipment and AccessAttachment already have a RemovedAt
// concept (see those packages) — this reuses it, matching how "Delete
// ONU" (internal/provisioning/kontron/service.DeauthorizationService)
// unassigns equipment without ever touching the underlying
// inventory.Device.
//
// Execute is deliberately bottom-up and idempotent: a Location already
// StatusInactive, a Service already StatusDisconnected, or equipment
// already RemovedAt is skipped, not re-processed. There is no cross-
// repository transaction anywhere in this codebase (see
// olt/service.OLTService.Create's own doc comment on the same
// limitation), so a failure partway through — most likely an
// unreachable OLT — leaves everything processed so far in its new state
// and returns the error; calling Execute again continues from there
// rather than starting over.
package removal

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessattachment"
	"github.com/paladindigitalgh/palladium-oss/internal/customer"
	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
	"github.com/paladindigitalgh/palladium-oss/internal/location"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/service"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// customerRepository is the seam RemovalService depends on instead of
// the full customer.CustomerRepository.
type customerRepository interface {
	Get(ctx context.Context, id uuid.UUID) (customer.Customer, error)
	Update(ctx context.Context, c customer.Customer) (customer.Customer, error)
}

// locationRepository is the seam RemovalService depends on instead of
// the full location.LocationRepository.
type locationRepository interface {
	ListByCustomerID(ctx context.Context, customerID uuid.UUID) ([]location.Location, error)
	Update(ctx context.Context, l location.Location) (location.Location, error)
}

// serviceRepository is the seam RemovalService depends on instead of the
// full service.ServiceRepository.
type serviceRepository interface {
	ListByLocationID(ctx context.Context, locationID uuid.UUID) ([]service.Service, error)
	Update(ctx context.Context, svc service.Service) (service.Service, error)
}

// deviceGetter is the seam RemovalService depends on instead of the full
// inventory.DeviceRepository — used only to show a human-readable device
// name in Preview, never in Execute.
type deviceGetter interface {
	Get(ctx context.Context, id uuid.UUID) (inventory.Device, error)
}

// equipmentLister is the seam RemovalService depends on instead of the
// full serviceequipment.ServiceEquipmentRepository.
type equipmentLister interface {
	ListActiveByServiceID(ctx context.Context, serviceID uuid.UUID) ([]serviceequipment.ServiceEquipment, error)
}

// equipmentUpdater is equipmentLister's write-side counterpart, satisfied
// by the real *serviceequipmentservice.ServiceEquipmentService rather
// than the raw repository, so its existing uniqueness-check logic runs
// on this write exactly as it does on every other caller's — the same
// reasoning internal/provisioning/kontron/service.DeauthorizationService
// gives for depending on the Service, not the repository, for this exact
// write. In practice that check never fires here: it only runs while
// e.Active(), and this always sets RemovedAt in the same call.
type equipmentUpdater interface {
	Update(ctx context.Context, e serviceequipment.ServiceEquipment) (serviceequipment.ServiceEquipment, error)
}

// attachmentGetter is the seam RemovalService depends on instead of the
// full accessattachment.AccessAttachmentRepository.
type attachmentGetter interface {
	GetActiveByServiceEquipmentID(ctx context.Context, serviceEquipmentID uuid.UUID) (accessattachment.AccessAttachment, error)
}

// attachmentUpdater is attachmentGetter's write-side counterpart,
// satisfied by the real *accessattachmentservice.AccessAttachmentService
// for the same reason equipmentUpdater is.
type attachmentUpdater interface {
	Update(ctx context.Context, a accessattachment.AccessAttachment) (accessattachment.AccessAttachment, error)
}

// serviceProfileRemover is the seam RemovalService depends on instead of
// the concrete *provisioningkontronservice.ServiceProfileService — the
// real OLT-side teardown ("the ONU service removal process").
type serviceProfileRemover interface {
	Remove(ctx context.Context, svc service.Service, equipment serviceequipment.ServiceEquipment) (string, error)
}

// RemovalService removes a Customer: see the package doc comment for
// what "remove" means here.
type RemovalService struct {
	customers      customerRepository
	locations      locationRepository
	services       serviceRepository
	devices        deviceGetter
	equipment      equipmentLister
	equipmentSvc   equipmentUpdater
	attachments    attachmentGetter
	attachmentsSvc attachmentUpdater
	profiles       serviceProfileRemover
	clock          clock.Clock
}

// NewRemovalService builds a RemovalService.
func NewRemovalService(
	customers customerRepository,
	locations locationRepository,
	services serviceRepository,
	devices deviceGetter,
	equipment equipmentLister,
	equipmentSvc equipmentUpdater,
	attachments attachmentGetter,
	attachmentsSvc attachmentUpdater,
	profiles serviceProfileRemover,
	c clock.Clock,
) *RemovalService {
	return &RemovalService{
		customers: customers, locations: locations, services: services, devices: devices,
		equipment: equipment, equipmentSvc: equipmentSvc,
		attachments: attachments, attachmentsSvc: attachmentsSvc,
		profiles: profiles, clock: c,
	}
}

// isONUOrONT reports whether role is one this action ever runs a real OLT
// command for — every other role (Router, Gateway, WiFiAccessPoint, UPS,
// Other) is just unassigned, the same distinction
// provisioningkontronservice.ServiceProfileService's own run method
// draws.
func isONUOrONT(role serviceequipment.EquipmentRole) bool {
	return role == serviceequipment.EquipmentRoleONU || role == serviceequipment.EquipmentRoleONT
}

// hasAppliedProfile reports whether status implies a Kontron service-
// profile is currently applied and therefore needs removing — Active and
// Suspended are the only two ServiceStatus values ProvisionService/
// ResumeService/SuspendService ever leave a Service in with a profile
// still on the ONU (see internal/provisioning/kontron/plugin's own doc
// comment).
func hasAppliedProfile(status service.ServiceStatus) bool {
	return status == service.ServiceStatusActive || status == service.ServiceStatusSuspended
}

// EquipmentPreview describes one active ServiceEquipment record that
// Execute would unassign.
type EquipmentPreview struct {
	ServiceEquipmentID uuid.UUID
	DeviceID           uuid.UUID
	DeviceName         string
	Role               serviceequipment.EquipmentRole
	WillRunOLTTeardown bool
}

// ServicePreview describes one Service that Execute would disconnect.
type ServicePreview struct {
	ServiceID   uuid.UUID
	Description string
	Status      service.ServiceStatus
	Equipment   []EquipmentPreview
}

// LocationPreview describes one Location that Execute would deactivate.
type LocationPreview struct {
	LocationID uuid.UUID
	Name       string
	Status     location.LocationStatus
	Services   []ServicePreview
}

// Preview describes everything Execute would do for one Customer,
// without changing anything.
type Preview struct {
	CustomerID uuid.UUID
	Locations  []LocationPreview
}

// Preview reports what Execute would do for customerID, read-only.
func (s *RemovalService) Preview(ctx context.Context, customerID uuid.UUID) (Preview, error) {
	if _, err := s.customers.Get(ctx, customerID); err != nil {
		return Preview{}, err
	}

	locations, err := s.locations.ListByCustomerID(ctx, customerID)
	if err != nil {
		return Preview{}, err
	}

	preview := Preview{CustomerID: customerID}
	for _, loc := range locations {
		services, err := s.services.ListByLocationID(ctx, loc.ID)
		if err != nil {
			return Preview{}, err
		}

		locPreview := LocationPreview{LocationID: loc.ID, Name: loc.Name, Status: loc.Status}
		for _, svc := range services {
			equipment, err := s.equipment.ListActiveByServiceID(ctx, svc.ID)
			if err != nil {
				return Preview{}, err
			}

			svcPreview := ServicePreview{ServiceID: svc.ID, Description: svc.Description, Status: svc.Status}
			for _, eq := range equipment {
				device, err := s.devices.Get(ctx, eq.DeviceID)
				if err != nil {
					return Preview{}, err
				}

				willTeardown := false
				if isONUOrONT(eq.Role) && hasAppliedProfile(svc.Status) {
					if _, err := s.attachments.GetActiveByServiceEquipmentID(ctx, eq.ID); err == nil {
						willTeardown = true
					}
				}

				svcPreview.Equipment = append(svcPreview.Equipment, EquipmentPreview{
					ServiceEquipmentID: eq.ID,
					DeviceID:           eq.DeviceID,
					DeviceName:         device.Name,
					Role:               eq.Role,
					WillRunOLTTeardown: willTeardown,
				})
			}
			locPreview.Services = append(locPreview.Services, svcPreview)
		}
		preview.Locations = append(preview.Locations, locPreview)
	}
	return preview, nil
}

// Execute removes customerID — see the package doc comment.
func (s *RemovalService) Execute(ctx context.Context, customerID uuid.UUID) error {
	cust, err := s.customers.Get(ctx, customerID)
	if err != nil {
		return err
	}

	locations, err := s.locations.ListByCustomerID(ctx, customerID)
	if err != nil {
		return err
	}

	for _, loc := range locations {
		if loc.Status == location.LocationStatusInactive {
			continue
		}
		if err := s.removeLocation(ctx, loc); err != nil {
			return err
		}
	}

	if cust.Status == customer.CustomerStatusArchived {
		return nil
	}
	cust.Status = customer.CustomerStatusArchived
	_, err = s.customers.Update(ctx, cust)
	return err
}

// removeLocation disconnects and unassigns every Service at loc, then
// deactivates loc itself.
func (s *RemovalService) removeLocation(ctx context.Context, loc location.Location) error {
	services, err := s.services.ListByLocationID(ctx, loc.ID)
	if err != nil {
		return err
	}

	for _, svc := range services {
		equipment, err := s.equipment.ListActiveByServiceID(ctx, svc.ID)
		if err != nil {
			return err
		}
		if err := s.unassignEquipment(ctx, svc, equipment); err != nil {
			return err
		}

		if svc.Status == service.ServiceStatusDisconnected {
			continue
		}
		now := s.clock.Now()
		svc.Status = service.ServiceStatusDisconnected
		svc.DisconnectedAt = &now
		if _, err := s.services.Update(ctx, svc); err != nil {
			return err
		}
	}

	loc.Status = location.LocationStatusInactive
	_, err = s.locations.Update(ctx, loc)
	return err
}

// unassignEquipment runs the real OLT teardown for each active
// ServiceEquipment in equipment that currently needs one, then marks its
// AccessAttachment (if any) and the ServiceEquipment itself removed. The
// underlying inventory.Device is never touched.
func (s *RemovalService) unassignEquipment(ctx context.Context, svc service.Service, equipment []serviceequipment.ServiceEquipment) error {
	now := s.clock.Now()
	for _, eq := range equipment {
		attachment, err := s.attachments.GetActiveByServiceEquipmentID(ctx, eq.ID)
		hasAttachment := err == nil

		if hasAttachment && isONUOrONT(eq.Role) && hasAppliedProfile(svc.Status) {
			if _, err := s.profiles.Remove(ctx, svc, eq); err != nil {
				return err
			}
		}

		if hasAttachment {
			attachment.RemovedAt = &now
			attachment.RemovalReason = "Customer removed"
			if _, err := s.attachmentsSvc.Update(ctx, attachment); err != nil {
				return err
			}
		}

		eq.RemovedAt = &now
		if _, err := s.equipmentSvc.Update(ctx, eq); err != nil {
			return err
		}
	}
	return nil
}
