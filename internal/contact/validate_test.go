package contact_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/contact"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// assertInvalid mirrors internal/location/validate_test.go's helper of
// the same name: every domain package's Validate() must return an
// *apperror.Error of KindInvalid.
func assertInvalid(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("Validate() = nil, want error")
	}

	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("Validate() error is not an *apperror.Error: %v", err)
	}
	if appErr.Kind != apperror.KindInvalid {
		t.Errorf("Kind = %q, want %q", appErr.Kind, apperror.KindInvalid)
	}
}

func validContact() contact.Contact {
	return contact.Contact{
		CustomerID: uuid.New(),
		Name:       "Jane Doe",
		Role:       contact.ContactRolePrimary,
		Status:     contact.ContactStatusActive,
	}
}

func TestContactValidate(t *testing.T) {
	if err := validContact().Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, contact.Contact{}.Validate())
}

func TestContactValidateRequiresCustomerID(t *testing.T) {
	c := validContact()
	c.CustomerID = uuid.Nil

	assertInvalid(t, c.Validate())
}

func TestContactValidateRequiresName(t *testing.T) {
	c := validContact()
	c.Name = ""

	assertInvalid(t, c.Validate())
}

func TestContactValidateRequiresKnownRole(t *testing.T) {
	unrecognized := validContact()
	unrecognized.Role = contact.ContactRole("Executive")
	assertInvalid(t, unrecognized.Validate())

	unset := validContact()
	unset.Role = ""
	assertInvalid(t, unset.Validate())

	for _, r := range []contact.ContactRole{
		contact.ContactRolePrimary,
		contact.ContactRoleBilling,
		contact.ContactRoleTechnical,
		contact.ContactRoleEmergency,
		contact.ContactRoleOther,
	} {
		c := validContact()
		c.Role = r
		if err := c.Validate(); err != nil {
			t.Errorf("Validate() (role %q) = %v, want nil", r, err)
		}
	}
}

func TestContactValidateRequiresKnownStatus(t *testing.T) {
	unrecognized := validContact()
	unrecognized.Status = contact.ContactStatus("Archived")
	assertInvalid(t, unrecognized.Validate())

	unset := validContact()
	unset.Status = ""
	assertInvalid(t, unset.Validate())

	for _, s := range []contact.ContactStatus{
		contact.ContactStatusActive,
		contact.ContactStatusInactive,
	} {
		c := validContact()
		c.Status = s
		if err := c.Validate(); err != nil {
			t.Errorf("Validate() (status %q) = %v, want nil", s, err)
		}
	}
}

func TestContactValidateEmailPhoneDescriptionAreOptional(t *testing.T) {
	c := validContact() // no email, phone, or description set
	if err := c.Validate(); err != nil {
		t.Errorf("Validate() (no email/phone/description) = %v, want nil", err)
	}

	c.Email = "jane@example.com"
	c.Phone = "555-0100"
	c.Description = "Prefers email over phone"
	if err := c.Validate(); err != nil {
		t.Errorf("Validate() (with email/phone/description) = %v, want nil", err)
	}
}
