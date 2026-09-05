package provisioning

import (
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether p has every required field set: a present
// ProductID, and non-empty Vendor and ProfileName. Description is
// optional and is never checked for presence, consistent with
// product.Product.Validate.
func (p ProvisioningProfile) Validate() error {
	errs := validate.New()

	if p.ProductID == uuid.Nil {
		errs.Add("product_id", "is required")
	}
	if !validate.Required(p.Vendor) {
		errs.Add("vendor", "is required")
	}
	if !validate.Required(p.ProfileName) {
		errs.Add("profile_name", "is required")
	}

	return errs.Err()
}
