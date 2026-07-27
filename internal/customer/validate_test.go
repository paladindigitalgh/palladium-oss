package customer_test

import (
	"errors"
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/customer"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// assertInvalid mirrors internal/inventory/validate_test.go's helper of
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

func validCustomer() customer.Customer {
	return customer.Customer{
		Name:         "Jane Doe",
		CustomerType: customer.CustomerTypeResidential,
		Status:       customer.CustomerStatusActive,
	}
}

func TestCustomerValidate(t *testing.T) {
	if err := validCustomer().Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, customer.Customer{}.Validate())
}

func TestCustomerValidateRequiresName(t *testing.T) {
	c := validCustomer()
	c.Name = ""

	assertInvalid(t, c.Validate())
}

func TestCustomerValidateRequiresKnownCustomerType(t *testing.T) {
	unrecognized := validCustomer()
	unrecognized.CustomerType = customer.CustomerType("Nonprofit")
	assertInvalid(t, unrecognized.Validate())

	unset := validCustomer()
	unset.CustomerType = ""
	assertInvalid(t, unset.Validate())

	for _, ct := range []customer.CustomerType{
		customer.CustomerTypeResidential,
		customer.CustomerTypeBusiness,
		customer.CustomerTypeGovernment,
		customer.CustomerTypeInternal,
	} {
		c := validCustomer()
		c.CustomerType = ct
		if err := c.Validate(); err != nil {
			t.Errorf("Validate() (customer_type %q) = %v, want nil", ct, err)
		}
	}
}

func TestCustomerValidateRequiresKnownStatus(t *testing.T) {
	unrecognized := validCustomer()
	unrecognized.Status = customer.CustomerStatus("Deleted")
	assertInvalid(t, unrecognized.Validate())

	unset := validCustomer()
	unset.Status = ""
	assertInvalid(t, unset.Validate())

	for _, s := range []customer.CustomerStatus{
		customer.CustomerStatusActive,
		customer.CustomerStatusInactive,
		customer.CustomerStatusArchived,
	} {
		c := validCustomer()
		c.Status = s
		if err := c.Validate(); err != nil {
			t.Errorf("Validate() (status %q) = %v, want nil", s, err)
		}
	}
}

func TestCustomerValidateDescriptionIsOptional(t *testing.T) {
	withoutDescription := validCustomer()
	withoutDescription.Description = ""
	if err := withoutDescription.Validate(); err != nil {
		t.Errorf("Validate() (no description) = %v, want nil", err)
	}

	withDescription := validCustomer()
	withDescription.Description = "Long-time residential customer"
	if err := withDescription.Validate(); err != nil {
		t.Errorf("Validate() (with description) = %v, want nil", err)
	}
}
