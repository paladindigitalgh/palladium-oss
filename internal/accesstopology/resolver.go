// Package accesstopology resolves where a given ServiceEquipment record
// currently sits in the physical access network: which OLT, and which of
// its logical interfaces. It is the same kind of problem
// internal/olt/connect.Dialer already solves one level down (an OLT's
// own ID -> how to reach it) — here, the graph goes the other direction,
// starting from a ServiceEquipment record and ending at the OLT ID and
// interface name a diagnostic needs to run against.
//
// This is deliberately not a new "ONU" domain concept. Per CLAUDE.md's
// domain philosophy ("Customers own Services. Services consume
// Resources."), an ONU is already fully represented by existing domain
// concepts: a Device (internal/inventory) — the physical unit — assigned
// to a Service via ServiceEquipment (internal/serviceequipment), and
// physically wired to an AccessInterface via an AccessAttachment
// (internal/accessattachment). This package is the traversal that reads
// that existing graph back out, not a new place to store anything.
//
// Resolver (this file) starts at a ServiceEquipmentID already in hand,
// the same way connect.Dialer starts at an OLT's ID rather than solving
// "how did the caller get this OLT ID" itself. CustomerResolver
// (customer_resolver.go) is the Customer-scoped counterpart, walking the
// three hops from a Customer down to every currently-active
// ServiceEquipment record it has (Customer -> Location -> Service ->
// ServiceEquipment) before handing each one to a Resolver in turn — the
// product question that chain raises (a Customer can have more than one
// active piece of equipment; CustomerResolver returns all of them,
// leaving it to the caller to decide how many to show or act on) is
// answered there, not here.
package accesstopology

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessattachment"
	"github.com/paladindigitalgh/palladium-oss/internal/accessinterface"
	"github.com/paladindigitalgh/palladium-oss/internal/ponport"
)

// Location identifies where on the access network a piece of equipment
// is currently attached: which OLT, and the name of the logical
// interface it occupies on that OLT (e.g. "xgs/1/1" — the same string
// internal/diagnostics/kontron.Client's per-interface methods expect
// verbatim, with no reformatting).
type Location struct {
	OLTID     uuid.UUID
	Interface string
}

// attachmentGetter, interfaceGetter, and portGetter are the seams
// Resolver (and NewResolver itself) depend on instead of the three full
// repository interfaces — mirroring the exact narrowing pattern
// internal/olt/connect.NewDialer already establishes in this codebase
// for the equivalent OLT -> ConnectionProfile -> Authentication chain:
// Locate only ever fetches one record of each kind. Every concrete
// repository (e.g. a real accessattachment.AccessAttachmentRepository
// implementation) already satisfies its narrower counterpart here
// structurally, so nothing is lost at a real call site — only
// resolver_test.go's fakes, which implement one method each, benefit
// from not also having to stub the rest of each repository's shape.
type attachmentGetter interface {
	GetActiveByServiceEquipmentID(ctx context.Context, serviceEquipmentID uuid.UUID) (accessattachment.AccessAttachment, error)
}

type interfaceGetter interface {
	Get(ctx context.Context, id uuid.UUID) (accessinterface.AccessInterface, error)
}

type portGetter interface {
	Get(ctx context.Context, id uuid.UUID) (ponport.PONPort, error)
}

// Resolver locates a ServiceEquipment record's current position in the
// access network.
type Resolver struct {
	attachments attachmentGetter
	interfaces  interfaceGetter
	ports       portGetter
}

// NewResolver builds a Resolver. Real callers pass their full
// accessattachment.AccessAttachmentRepository /
// accessinterface.AccessInterfaceRepository / ponport.PONPortRepository
// implementations directly — each already satisfies the narrower
// interface NewResolver actually declares, per this file's own doc
// comment on attachmentGetter.
func NewResolver(attachments attachmentGetter, interfaces interfaceGetter, ports portGetter) *Resolver {
	return &Resolver{attachments: attachments, interfaces: interfaces, ports: ports}
}

// Locate resolves the Location of the equipment identified by
// serviceEquipmentID: its current active AccessAttachment, that
// attachment's AccessInterface, and that interface's PONPort — from
// which the OLT ID and interface name are read directly (PONPort.OLTID
// and AccessInterface.Name).
//
// A not-found error from any of the three Get calls is returned exactly
// as that repository produced it (already an apperror.KindNotFound
// error, by this codebase's established convention).
// GetActiveByServiceEquipmentID's own not-found case is the expected,
// common one — equipment with nothing currently attached to it, e.g. an
// ONU sitting in a warehouse rather than installed at a customer's
// premises — not an exceptional one; a caller should be prepared to
// treat it as "no ONU location for this equipment yet," not a bug.
func (r *Resolver) Locate(ctx context.Context, serviceEquipmentID uuid.UUID) (Location, error) {
	attachment, err := r.attachments.GetActiveByServiceEquipmentID(ctx, serviceEquipmentID)
	if err != nil {
		return Location{}, err
	}

	iface, err := r.interfaces.Get(ctx, attachment.AccessInterfaceID)
	if err != nil {
		return Location{}, err
	}

	port, err := r.ports.Get(ctx, iface.PONPortID)
	if err != nil {
		return Location{}, err
	}

	return Location{OLTID: port.OLTID, Interface: iface.Name}, nil
}
