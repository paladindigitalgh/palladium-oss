package provisioning

import (
	"context"

	"github.com/google/uuid"
)

// ProvisioningRepository persists ProvisioningJobs. Get, List, Create,
// Update, and Delete follow the exact shape of every other repository in
// this codebase (see e.g. serviceequipment.ServiceEquipmentRepository):
// Create and Update return the persisted entity so a caller sees
// anything the store sets (e.g. timestamps) without a second read.
//
// ListByServiceID is this domain's one addition, mirroring
// serviceequipment.ServiceEquipmentRepository.GetActiveByDeviceID's
// precedent for a query shaped around one specific need — here, "every
// ProvisioningJob that has ever existed for this Service," the natural
// way to look up a Service's provisioning history.
//
// The repository is responsible only for persistence — no business
// rules, including no state-transition enforcement. Update persists
// whatever ProvisioningJob it is given, trusting its caller completely,
// the same as every other repository's Update in this codebase.
// internal/provisioning/service.ProvisioningService is where transition
// rules are enforced, before Update is ever called.
//
// Nothing in this package implements ProvisioningRepository — no SQL, no
// migrations — so the domain has zero dependency on any storage
// technology. A concrete implementation (internal/provisioning/postgres)
// satisfies it.
type ProvisioningRepository interface {
	Get(ctx context.Context, id uuid.UUID) (ProvisioningJob, error)
	List(ctx context.Context) ([]ProvisioningJob, error)
	ListByServiceID(ctx context.Context, serviceID uuid.UUID) ([]ProvisioningJob, error)
	Create(ctx context.Context, job ProvisioningJob) (ProvisioningJob, error)
	Update(ctx context.Context, job ProvisioningJob) (ProvisioningJob, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
