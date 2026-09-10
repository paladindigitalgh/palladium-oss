// Package service is the OLT domain's business logic layer. It sits
// between the HTTP layer and the repository layer: HTTP handlers never
// call a repository directly (see internal/olt/httpapi), and
// repositories never validate or otherwise reason about business rules
// (see internal/olt/postgres, which trusts its caller) — this is where
// those two responsibilities meet. It mirrors internal/product/service
// exactly, with one deliberate deviation: Create's auto-port-creation
// (see below).
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/olt"
	"github.com/paladindigitalgh/palladium-oss/internal/oltmodel"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/ponport"
)

// OLTService is the OLT domain's business logic.
//
// Unlike every other domain service in this codebase (each depends on
// exactly one domain's repository — see e.g.
// internal/product/service.ProductService), OLTService depends on three:
// olt.OLTRepository for the OLT itself, oltmodel.OLTModelRepository to
// look up a newly created OLT's PONPortCount, and ponport.PONPortRepository
// to create the resulting ports. This is a deliberate exception, not a
// pattern to copy elsewhere without the same justification: "how many
// PON ports does this OLT have" is data that lives on OLTModel (see
// internal/oltmodel's own doc comment on why it must — CLAUDE.md's
// Plugin Philosophy forbids hardcoding a vendor's chassis line into Go
// code), and "create N port records when an OLT is created" is a real
// business rule, not a vendor-specific decision — an inert, mechanical
// consequence of the OLTModel an operator already chose. Composing two
// repositories to enforce it here has a real precedent one layer up:
// internal/workflow/engine.DefaultEngine already composes
// service.ServiceRepository, serviceequipment.ServiceEquipmentRepository,
// and plugin.Registry for the same reason — a real cross-domain effect
// with nowhere narrower to live. Building a whole workflow (see
// docs/05-WORKFLOW-ENGINE.md) for what is otherwise a synchronous,
// non-resumable data cascade would be the actual overengineering here.
type OLTService struct {
	olts      olt.OLTRepository
	oltModels oltmodel.OLTModelRepository
	ponPorts  ponport.PONPortRepository
}

// NewOLTService builds an OLTService.
func NewOLTService(olts olt.OLTRepository, oltModels oltmodel.OLTModelRepository, ponPorts ponport.PONPortRepository) *OLTService {
	return &OLTService{olts: olts, oltModels: oltModels, ponPorts: ponPorts}
}

// Get retrieves an OLT by ID.
func (s *OLTService) Get(ctx context.Context, id uuid.UUID) (olt.OLT, error) {
	return s.olts.Get(ctx, id)
}

// List returns every OLT.
func (s *OLTService) List(ctx context.Context) ([]olt.OLT, error) {
	return s.olts.List(ctx)
}

// Create validates o, persists it, and then auto-creates PON ports
// 1..N on the new OLT, where N is the referenced OLTModel's
// PONPortCount (see internal/oltmodel.OLTModel) — the whole reason this
// service depends on two repositories beyond its own (see this type's
// own doc comment).
//
// The OLTModel lookup happens after the OLT is persisted, not before:
// o.OLTModelID already passed field-presence validation (see
// internal/olt/validate.go), and the foreign key
// (database/migrations/00033_oltmodel_olt_models.sql) guarantees that
// once the OLT insert itself succeeds, OLTModelID names a real row —
// this call is not re-validating something Create already confirmed,
// only reading data Create needs next.
//
// Port creation is a best-effort cascade, not a single atomic
// transaction spanning two repositories — this codebase's repository
// layer has no cross-repository transaction primitive today, the same
// limitation internal/workflow/engine.DefaultEngine's own Execute
// accepts for its own multi-step, fail-fast execution (see that type's
// doc comment). If the OLTModel lookup or any individual port creation
// fails partway through, the OLT itself is left persisted with
// whichever ports were created before the failure — Create returns the
// error, but does not attempt to delete the OLT or the partial ports it
// already created. An operator seeing this error can inspect the OLT's
// existing PON ports (see internal/ponport) and create any missing ones
// by hand; building automatic rollback here for a failure mode that
// requires two independent repository writes to already be behaving
// atypically (the OLT itself just wrote successfully) would be solving
// a problem no caller has hit yet.
func (s *OLTService) Create(ctx context.Context, o olt.OLT) (olt.OLT, error) {
	if err := o.Validate(); err != nil {
		return olt.OLT{}, err
	}

	created, err := s.olts.Create(ctx, o)
	if err != nil {
		return olt.OLT{}, err
	}

	model, err := s.oltModels.Get(ctx, created.OLTModelID)
	if err != nil {
		return olt.OLT{}, apperror.Internal(
			fmt.Sprintf("olt %s created but its olt model %s could not be loaded to auto-create pon ports", created.ID, created.OLTModelID), err)
	}

	for portNumber := 1; portNumber <= model.PONPortCount; portNumber++ {
		port := ponport.PONPort{OLTID: created.ID, PortNumber: portNumber}
		if err := port.Validate(); err != nil {
			return olt.OLT{}, apperror.Internal(
				fmt.Sprintf("olt %s created but pon port %d failed validation", created.ID, portNumber), err)
		}
		if _, err := s.ponPorts.Create(ctx, port); err != nil {
			return olt.OLT{}, apperror.Internal(
				fmt.Sprintf("olt %s created but pon port %d could not be created", created.ID, portNumber), err)
		}
	}

	return created, nil
}

// Update validates o and, if valid, persists the change. See Create for
// why validation happens here rather than elsewhere. Unlike Create,
// Update never touches PON ports: changing an OLT's OLTModelID after
// creation does not retroactively add or remove port records — that
// would risk silently deleting a port an operator has since wired up to
// real customer equipment.
func (s *OLTService) Update(ctx context.Context, o olt.OLT) (olt.OLT, error) {
	if err := o.Validate(); err != nil {
		return olt.OLT{}, err
	}
	return s.olts.Update(ctx, o)
}

// Delete removes the OLT identified by id, first deleting every PON port
// on it.
//
// This cascade exists for the same reason Create auto-creates PON ports
// in the first place: pon_ports.olt_id is ON DELETE RESTRICT
// (database/migrations/00018_pon_ports.sql), and since Create now gives
// every OLT at least one PON port, without this cascade no OLT could
// ever be deleted at all — not even one just created by mistake, with
// the wrong OLTModel, that nothing has touched yet. Deleting a PON port
// is itself RESTRICTed by access_interfaces.pon_port_id
// (database/migrations/00019_accessinterface_access_interfaces.sql), so
// this only ever succeeds for an OLT whose ports are genuinely unused —
// the first port delete that fails because a real AccessInterface is
// attached aborts the whole operation and returns that conflict, leaving
// both the OLT and every port up to that point deleted. That partial
// result is the same best-effort-cascade tradeoff Create's own doc
// comment already accepts, applied in reverse: an operator who hits this
// conflict is being told, correctly, that this OLT is actually in use.
//
// s.ponPorts.List has no server-side filter by OLTID (see
// ponport.PONPortRepository — nothing in this codebase needs one yet),
// so this filters client-side the same way
// frontend/src/services/ponPorts/ponPortRepository.ts's
// listPONPortsByOLTId already does.
//
// A second, unrelated RESTRICT can also reject the final s.olts.Delete
// call even when this OLT has no PON ports at all:
// onu_authorizations.olt_id (database/migrations/00037_onuauthorization_
// onu_authorizations.sql), added after this cascade was written. Unlike
// PON ports, OnuAuthorization rows are audit history (see
// onuauthorization.Repository's doc comment on why that package
// deliberately exposes no List/Delete) — this method does not, and
// should not, try to clear them, so an OLT that ever had a device
// authorized on it stays undeletable by design until an operator
// resolves that history directly. The error this produces is a generic
// "violates a foreign key relationship" either way (see
// internal/olt/postgres's translateError), so callers must not assume a
// delete conflict on an OLT always means PON ports.
func (s *OLTService) Delete(ctx context.Context, id uuid.UUID) error {
	ports, err := s.ponPorts.List(ctx)
	if err != nil {
		return err
	}
	for _, port := range ports {
		if port.OLTID != id {
			continue
		}
		if err := s.ponPorts.Delete(ctx, port.ID); err != nil {
			return err
		}
	}
	return s.olts.Delete(ctx, id)
}
