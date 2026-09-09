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
// Country) and Latitude/Longitude remain optional — goal 1 says so
// explicitly ("address fields are optional for now", "latitude and
// longitude are optional") — consistent with how Description is handled
// everywhere else in this codebase. Deliberately absent: any check that
// Latitude/Longitude, if given, fall within a valid range (-90..90,
// -180..180). That is a real validation a GIS-aware version of this
// package might add; this milestone's explicit "no GIS" scope means this
// package does not reason about what makes a coordinate valid, only
// whether one was supplied.
//
// State and PostalCode are the two exceptions: still optional, but their
// shape is checked once a value is present, via
// validate.USState/validate.USPostalCode — a two-letter USPS
// abbreviation and a 5-digit ZIP, applied unconditionally regardless of
// Country. Country is free text defaulting to "US" everywhere it is set
// (see frontend/src/components/dialogs/LocationFormDialog.vue) and no
// other value is used anywhere in this codebase today, so this
// deliberately does not branch on it — the same "don't build for a
// hypothetical" reasoning this codebase applies throughout. A real
// non-US address is a future problem for whenever Country actually
// varies.
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
	if l.State != "" && !validate.USState(l.State) {
		errs.Add("state", "must be a two-letter state abbreviation (e.g. CA)")
	}
	if l.PostalCode != "" && !validate.USPostalCode(l.PostalCode) {
		errs.Add("postal_code", "must be 5 digits (e.g. 94103)")
	}

	return errs.Err()
}
