package serviceequipment

import "strings"

// EquipmentRole classifies what part a Device plays in delivering a
// Service. It is a distinct type, not a raw string, following the exact
// pattern of product.ProductCategory.
type EquipmentRole string

// The seven defined roles. There is no zero-value/default role — an
// empty EquipmentRole is invalid — so EquipmentRole is effectively
// required on every ServiceEquipment (see ServiceEquipment.Validate in
// validate.go).
const (
	EquipmentRoleONU             EquipmentRole = "ONU"
	EquipmentRoleGateway         EquipmentRole = "Gateway"
	EquipmentRoleRouter          EquipmentRole = "Router"
	EquipmentRoleONT             EquipmentRole = "ONT"
	EquipmentRoleWiFiAccessPoint EquipmentRole = "WiFiAccessPoint"
	EquipmentRoleUPS             EquipmentRole = "UPS"
	EquipmentRoleOther           EquipmentRole = "Other"
)

// equipmentRoleOrder is the authoritative, ordered set of valid roles. It
// backs both Valid and validation error messages so the two can never
// disagree.
var equipmentRoleOrder = []EquipmentRole{
	EquipmentRoleONU,
	EquipmentRoleGateway,
	EquipmentRoleRouter,
	EquipmentRoleONT,
	EquipmentRoleWiFiAccessPoint,
	EquipmentRoleUPS,
	EquipmentRoleOther,
}

// Valid reports whether r is one of the seven defined EquipmentRole
// values.
func (r EquipmentRole) Valid() bool {
	for _, v := range equipmentRoleOrder {
		if r == v {
			return true
		}
	}
	return false
}

// String implements fmt.Stringer.
func (r EquipmentRole) String() string {
	return string(r)
}

// equipmentRoleNames renders the defined roles as a comma-separated list,
// for use in validation error messages.
func equipmentRoleNames() string {
	names := make([]string, len(equipmentRoleOrder))
	for i, r := range equipmentRoleOrder {
		names[i] = string(r)
	}
	return strings.Join(names, ", ")
}
