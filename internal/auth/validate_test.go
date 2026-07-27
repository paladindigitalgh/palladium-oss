package auth_test

import (
	"errors"
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
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

func TestUserValidate(t *testing.T) {
	valid := auth.User{Email: "jane@example.com", PasswordHash: "$2a$10$examplehash"}
	if err := valid.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, auth.User{}.Validate())
}

func TestUserValidateRequiresEmail(t *testing.T) {
	u := auth.User{PasswordHash: "$2a$10$examplehash"}

	assertInvalid(t, u.Validate())
}

func TestUserValidateRejectsMalformedEmail(t *testing.T) {
	cases := []string{
		"not-an-email",
		"missing-domain@",
		"@missing-local.com",
		"no-at-sign.example.com",
		"spaces in@address.com",
	}

	for _, email := range cases {
		u := auth.User{Email: email, PasswordHash: "$2a$10$examplehash"}
		if err := u.Validate(); err == nil {
			t.Errorf("Validate() = nil for email %q, want error", email)
		} else {
			assertInvalid(t, err)
		}
	}
}

func TestUserValidateRequiresPasswordHash(t *testing.T) {
	u := auth.User{Email: "jane@example.com"}

	assertInvalid(t, u.Validate())
}
