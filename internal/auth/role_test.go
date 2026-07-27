package auth_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
)

func TestRoleValidAcceptsDefinedValues(t *testing.T) {
	for _, role := range []auth.Role{auth.RoleAdministrator, auth.RoleOperator, auth.RoleViewer} {
		if !role.Valid() {
			t.Errorf("%q.Valid() = false, want true", role)
		}
	}
}

func TestRoleValidRejectsUnrecognizedValues(t *testing.T) {
	cases := []auth.Role{
		"",              // zero value: there is no default role
		"administrator", // wrong case
		"OPERATOR",      // wrong case
		"SuperAdmin",    // not a defined role at all
	}

	for _, role := range cases {
		if role.Valid() {
			t.Errorf("%q.Valid() = true, want false", role)
		}
	}
}

func TestRoleStringReturnsUnderlyingValue(t *testing.T) {
	if got := auth.RoleOperator.String(); got != "Operator" {
		t.Errorf("String() = %q, want %q", got, "Operator")
	}
}
