package location_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/location"
)

func TestLocationTypeValidAcceptsDefinedValues(t *testing.T) {
	defined := []location.LocationType{
		location.LocationTypeService,
		location.LocationTypeBilling,
		location.LocationTypeOffice,
		location.LocationTypeWarehouse,
		location.LocationTypePOP,
		location.LocationTypeDataCenter,
		location.LocationTypeOther,
	}

	for _, lt := range defined {
		if !lt.Valid() {
			t.Errorf("%q.Valid() = false, want true", lt)
		}
	}
}

func TestLocationTypeValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []location.LocationType{
		"",        // zero value: there is no default type
		"service", // wrong case
		"OFFICE",  // wrong case
		"HeadEnd", // not a defined type at all
	}

	for _, lt := range cases {
		if lt.Valid() {
			t.Errorf("%q.Valid() = true, want false", lt)
		}
	}
}

func TestLocationTypeStringReturnsUnderlyingValue(t *testing.T) {
	if got := location.LocationTypeDataCenter.String(); got != "DataCenter" {
		t.Errorf("String() = %q, want %q", got, "DataCenter")
	}
}
