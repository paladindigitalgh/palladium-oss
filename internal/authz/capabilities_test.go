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

// TestCanReadCatalog and TestCanWriteCatalog are the same direct proof as
// TestCanReadLocations/TestCanWriteLocations, applied to the Product
// Catalog domain's access-control table ("apply the standard RBAC
// matrix").
func TestCanReadCatalog(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        true,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanReadCatalog(role); got != want {
			t.Errorf("CanReadCatalog(%q) = %v, want %v", role, got, want)
		}
	}
}

func TestCanWriteCatalog(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        false,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanWriteCatalog(role); got != want {
			t.Errorf("CanWriteCatalog(%q) = %v, want %v", role, got, want)
		}
	}
}

// TestCanReadServices and TestCanWriteServices are the same direct proof
// as TestCanReadCatalog/TestCanWriteCatalog, applied to the Service
// domain's access-control table ("apply the standard RBAC matrix").
func TestCanReadServices(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        true,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanReadServices(role); got != want {
			t.Errorf("CanReadServices(%q) = %v, want %v", role, got, want)
		}
	}
}

func TestCanWriteServices(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        false,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanWriteServices(role); got != want {
			t.Errorf("CanWriteServices(%q) = %v, want %v", role, got, want)
		}
	}
}

// TestCanReadServiceEquipment and TestCanWriteServiceEquipment are the
// same direct proof as TestCanReadServices/TestCanWriteServices, applied
// to the Service Equipment domain's access-control table ("apply the
// standard RBAC matrix").
func TestCanReadServiceEquipment(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        true,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanReadServiceEquipment(role); got != want {
			t.Errorf("CanReadServiceEquipment(%q) = %v, want %v", role, got, want)
		}
	}
}

func TestCanWriteServiceEquipment(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        false,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanWriteServiceEquipment(role); got != want {
			t.Errorf("CanWriteServiceEquipment(%q) = %v, want %v", role, got, want)
		}
	}
}

// TestNoAdministratorExclusiveCapabilityForSitesOrCustomers is the direct
// check behind "no Site endpoint should require Administrator
// exclusively" (goal 4) and its Customer, Location, Catalog, Service, and
// Service Equipment equivalents: for every capability a Site, Customer,
// Location, Catalog/Product, Service, or Service Equipment endpoint
// actually uses, at least one non-Administrator role must also satisfy
// it.
func TestNoAdministratorExclusiveCapabilityForSitesOrCustomers(t *testing.T) {
	capabilities := map[string]func(auth.Role) bool{
		"CanReadInventory":         authz.CanReadInventory,
		"CanWriteInventory":        authz.CanWriteInventory,
		"CanReadCustomers":         authz.CanReadCustomers,
		"CanWriteCustomers":        authz.CanWriteCustomers,
		"CanReadLocations":         authz.CanReadLocations,
		"CanWriteLocations":        authz.CanWriteLocations,
		"CanReadCatalog":           authz.CanReadCatalog,
		"CanWriteCatalog":          authz.CanWriteCatalog,
		"CanReadServices":          authz.CanReadServices,
		"CanWriteServices":         authz.CanWriteServices,
		"CanReadServiceEquipment":  authz.CanReadServiceEquipment,
		"CanWriteServiceEquipment": authz.CanWriteServiceEquipment,
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
