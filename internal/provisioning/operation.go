package provisioning

import "strings"

// ProvisioningOperation classifies what a ProvisioningJob is asking to
// happen to a Service. It is a distinct type, not a raw string,
// following the exact pattern of serviceequipment.EquipmentRole.
type ProvisioningOperation string

// The six defined operations. There is no zero-value/default operation —
// an empty ProvisioningOperation is invalid — so ProvisioningOperation is
// effectively required on every ProvisioningJob (see
// ProvisioningJob.Validate in validate.go).
const (
	ProvisioningOperationProvision   ProvisioningOperation = "Provision"
	ProvisioningOperationReprovision ProvisioningOperation = "Reprovision"
	ProvisioningOperationSuspend     ProvisioningOperation = "Suspend"
	ProvisioningOperationResume      ProvisioningOperation = "Resume"
	ProvisioningOperationDisconnect  ProvisioningOperation = "Disconnect"
	ProvisioningOperationSynchronize ProvisioningOperation = "Synchronize"
)

// provisioningOperationOrder is the authoritative, ordered set of valid
// operations. It backs both Valid and validation error messages so the
// two can never disagree.
var provisioningOperationOrder = []ProvisioningOperation{
	ProvisioningOperationProvision,
	ProvisioningOperationReprovision,
	ProvisioningOperationSuspend,
	ProvisioningOperationResume,
	ProvisioningOperationDisconnect,
	ProvisioningOperationSynchronize,
}

// Valid reports whether o is one of the six defined ProvisioningOperation
// values.
func (o ProvisioningOperation) Valid() bool {
	for _, v := range provisioningOperationOrder {
		if o == v {
			return true
		}
	}
	return false
}

// String implements fmt.Stringer.
func (o ProvisioningOperation) String() string {
	return string(o)
}

// provisioningOperationNames renders the defined operations as a
// comma-separated list, for use in validation error messages.
func provisioningOperationNames() string {
	names := make([]string, len(provisioningOperationOrder))
	for i, o := range provisioningOperationOrder {
		names[i] = string(o)
	}
	return strings.Join(names, ", ")
}
