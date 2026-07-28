package location_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/location"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// assertInvalid mirrors internal/customer/validate_test.go's helper of
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

func validLocation() location.Location {
	return location.Location{
		CustomerID: uuid.New(),
		Name:       "Main Service Address",
		Type:       location.LocationTypeService,
		Status:     location.LocationStatusActive,
	}
}

func TestLocationValidate(t *testing.T) {
	if err := validLocation().Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, location.Location{}.Validate())
}

func TestLocationValidateRequiresCustomerID(t *testing.T) {
	l := validLocation()
	l.CustomerID = uuid.Nil

	assertInvalid(t, l.Validate())
}

func TestLocationValidateRequiresName(t *testing.T) {
	l := validLocation()
	l.Name = ""

	assertInvalid(t, l.Validate())
}

func TestLocationValidateRequiresKnownType(t *testing.T) {
	unrecognized := validLocation()
	unrecognized.Type = location.LocationType("HeadEnd")
	assertInvalid(t, unrecognized.Validate())

	unset := validLocation()
	unset.Type = ""
	assertInvalid(t, unset.Validate())

	for _, lt := range []location.LocationType{
		location.LocationTypeService,
		location.LocationTypeBilling,
		location.LocationTypeOffice,
		location.LocationTypeWarehouse,
		location.LocationTypePOP,
		location.LocationTypeDataCenter,
		location.LocationTypeOther,
	} {
		l := validLocation()
		l.Type = lt
		if err := l.Validate(); err != nil {
			t.Errorf("Validate() (type %q) = %v, want nil", lt, err)
		}
	}
}

func TestLocationValidateRequiresKnownStatus(t *testing.T) {
	unrecognized := validLocation()
	unrecognized.Status = location.LocationStatus("Archived")
	assertInvalid(t, unrecognized.Validate())

	unset := validLocation()
	unset.Status = ""
	assertInvalid(t, unset.Validate())

	for _, s := range []location.LocationStatus{
		location.LocationStatusActive,
		location.LocationStatusInactive,
	} {
		l := validLocation()
		l.Status = s
		if err := l.Validate(); err != nil {
			t.Errorf("Validate() (status %q) = %v, want nil", s, err)
		}
	}
}

func TestLocationValidateAddressFieldsAreOptional(t *testing.T) {
	l := validLocation() // no address fields set
	if err := l.Validate(); err != nil {
		t.Errorf("Validate() (no address) = %v, want nil", err)
	}

	l.Address1 = "123 Main St"
	l.Address2 = "Suite 100"
	l.City = "Springfield"
	l.State = "IL"
	l.PostalCode = "62701"
	l.Country = "US"
	if err := l.Validate(); err != nil {
		t.Errorf("Validate() (with address) = %v, want nil", err)
	}
}

func TestLocationValidateLatitudeLongitudeAreOptional(t *testing.T) {
	withoutCoordinates := validLocation()
	if err := withoutCoordinates.Validate(); err != nil {
		t.Errorf("Validate() (no coordinates) = %v, want nil", err)
	}

	lat, lng := 39.7817, -89.6501
	withCoordinates := validLocation()
	withCoordinates.Latitude = &lat
	withCoordinates.Longitude = &lng
	if err := withCoordinates.Validate(); err != nil {
		t.Errorf("Validate() (with coordinates) = %v, want nil", err)
	}
}
