// Package onuauthorization models Palladium's record of a Device
// (internal/inventory) being authorized directly on an OLT (internal/olt)
// at a specific interface, independent of any Service assignment.
//
// Before this package existed, "where is this Device authorized on the
// network" was only ever derived on the fly from a Service-anchored
// chain: ServiceEquipment (internal/serviceequipment) links a Service to
// a Device, AccessAttachment (internal/accessattachment) links that
// ServiceEquipment to an AccessInterface, and accesstopology.Resolver
// walks that chain back out to an OLTID/Interface pair. That chain
// cannot represent a Device that has been authorized on an OLT but not
// yet sold to any Customer — ServiceEquipment.ServiceID is required (see
// that package's own doc comment: "the link between a subscriber's
// purchased Service ... and the ... Device"), so a bare authorized
// Device has nowhere in that graph to sit. In practice this meant such a
// Device could be authorized once (see
// internal/provisioning/kontron/service.AuthorizeAndCreateDeviceService)
// but never deauthorized again through Palladium: DeauthorizationService
// had no way to learn which OLT/interface to run the command against.
//
// This package holds only the domain model, field validation, and the
// repository interface — no SQL, no migrations, no HTTP CRUD — mirroring
// internal/accessattachment's own package exactly. It does not import
// internal/inventory or internal/olt: DeviceID and OLTID are bare
// uuid.UUID values, the same reasoning internal/serviceequipment/model.go
// gives for not importing internal/service or internal/inventory.
//
// This is deliberately not folded into inventory.Device itself: OLTID
// and Interface describe the Device's current network position, not
// what the Device physically is — the exact same separation CLAUDE.md's
// Core Philosophy draws between inventory and the customers/services
// consuming it ("Resources exist independently of Customers"), applied
// here to network position instead of ownership. Interface is a bare
// string, not a foreign key to access_interfaces(id), mirroring
// accesstopology.Location's own Interface field exactly: it is the raw
// interface path a vendor plugin's SSH client expects verbatim (e.g.
// "xgs/6/2"), not the higher-level, operator-administered
// AccessInterface concept.
package onuauthorization

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// OnuAuthorization records that DeviceID has been authorized on OLTID at
// Interface.
//
// AuthorizedAt is always set (the moment the OLT-side authorize command
// succeeded). DeauthorizedAt is nil until the matching deauthorize
// command succeeds — the same *time.Time / nil-means-active pattern
// serviceequipment.ServiceEquipment.RemovedAt and
// accessattachment.AccessAttachment.RemovedAt already establish, for the
// identical reason: the zero value of time.Time is a real instant, so it
// cannot double as "not yet deauthorized."
type OnuAuthorization struct {
	ID             uuid.UUID
	DeviceID       uuid.UUID
	OLTID          uuid.UUID
	Interface      string
	AuthorizedAt   time.Time
	DeauthorizedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

// Active reports whether this authorization is still in effect on the
// OLT: it has not been deauthorized. Mirrors
// accessattachment.AccessAttachment.Active /
// serviceequipment.ServiceEquipment.Active exactly.
func (a OnuAuthorization) Active() bool {
	return a.DeauthorizedAt == nil
}

// Repository persists OnuAuthorization records. It is deliberately
// narrower than a typical domain's repository (e.g.
// accessattachment.AccessAttachmentRepository's full Get/List/Create/
// Update/Delete): nothing browses OnuAuthorization records directly
// through the UI today — this exists purely to let
// DeauthorizationService resolve a bare authorized Device's OLT/interface
// later, so it exposes only what that round trip needs.
//
// GetActiveByDeviceID returns an apperror.KindNotFound error when
// deviceID has no active authorization — the expected, common case for
// any Device never authorized this way (created manually, or already
// deauthorized), not an exceptional one. Mirrors
// accessattachment.AccessAttachmentRepository.GetActiveByServiceEquipmentID.
type Repository interface {
	Create(ctx context.Context, authorization OnuAuthorization) (OnuAuthorization, error)
	Update(ctx context.Context, authorization OnuAuthorization) (OnuAuthorization, error)
	GetActiveByDeviceID(ctx context.Context, deviceID uuid.UUID) (OnuAuthorization, error)
}
