// Package provisioning models Palladium's Provisioning Job domain (v1):
// the orchestration record for a request to provision, modify, suspend,
// resume, disconnect, or synchronize a Service (see internal/service) —
// and the lifecycle that record moves through. This package holds only
// the domain model, field validation, and the repository interface — no
// SQL, no migrations, no HTTP CRUD — mirroring internal/serviceequipment's
// own package exactly.
//
// This package does not import internal/service or internal/auth.
// ServiceID and RequestedByUserID are bare uuid.UUID / *uuid.UUID values,
// not references to service.Service or auth.User: the foreign keys to
// services(id) and users(id) are database concepts, enforced by
// internal/provisioning/postgres and its migration, not Go package
// dependencies — the same reasoning internal/serviceequipment/model.go
// documents for why it does not import internal/service or
// internal/inventory.
//
// A ProvisioningJob is purely the orchestration record and its lifecycle.
// Per this milestone's explicit scope, it is not, and never becomes:
//
//   - a background worker, a queue, or a scheduler: nothing here executes
//     anything or decides when work runs. A future execution layer (see
//     docs/ARCHITECTURE.md's Provisioning and Workflow Engine domains)
//     will read and write ProvisioningJob rows, not the other way around.
//   - a connector to GenieACS, MikroTik, a Kontron OLT, DHCP, DNS, or
//     IPAM: this package has no knowledge any of those systems exist.
//   - automation: RetryCount is stored, never interpreted — no backoff,
//     no automatic re-scheduling, no idempotency handling (see
//     internal/provisioning/service's package doc comment for exactly
//     what does and does not touch it).
//
// Everything above is a real feature some future milestone will add. None
// of it is implied by what exists today.
package provisioning

import (
	"time"

	"github.com/google/uuid"
)

// ProvisioningJob is a request to perform some Operation against a
// Service, tracked through a lifecycle (see ProvisioningStatus in
// status.go).
//
// RequestedByUserID is *uuid.UUID, not uuid.UUID: it is nullable — a
// ProvisioningJob created by a future automated process (this milestone
// implements none) would have no human requester to record, and forcing
// a value would misrepresent that. ErrorMessage, StartedAt, and
// CompletedAt are each nullable for the same reason
// internal/serviceequipment/model.go gives for InstalledAt/RemovedAt:
// they only ever apply once the job has actually reached the relevant
// point in its lifecycle, and none of their zero values (empty string,
// 0001-01-01) can safely stand in for "not yet set" without risking that
// being mistaken for real data.
type ProvisioningJob struct {
	ID                uuid.UUID
	ServiceID         uuid.UUID
	RequestedByUserID *uuid.UUID
	Operation         ProvisioningOperation
	Status            ProvisioningStatus
	RetryCount        int
	ErrorMessage      *string
	StartedAt         *time.Time
	CompletedAt       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
