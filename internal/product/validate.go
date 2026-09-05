package product

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether p has every required field set: a present
// CatalogID, a present ProviderID, a present Name, and a Category and
// Status that are each one of their defined values (see category.go and
// status.go). Description is optional and is never checked for
// presence, consistent with location.Location.Validate.
func (p Product) Validate() error {
	errs := validate.New()

	if p.CatalogID == uuid.Nil {
		errs.Add("catalog_id", "is required")
	}
	if p.ProviderID == uuid.Nil {
		errs.Add("provider_id", "is required")
	}
	if !validate.Required(p.Name) {
		errs.Add("name", "is required")
	}
	if !p.Category.Valid() {
		errs.Add("category", fmt.Sprintf("must be one of: %s", productCategoryNames()))
	}
	if !p.Status.Valid() {
		errs.Add("status", fmt.Sprintf("must be one of: %s", productStatusNames()))
	}

	return errs.Err()
}
