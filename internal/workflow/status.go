package workflow

import "strings"

// Status is a WorkflowInstance's lifecycle state, with a defined
// transition table — a direct port of the former
// provisioning.ProvisioningStatus.
type Status string

const (
	StatusPending   Status = "Pending"
	StatusRunning   Status = "Running"
	StatusSucceeded Status = "Succeeded"
	StatusFailed    Status = "Failed"
	StatusCancelled Status = "Cancelled"
)

var statusOrder = []Status{
	StatusPending,
	StatusRunning,
	StatusSucceeded,
	StatusFailed,
	StatusCancelled,
}

// Valid reports whether s is one of the five defined Status values.
func (s Status) Valid() bool {
	for _, v := range statusOrder {
		if s == v {
			return true
		}
	}
	return false
}

// String implements fmt.Stringer.
func (s Status) String() string { return string(s) }

func statusNames() string {
	names := make([]string, len(statusOrder))
	for i, s := range statusOrder {
		names[i] = string(s)
	}
	return strings.Join(names, ", ")
}

// transitions is the authoritative set of allowed Status transitions:
//
//	Pending   -> Running, Cancelled
//	Running   -> Succeeded, Failed, Cancelled
//	Failed    -> Pending
//	Succeeded -> (terminal)
//	Cancelled -> (terminal)
var transitions = map[Status][]Status{
	StatusPending: {StatusRunning, StatusCancelled},
	StatusRunning: {StatusSucceeded, StatusFailed, StatusCancelled},
	StatusFailed:  {StatusPending},
}

// CanTransitionTo reports whether moving from s to target is an allowed
// transition. It is a pure fact about the state machine's shape;
// internal/workflow/service.Service is where this gets applied to an
// actual, persisted WorkflowInstance.
func (s Status) CanTransitionTo(target Status) bool {
	for _, allowed := range transitions[s] {
		if allowed == target {
			return true
		}
	}
	return false
}
