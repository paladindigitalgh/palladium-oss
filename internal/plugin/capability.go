package plugin

// Capability names one discrete operation a Plugin can perform against a
// Resource. Palladium's core never invokes vendor-specific functions
// directly — it asks a Registry which Plugin provides a Capability (see
// docs/06-PLUGIN-ARCHITECTURE.md, "Capability Model") and calls that
// Plugin's Execute with it.
//
// These six values are a direct port of the former
// provisioning.ProvisioningOperation enum: the same six service-lifecycle
// operations, regrounded as plugin capabilities rather than a
// provisioning-specific concept.
type Capability string

const (
	ProvisionService   Capability = "ProvisionService"
	ReprovisionService Capability = "ReprovisionService"
	SuspendService     Capability = "SuspendService"
	ResumeService      Capability = "ResumeService"
	DisconnectService  Capability = "DisconnectService"
	SynchronizeService Capability = "SynchronizeService"
)
