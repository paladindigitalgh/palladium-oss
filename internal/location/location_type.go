package location

import "strings"

// LocationType classifies what a Location is used for, independent of
// LocationStatus (which tracks whether it is currently active). It is a
// distinct type, not a raw string, following the exact pattern of
// inventory.DeviceStatus, auth.Role, and customer.CustomerType: an
// unrecognized value is caught by validation instead of silently
// persisted.
type LocationType string

// The seven defined location types. There is no zero-value/default type
// — an empty LocationType is invalid — so Type is effectively required on
// every Location (see Location.Validate in validate.go), for the same
// reason inventory.DeviceStatus has no default status.
const (
	LocationTypeService    LocationType = "Service"
	LocationTypeBilling    LocationType = "Billing"
	LocationTypeOffice     LocationType = "Office"
	LocationTypeWarehouse  LocationType = "Warehouse"
	LocationTypePOP        LocationType = "POP"
	LocationTypeDataCenter LocationType = "DataCenter"
	LocationTypeOther      LocationType = "Other"
)

// locationTypeOrder is the authoritative, ordered set of valid types. It
// backs both Valid and validation error messages so the two can never
// disagree.
var locationTypeOrder = []LocationType{
	LocationTypeService,
	LocationTypeBilling,
	LocationTypeOffice,
	LocationTypeWarehouse,
	LocationTypePOP,
	LocationTypeDataCenter,
	LocationTypeOther,
}

// Valid reports whether t is one of the seven defined LocationType
// values.
func (t LocationType) Valid() bool {
	for _, v := range locationTypeOrder {
		if t == v {
			return true
		}
	}
	return false
}

// String implements fmt.Stringer.
func (t LocationType) String() string {
	return string(t)
}

// locationTypeNames renders the defined types as a comma-separated list,
// for use in validation error messages.
func locationTypeNames() string {
	names := make([]string, len(locationTypeOrder))
	for i, t := range locationTypeOrder {
		names[i] = string(t)
	}
	return strings.Join(names, ", ")
}
