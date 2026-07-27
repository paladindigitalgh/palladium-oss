package validate_test

import (
	"errors"
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

func TestErrNilWhenNothingAdded(t *testing.T) {
	errs := validate.New()

	if got := errs.Err(); got != nil {
		t.Errorf("Err() = %v, want nil", got)
	}
}

func TestErrReturnsInvalidAppError(t *testing.T) {
	errs := validate.New()
	errs.Add("name", "is required")
	errs.Add("email", "is not a valid address")

	err := errs.Err()
	if err == nil {
		t.Fatal("Err() = nil, want an error")
	}

	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatal("Err() did not return an *apperror.Error")
	}
	if appErr.Kind != apperror.KindInvalid {
		t.Errorf("Kind = %q, want %q", appErr.Kind, apperror.KindInvalid)
	}
}

func TestFieldsReturnsAccumulatedErrors(t *testing.T) {
	errs := validate.New()
	errs.Add("name", "is required")

	fields := errs.Fields()
	if len(fields) != 1 {
		t.Fatalf("len(Fields()) = %d, want 1", len(fields))
	}
	if fields[0].Field != "name" || fields[0].Message != "is required" {
		t.Errorf("Fields()[0] = %+v, want {name is required}", fields[0])
	}
}

func TestRequired(t *testing.T) {
	cases := map[string]bool{
		"":      false,
		"   ":   false,
		"value": true,
	}
	for input, want := range cases {
		if got := validate.Required(input); got != want {
			t.Errorf("Required(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestMaxLength(t *testing.T) {
	if validate.MaxLength("hello", 4) {
		t.Error("MaxLength(\"hello\", 4) = true, want false")
	}
	if !validate.MaxLength("hello", 5) {
		t.Error("MaxLength(\"hello\", 5) = false, want true")
	}
}

func TestMinLength(t *testing.T) {
	if validate.MinLength("hi", 3) {
		t.Error("MinLength(\"hi\", 3) = true, want false")
	}
	if !validate.MinLength("hi", 2) {
		t.Error("MinLength(\"hi\", 2) = false, want true")
	}
}

func TestInRange(t *testing.T) {
	if !validate.InRange(5, 1, 10) {
		t.Error("InRange(5, 1, 10) = false, want true")
	}
	if validate.InRange(11, 1, 10) {
		t.Error("InRange(11, 1, 10) = true, want false")
	}
}
