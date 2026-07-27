package inventory

import "strings"

// DeviceStatus is a Device's lifecycle state. It is a distinct type, not a
// raw string threaded through the codebase, so an unrecognized value is
// caught by validation instead of being silently persisted — see CLAUDE.md's
// Error Handling section ("errors should be actionable, descriptive").
//
// The values echo the Inventory Philosophy lifecycle in
// docs/ARCHITECTURE.md (Ordered -> Received -> Stored -> Installed ->
// Provisioned -> Assigned -> Retired -> Disposed), collapsed to what a
// generic Device record can meaningfully distinguish at this stage:
// InStock stands in for Stored/Provisioned/Assigned, since separating
// those requires the networking and service concepts this milestone
// intentionally excludes.
type DeviceStatus string

// The defined DeviceStatus values. There is no "unknown"/zero-value status:
// an empty DeviceStatus is invalid, so Status is effectively required on
// every Device (see Device.Validate in validate.go).
const (
	DeviceStatusOrdered     DeviceStatus = "Ordered"
	DeviceStatusReceived    DeviceStatus = "Received"
	DeviceStatusInStock     DeviceStatus = "InStock"
	DeviceStatusInstalled   DeviceStatus = "Installed"
	DeviceStatusMaintenance DeviceStatus = "Maintenance"
	DeviceStatusRetired     DeviceStatus = "Retired"
	DeviceStatusDisposed    DeviceStatus = "Disposed"
)

// deviceStatusOrder is the authoritative, ordered set of valid statuses. It
// backs both Valid and validation error messages so the two can never
// disagree with each other.
var deviceStatusOrder = []DeviceStatus{
	DeviceStatusOrdered,
	DeviceStatusReceived,
	DeviceStatusInStock,
	DeviceStatusInstalled,
	DeviceStatusMaintenance,
	DeviceStatusRetired,
	DeviceStatusDisposed,
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
