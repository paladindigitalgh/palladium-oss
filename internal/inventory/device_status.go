package inventory

import "strings"

// DeviceStatus is a Device's lifecycle state. It is a distinct type, not a
// raw string threaded through the codebase, so an unrecognized value is
// caught by validation instead of being silently persisted — see CLAUDE.md's
// Error Handling section ("errors should be actionable, descriptive").
//
// Palladium's Device Collection View is scoped to CPE — customer-premises
// equipment out in homes and businesses, not shelf/rack inventory — so this
// tracks only what actually varies for that kind of device: whether it is
// currently serving a Customer. Unused is the default for any Device that
// exists in Palladium but is not attached to a Customer's Service — new
// (never yet assigned), or previously assigned and detached. Active means
// exactly one thing: it is currently attached to an active
// serviceequipment.ServiceEquipment record. Retired means it has been fully
// pulled out of service (see
// internal/provisioning/kontron/service.DeauthorizationService), the
// terminal state.
//
// Active and Unused are set automatically, not chosen by an operator on
// creation (see internal/serviceequipment/service.ServiceEquipmentService's
// Create/Update, which flips a Device to Active when it gains an active
// ServiceEquipment record and back to Unused when that record's RemovedAt is
// set) — a person can still correct it by hand through Edit Device, but the
// New Device form no longer offers Status as a choice at all, defaulting
// every new Device to Unused (see frontend DeviceFormDialog.vue).
type DeviceStatus string

// The defined DeviceStatus values. There is no "unknown"/zero-value status:
// an empty DeviceStatus is invalid, so Status is effectively required on
// every Device (see Device.Validate in validate.go).
const (
	DeviceStatusActive  DeviceStatus = "Active"
	DeviceStatusUnused  DeviceStatus = "Unused"
	DeviceStatusRetired DeviceStatus = "Retired"
)

// deviceStatusOrder is the authoritative, ordered set of valid statuses. It
// backs both Valid and validation error messages so the two can never
// disagree with each other.
var deviceStatusOrder = []DeviceStatus{
	DeviceStatusUnused,
	DeviceStatusActive,
	DeviceStatusRetired,
}

// Valid reports whether s is one of the defined DeviceStatus values.
func (s DeviceStatus) Valid() bool {
	for _, v := range deviceStatusOrder {
		if s == v {
			return true
		}
	}
	return false
}

// String implements fmt.Stringer.
func (s DeviceStatus) String() string {
	return string(s)
}

// deviceStatusNames renders the defined statuses as a comma-separated list,
// for use in validation error messages.
func deviceStatusNames() string {
	names := make([]string, len(deviceStatusOrder))
	for i, s := range deviceStatusOrder {
		names[i] = string(s)
	}
	return strings.Join(names, ", ")
}
