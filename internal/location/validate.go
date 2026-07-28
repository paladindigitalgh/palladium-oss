package location

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether l has every required field set: a present
// CustomerID, a present Name, and a Type and Status that are each one of
// their defined values (see location_type.go and status.go).
//
// The address fields (Address1, Address2, City, State, PostalCode,
// Country) and Latitude/Longitude are never checked — goal 1 says so
// explicitly ("address fields are optional for now", "latitude and
// longitude are optional") — consistent with how Description is handled
// everywhere else in this codebase. Deliberately absent: any check that
// Latitude/Longitude, if given, fall within a valid range (-90..90,
// -180..180). That is a real validation a GIS-aware version of this
// package might add; this milestone's explicit "no GIS" scope means this
// package does not reason about what makes a coordinate valid, only
// whether one was supplied.
func (l Location) Validate() error {
	errs := validate.New()

	if l.CustomerID == uuid.Nil {
		errs.Add("customer_id", "is required")
	}
	if !validate.Required(l.Name) {
		errs.Add("name", "is required")
	}
	if !l.Type.Valid() {
		errs.Add("type", fmt.Sprintf("must be one of: %s", locationTypeNames()))
	}
	if !l.Status.Valid() {
		errs.Add("status", fmt.Sprintf("must be one of: %s", locationStatusNames()))
	}

	return errs.Err()
}
