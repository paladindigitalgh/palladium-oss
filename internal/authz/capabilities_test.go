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

// TestCanReadCustomers and TestCanWriteCustomers are the same direct proof
// as TestCanReadInventory/TestCanWriteInventory, applied to the Customer
// domain's access-control table (goal 6: "apply the same authorization
// model as Sites").
func TestCanReadCustomers(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        true,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanReadCustomers(role); got != want {
			t.Errorf("CanReadCustomers(%q) = %v, want %v", role, got, want)
		}
	}
}

func TestCanWriteCustomers(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        false,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanWriteCustomers(role); got != want {
			t.Errorf("CanWriteCustomers(%q) = %v, want %v", role, got, want)
		}
	}
}

// TestCanReadLocations and TestCanWriteLocations are the same direct proof
// as TestCanReadCustomers/TestCanWriteCustomers, applied to the Location
// domain's access-control table ("match Customer permissions").
func TestCanReadLocations(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        true,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanReadLocations(role); got != want {
			t.Errorf("CanReadLocations(%q) = %v, want %v", role, got, want)
		}
	}
}

func TestCanWriteLocations(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        false,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanWriteLocations(role); got != want {
			t.Errorf("CanWriteLocations(%q) = %v, want %v", role, got, want)
		}
	}
}

// TestNoAdministratorExclusiveCapabilityForSitesOrCustomers is the direct
// check behind "no Site endpoint should require Administrator
// exclusively" (goal 4) and its Customer and Location equivalents: for
// every capability a Site, Customer, or Location endpoint actually uses,
// at least one non-Administrator role must also satisfy it.
func TestNoAdministratorExclusiveCapabilityForSitesOrCustomers(t *testing.T) {
	capabilities := map[string]func(auth.Role) bool{
		"CanReadInventory":  authz.CanReadInventory,
		"CanWriteInventory": authz.CanWriteInventory,
		"CanReadCustomers":  authz.CanReadCustomers,
		"CanWriteCustomers": authz.CanWriteCustomers,
		"CanReadLocations":  authz.CanReadLocations,
		"CanWriteLocations": authz.CanWriteLocations,
	}

	nonAdminRoles := []auth.Role{auth.RoleOperator, auth.RoleViewer}

	for name, capability := range capabilities {
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
