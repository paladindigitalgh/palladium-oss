package serviceequipment_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

func TestEquipmentRoleValidAcceptsDefinedValues(t *testing.T) {
	defined := []serviceequipment.EquipmentRole{
		serviceequipment.EquipmentRoleONU,
		serviceequipment.EquipmentRoleGateway,
		serviceequipment.EquipmentRoleRouter,
		serviceequipment.EquipmentRoleONT,
		serviceequipment.EquipmentRoleWiFiAccessPoint,
		serviceequipment.EquipmentRoleUPS,
		serviceequipment.EquipmentRoleOther,
	}

	for _, r := range defined {
		if !r.Valid() {
			t.Errorf("%q.Valid() = false, want true", r)
		}
	}
}

func TestEquipmentRoleValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []serviceequipment.EquipmentRole{
		"",       // zero value: there is no default role
		"onu",    // wrong case
		"ROUTER", // wrong case
		"Switch", // not a defined role at all
	}

	for _, r := range cases {
		if r.Valid() {
			t.Errorf("%q.Valid() = true, want false", r)
		}
	}
}

func TestEquipmentRoleStringReturnsUnderlyingValue(t *testing.T) {
	if got := serviceequipment.EquipmentRoleWiFiAccessPoint.String(); got != "WiFiAccessPoint" {
		t.Errorf("String() = %q, want %q", got, "WiFiAccessPoint")
	}
}
