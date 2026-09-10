package service

import "strings"

// ServiceStatus is a Service's lifecycle state. It is a distinct type,
// not a raw string, following the exact pattern of
// product.ProductStatus and location.LocationStatus.
//
// Unlike Catalog's or Location's flat two-value lifecycle, this is a
// four-value progression that mirrors the real lifecycle a subscriber
// purchase goes through: Pending (purchased, not yet delivered) ->
// Active (in service) -> Suspended (temporarily withheld, e.g.
// non-payment) -> Disconnected (permanently ended). This package does not
// enforce that services move through these states in that order — no
// state machine, no transition table — because this milestone is
// explicitly not provisioning or automation; Validate only checks that
// Status is one of these four defined values, the same shallow check
// every other status field in this codebase gets.
type ServiceStatus string

// The four defined statuses. There is no zero-value/default status — an
// empty ServiceStatus is invalid — so ServiceStatus is effectively
// required on every Service (see Service.Validate in validate.go).
const (
	ServiceStatusPending      ServiceStatus = "Pending"
	ServiceStatusActive       ServiceStatus = "Active"
	ServiceStatusSuspended    ServiceStatus = "Suspended"
	ServiceStatusDisconnected ServiceStatus = "Disconnected"
)

// serviceStatusOrder is the authoritative, ordered set of valid statuses.
// It backs both Valid and validation error messages so the two can never
// disagree.
var serviceStatusOrder = []ServiceStatus{
	ServiceStatusPending,
	ServiceStatusActive,
	ServiceStatusSuspended,
	ServiceStatusDisconnected,
}

// Valid reports whether s is one of the four defined ServiceStatus
// values.
func (s ServiceStatus) Valid() bool {
	for _, v := range serviceStatusOrder {
		if s == v {
			return true
		}
	}
	return false
}

// String implements fmt.Stringer.
func (s ServiceStatus) String() string {
	return string(s)
}

// HasAppliedProfile reports whether s implies a vendor service-profile is
// currently applied to the Service's equipment on the OLT — Active is
// the only status ProvisionService/ResumeService ever leave a Service in
// with a profile still on the ONU. Pending never had one applied yet,
// and Suspended and Disconnected have both already had it removed:
// SuspendService and DisconnectService are, at the Kontron config level,
// the identical action (see
// internal/provisioning/kontron.Client.RemoveServiceProfile's own doc
// comment) — confirmed live against the real Kontron/Iskratel C16
// (2026-09-10), where a Suspended Service's profile really was already
// gone from the ONU. An earlier version of this method also treated
// Suspended as still-applied; that was wrong, not just imprecise, and
// would have made internal/customer/removal.RemovalService try to
// remove an already-removed profile a second time, which the real
// device rejects outright rather than treating as a harmless no-op.
// This is the single source of truth RemovalService and
// internal/service/service.ServiceService.Delete both use to decide
// whether a real OLT teardown is still owed before a Service's records
// can safely be removed.
func (s ServiceStatus) HasAppliedProfile() bool {
	return s == ServiceStatusActive
}

// serviceStatusNames renders the defined statuses as a comma-separated
// list, for use in validation error messages.
func serviceStatusNames() string {
	names := make([]string, len(serviceStatusOrder))
	for i, s := range serviceStatusOrder {
		names[i] = string(s)
	}
	return strings.Join(names, ", ")
}
