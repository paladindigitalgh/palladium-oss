package accessinterface

import "strings"

// Technology identifies which access technology an AccessInterface
// speaks. It is a distinct type, not a raw string, following the exact
// pattern of olt.Vendor.
type Technology string

// The four defined technologies. There is no zero-value/default
// technology — an empty Technology is invalid — so Technology is
// effectively required on every AccessInterface (see
// AccessInterface.Validate in validate.go).
const (
	TechnologyGPON           Technology = "GPON"
	TechnologyXGSPON         Technology = "XGSPON"
	TechnologyActiveEthernet Technology = "ActiveEthernet"
	TechnologyOther          Technology = "Other"
)

// technologyOrder is the authoritative, ordered set of valid
// technologies. It backs both Valid and validation error messages so the
// two can never disagree.
var technologyOrder = []Technology{
	TechnologyGPON,
	TechnologyXGSPON,
	TechnologyActiveEthernet,
	TechnologyOther,
}

// Valid reports whether t is one of the four defined Technology values.
func (t Technology) Valid() bool {
	for _, defined := range technologyOrder {
		if t == defined {
			return true
		}
	}
	return false
}

// String implements fmt.Stringer.
func (t Technology) String() string {
	return string(t)
}

// technologyNames renders the defined technologies as a comma-separated
// list, for use in validation error messages.
func technologyNames() string {
	names := make([]string, len(technologyOrder))
	for i, t := range technologyOrder {
		names[i] = string(t)
	}
	return strings.Join(names, ", ")
}
