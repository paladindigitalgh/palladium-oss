package accessinterface_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/accessinterface"
)

func TestTechnologyValidAcceptsDefinedValues(t *testing.T) {
	defined := []accessinterface.Technology{
		accessinterface.TechnologyGPON,
		accessinterface.TechnologyXGSPON,
		accessinterface.TechnologyActiveEthernet,
		accessinterface.TechnologyOther,
	}

	for _, tech := range defined {
		if !tech.Valid() {
			t.Errorf("%q.Valid() = false, want true", tech)
		}
	}
}

func TestTechnologyValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []accessinterface.Technology{
		"",        // zero value: there is no default technology
		"gpon",    // wrong case
		"XGS-PON", // not a defined technology
		"EPON",    // not a defined technology
	}

	for _, tech := range cases {
		if tech.Valid() {
			t.Errorf("%q.Valid() = true, want false", tech)
		}
	}
}

func TestTechnologyStringReturnsUnderlyingValue(t *testing.T) {
	if got := accessinterface.TechnologyGPON.String(); got != "GPON" {
		t.Errorf("String() = %q, want %q", got, "GPON")
	}
}
