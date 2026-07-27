package inventory

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether s has every required field set. It returns a
// platform validation error (see internal/platform/validate and
// internal/platform/apperror) rather than a bespoke type, so every domain
// package's validation failures look the same to callers.
func (s Site) Validate() error {
	errs := validate.New()
	if !validate.Required(s.Name) {
		errs.Add("name", "is required")
	}
	return errs.Err()
}

// Validate reports whether b has every required field set.
func (b Building) Validate() error {
	errs := validate.New()
	if !validate.Required(b.Name) {
		errs.Add("name", "is required")
	}
	if b.SiteID == uuid.Nil {
		errs.Add("site_id", "is required")
	}
	return errs.Err()
}

// Validate reports whether r has every required field set.
func (r Room) Validate() error {
	errs := validate.New()
	if !validate.Required(r.Name) {
		errs.Add("name", "is required")
	}
	if r.BuildingID == uuid.Nil {
		errs.Add("building_id", "is required")
	}
	return errs.Err()
}

// Validate reports whether r has every required field set. RoomID is
// intentionally not checked: it is nullable by design (see model.go), so a
// nil value is valid, not a validation failure.
func (r Rack) Validate() error {
	errs := validate.New()
	if !validate.Required(r.Name) {
		errs.Add("name", "is required")
	}
	return errs.Err()
}

// Validate reports whether d has every required field set.
//
// RackID is intentionally not checked, for the same reason as Rack.RoomID.
// AssetTag is optional and is never checked for presence. Status has no
// zero value among the defined statuses (see device_status.go), so it is
// effectively required: an empty Status fails the "must be one of" check
// below along with any unrecognized value.
func (d Device) Validate() error {
	errs := validate.New()
	if !validate.Required(d.Name) {
		errs.Add("name", "is required")
	}
	if !validate.Required(d.Manufacturer) {
		errs.Add("manufacturer", "is required")
	}
	if !validate.Required(d.Model) {
		errs.Add("model", "is required")
	}
	if !validate.Required(d.SerialNumber) {
		errs.Add("serial_number", "is required")
	}
	if !d.Status.Valid() {
		errs.Add("status", fmt.Sprintf("must be one of: %s", deviceStatusNames()))
	}
	return errs.Err()
}
