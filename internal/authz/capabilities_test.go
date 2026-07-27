package authz_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/authz"
)

// TestCanReadInventory and TestCanWriteInventory are the literal, direct
// proof of this milestone's goal 4 access-control table for the Site
// endpoints:
//
//	GET /sites, GET /sites/{id}                     -> Administrator, Operator, Viewer
//	POST /sites, PUT /sites/{id}, DELETE /sites/{id} -> Administrator, Operator
func TestCanReadInventory(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        true,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanReadInventory(role); got != want {
			t.Errorf("CanReadInventory(%q) = %v, want %v", role, got, want)
		}
	}
}

func TestCanWriteInventory(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        false,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanWriteInventory(role); got != want {
			t.Errorf("CanWriteInventory(%q) = %v, want %v", role, got, want)
		}
	}
}

func TestCanManageUsers(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      false,
		auth.RoleViewer:        false,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanManageUsers(role); got != want {
			t.Errorf("CanManageUsers(%q) = %v, want %v", role, got, want)
		}
	}
}

// TestNoSiteCapabilityIsAdministratorExclusive is the direct check behind
// goal 4's "no Site endpoint should require Administrator exclusively":
// for every capability a Site endpoint actually uses, at least one
// non-Administrator role must also satisfy it.
func TestNoSiteCapabilityIsAdministratorExclusive(t *testing.T) {
	siteCapabilities := map[string]func(auth.Role) bool{
		"CanReadInventory":  authz.CanReadInventory,
		"CanWriteInventory": authz.CanWriteInventory,
	}

	nonAdminRoles := []auth.Role{auth.RoleOperator, auth.RoleViewer}

	for name, capability := range siteCapabilities {
		allowsANonAdmin := false
		for _, role := range nonAdminRoles {
			if capability(role) {
				allowsANonAdmin = true
				break
			}
		}
		if !allowsANonAdmin {
			t.Errorf("%s allows no role but Administrator", name)
		}
	}
}
