package provisioning

import "strings"

// ProvisioningStatus is a ProvisioningJob's lifecycle state. It is a
// distinct type, not a raw string, following the exact pattern of
// service.ServiceStatus.
//
// Unlike every prior status type in this codebase, ProvisioningStatus
// also carries a defined transition table (see CanTransitionTo below) —
// this is, per this milestone's explicit framing, "intentionally the
// first domain with a lightweight state machine." The table lives here,
// as a pure, dependency-free method on the type itself, rather than in
// internal/provisioning/service: it is a fact about what
// ProvisioningStatus values mean in relation to each other (the same
// kind of fact Valid below already captures — "is this a status at
// all"), not business logic that needs a repository or a clock to
// evaluate. internal/provisioning/service is where that fact gets
// enforced against a real, persisted ProvisioningJob; this is only the
// rule itself.
type ProvisioningStatus string

// The five defined statuses. There is no zero-value/default status — an
// empty ProvisioningStatus is invalid — so ProvisioningStatus is
// effectively required on every ProvisioningJob (see
// ProvisioningJob.Validate in validate.go).
const (
	ProvisioningStatusPending   ProvisioningStatus = "Pending"
	ProvisioningStatusRunning   ProvisioningStatus = "Running"
	ProvisioningStatusSucceeded ProvisioningStatus = "Succeeded"
	ProvisioningStatusFailed    ProvisioningStatus = "Failed"
	ProvisioningStatusCancelled ProvisioningStatus = "Cancelled"
)

// provisioningStatusOrder is the authoritative, ordered set of valid
// statuses. It backs both Valid and validation error messages so the two
// can never disagree.
var provisioningStatusOrder = []ProvisioningStatus{
	ProvisioningStatusPending,
	ProvisioningStatusRunning,
	ProvisioningStatusSucceeded,
	ProvisioningStatusFailed,
	ProvisioningStatusCancelled,
}

// Valid reports whether s is one of the five defined ProvisioningStatus
// values.
func (s ProvisioningStatus) Valid() bool {
	for _, v := range provisioningStatusOrder {
		if s == v {
			return true
		}
	}
	return false
}

// String implements fmt.Stringer.
func (s ProvisioningStatus) String() string {
	return string(s)
}

// provisioningStatusNames renders the defined statuses as a
// comma-separated list, for use in validation error messages.
func provisioningStatusNames() string {
	names := make([]string, len(provisioningStatusOrder))
	for i, s := range provisioningStatusOrder {
		names[i] = string(s)
	}
	return strings.Join(names, ", ")
}

// provisioningTransitions is the authoritative set of allowed
// ProvisioningStatus transitions, exactly as this milestone specifies:
//
//	Pending   -> Running, Cancelled
//	Running   -> Succeeded, Failed, Cancelled
//	Failed    -> Pending
//	Succeeded -> (terminal: no outgoing transitions)
//	Cancelled -> (terminal: no outgoing transitions)
//
// Succeeded and Cancelled have no entry below (an absent key and a
// present key mapping to an empty slice behave identically for
// CanTransitionTo's lookup, so the terminal statuses are simply omitted
// rather than written out as empty entries).
var provisioningTransitions = map[ProvisioningStatus][]ProvisioningStatus{
	ProvisioningStatusPending: {ProvisioningStatusRunning, ProvisioningStatusCancelled},
	ProvisioningStatusRunning: {ProvisioningStatusSucceeded, ProvisioningStatusFailed, ProvisioningStatusCancelled},
	ProvisioningStatusFailed:  {ProvisioningStatusPending},
}

// CanTransitionTo reports whether moving from s to target is one of this
// milestone's allowed transitions. It says nothing about whether such a
// move is happening right now — it is a pure fact about the state
// machine's shape, the same way Valid is a pure fact about which values
// exist at all. internal/provisioning/service.ProvisioningService is
// where this gets applied to an actual ProvisioningJob fetched from the
// repository, and where an apperror.KindConflict is raised if it
// disallows the requested transition.
func (s ProvisioningStatus) CanTransitionTo(target ProvisioningStatus) bool {
	for _, allowed := range provisioningTransitions[s] {
		if allowed == target {
			return true
		}
	}
	return false
}
