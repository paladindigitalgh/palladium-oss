package accessinterface

import "strings"

// Status is an AccessInterface's operational state. It is a distinct
// type, not a raw string, following the exact pattern of
// accessnetwork.AccessNetworkStatus.
//
// Like AccessNetworkStatus, this is a flat, two-value lifecycle: an
// interface answers exactly one question here, "is it currently allowed
// to be used," not a richer provisioning-style lifecycle (up/down,
// alarm, ...) — those are real-time or provisioning concerns this
// milestone explicitly excludes (see this package's doc comment).
type Status string

// The two defined statuses. There is no zero-value/default status — an
// empty Status is invalid — so Status is effectively required on every
// AccessInterface (see AccessInterface.Validate in validate.go).
const (
	StatusActive   Status = "Active"
	StatusDisabled Status = "Disabled"
)

// statusOrder is the authoritative, ordered set of valid statuses. It
// backs both Valid and validation error messages so the two can never
// disagree.
var statusOrder = []Status{
	StatusActive,
	StatusDisabled,
}

// Valid reports whether s is one of the two defined Status values.
func (s Status) Valid() bool {
	for _, v := range statusOrder {
		if s == v {
			return true
		}
	}
	return false
}

// String implements fmt.Stringer.
func (s Status) String() string {
	return string(s)
}

// statusNames renders the defined statuses as a comma-separated list,
// for use in validation error messages.
func statusNames() string {
	names := make([]string, len(statusOrder))
	for i, s := range statusOrder {
		names[i] = string(s)
	}
	return strings.Join(names, ", ")
}
