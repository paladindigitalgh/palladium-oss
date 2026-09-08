package oltmodel

import "strings"

// Vendor identifies which manufacturer built an OLTModel's chassis type.
// It is a distinct type, not a raw string, following the exact pattern
// of product.ProductCategory.
//
// This type moved here from internal/olt/vendor.go: vendor now describes
// the catalog entry (a "Kontron C16" chassis type), not the individual
// physical OLT record — see this package's own doc comment for why. Its
// content and reasoning are otherwise unchanged: naming a vendor as a
// label on a record is not vendor-specific *logic* — this type has no
// behavior that differs by vendor, no branch anywhere in this package or
// its repository or service layer asks "if Vendor == Kontron do X." It
// is exactly as inert as inventory.Device.Manufacturer, a plain
// descriptive field, just closed to a known set of values instead of
// free text. The moment any code makes a decision based on Vendor's
// value, that decision belongs in a plugin (see internal/plugin, and
// concretely internal/provisioning/kontron), not here.
type Vendor string

// The five defined vendors. There is no zero-value/default vendor — an
// empty Vendor is invalid — so Vendor is effectively required on every
// OLTModel (see OLTModel.Validate in validate.go).
const (
	VendorKontron Vendor = "Kontron"
	VendorNokia   Vendor = "Nokia"
	VendorCalix   Vendor = "Calix"
	VendorAdtran  Vendor = "Adtran"
	VendorOther   Vendor = "Other"
)

// vendorOrder is the authoritative, ordered set of valid vendors. It
// backs both Valid and validation error messages so the two can never
// disagree.
var vendorOrder = []Vendor{
	VendorKontron,
	VendorNokia,
	VendorCalix,
	VendorAdtran,
	VendorOther,
}

// Valid reports whether v is one of the five defined Vendor values.
func (v Vendor) Valid() bool {
	for _, defined := range vendorOrder {
		if v == defined {
			return true
		}
	}
	return false
}

// String implements fmt.Stringer.
func (v Vendor) String() string {
	return string(v)
}

// vendorNames renders the defined vendors as a comma-separated list, for
// use in validation error messages.
func vendorNames() string {
	names := make([]string, len(vendorOrder))
	for i, v := range vendorOrder {
		names[i] = string(v)
	}
	return strings.Join(names, ", ")
}
