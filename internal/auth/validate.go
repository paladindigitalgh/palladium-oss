package auth

import (
	"fmt"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether u has every required field set: a present,
// correctly shaped Email; a non-empty PasswordHash (present "when
// persisted", per this milestone's goals — Validate is meant to be called
// on a User about to be stored, after a plaintext password has already
// gone through HashPassword, never before); and a Role that is one of the
// three defined values. It cannot verify PasswordHash was actually
// produced by HashPassword — that is a property of how the value was
// constructed, not something the shape of the string reveals.
func (u User) Validate() error {
	errs := validate.New()

	if !validate.Required(u.Email) {
		errs.Add("email", "is required")
	} else if !validate.Email(u.Email) {
		errs.Add("email", "is not a valid email address")
	}

	if !validate.Required(u.PasswordHash) {
		errs.Add("password_hash", "is required")
	}

	if !u.Role.Valid() {
		errs.Add("role", fmt.Sprintf("must be one of: %s", roleNames()))
	}

	return errs.Err()
}
