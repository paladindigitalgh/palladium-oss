// Package connectors defines the abstraction internal/provisioning/engine
// orchestrates through, and nothing else. It contains no network
// communication, no vendor SDKs, and no concrete GenieACS, MikroTik, or
// Kontron implementation — those are explicitly out of scope for this
// milestone (see docs/ARCHITECTURE.md's Plugin Philosophy: "Everything
// vendor-specific belongs in plugins. The core system must never contain
// vendor-specific logic."). This package is the "well-defined interface"
// that philosophy calls for; a future milestone builds real connectors
// against it, entirely outside this package.
package connectors

import (
	"context"
	"errors"

	"github.com/paladindigitalgh/palladium-oss/internal/service"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// ErrNotImplemented is the sentinel a Connector returns from any of the
// six operation methods it does not support. Goal 2 explicitly allows
// this ("methods may return a 'not implemented' error") because a real
// connector is expected to support only the operations meaningful for
// its equipment — e.g. a WiFi access point connector plausibly has
// nothing useful to do for Reprovision — and forcing every connector to
// implement all six with real behavior would invent work this milestone
// does not ask for. It is a plain error, not an *apperror.Error: this
// package models an operational/execution boundary (talking to external
// systems), not an HTTP-facing API boundary, so it has no reason to carry
// apperror's Kind taxonomy — internal/provisioning/engine, which does
// sit closer to that boundary, is responsible for deciding how a
// Connector error should ultimately be reported (see its own doc
// comments).
var ErrNotImplemented = errors.New("connectors: operation not implemented")

// Request is what internal/provisioning/engine passes to every Connector
// method: the Service being acted on and the single piece of Service
// Equipment this particular call concerns. The engine calls a Connector
// once per active ServiceEquipment record for a Service (see
// internal/provisioning/engine's doc comment on why), so Request
// intentionally carries one Equipment, not a slice — a connector method
// answers "what do I do for this one device," never "what do I do for
// this whole service," which keeps a future real connector's
// implementation simple: it never has to loop internally to figure out
// which of several devices in a batch it actually owns.
//
// This is a small, purpose-built struct rather than passing Service and
// ServiceEquipment as separate parameters mainly for the same reason
// every DTO/request type in this codebase exists — a stable, named
// shape a future connector's method signature does not need to change
// merely because the engine starts passing one more piece of context.
type Request struct {
	Service   service.Service
	Equipment serviceequipment.ServiceEquipment
}

// Connector is the abstraction every external provisioning system will
// eventually implement — a GenieACS-backed CPE connector, a Kontron OLT
// connector, a MikroTik router connector, and so on (see this package's
// doc comment for why none of those exist yet). Each method corresponds
// exactly to one provisioning.ProvisioningOperation value (see
// internal/provisioning/operation.go); internal/provisioning/engine is
// responsible for calling the one method matching a given
// ProvisioningJob's Operation — this interface has no method that takes
// an Operation and dispatches internally, because "which method runs" is
// an orchestration decision belonging to the engine, not something every
// connector implementation should have to re-derive.
//
// No method returns anything beyond error: goal 2 is explicit that this
// milestone performs "no network communication," so there is nothing yet
// for a connector to hand back beyond success or failure. A future
// milestone that adds real connectors may need to widen these signatures
// (e.g. to return a vendor-assigned identifier) — deliberately not
// designed for here, per CLAUDE.md's guidance against building for
// hypothetical future requirements.
type Connector interface {
	// Name identifies this Connector for registration and lookup (see
	// Registry) and for inclusion in error messages and audit trails.
	Name() string

	Provision(ctx context.Context, req Request) error
	Reprovision(ctx context.Context, req Request) error
	Suspend(ctx context.Context, req Request) error
	Resume(ctx context.Context, req Request) error
	Disconnect(ctx context.Context, req Request) error
	Synchronize(ctx context.Context, req Request) error
}
