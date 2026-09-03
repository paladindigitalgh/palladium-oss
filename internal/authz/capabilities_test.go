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

// TestCanReadContacts and TestCanWriteContacts are the same direct proof
// as TestCanReadLocations/TestCanWriteLocations, applied to the Contact
// domain's access-control table ("match Customer/Location permissions").
func TestCanReadContacts(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        true,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanReadContacts(role); got != want {
			t.Errorf("CanReadContacts(%q) = %v, want %v", role, got, want)
		}
	}
}

func TestCanWriteContacts(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        false,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanWriteContacts(role); got != want {
			t.Errorf("CanWriteContacts(%q) = %v, want %v", role, got, want)
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

// TestCanReadWorkflow and TestCanWriteWorkflow are the same
// direct proof as TestCanReadServiceEquipment/TestCanWriteServiceEquipment,
// applied to the Workflow domain's access-control table ("apply the
// standard RBAC matrix").
func TestCanReadWorkflow(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        true,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanReadWorkflow(role); got != want {
			t.Errorf("CanReadWorkflow(%q) = %v, want %v", role, got, want)
		}
	}
}

func TestCanWriteWorkflow(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        false,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanWriteWorkflow(role); got != want {
			t.Errorf("CanWriteWorkflow(%q) = %v, want %v", role, got, want)
		}
	}
}

// TestCanReadAccessNetwork and TestCanWriteAccessNetwork are the same
// direct proof as TestCanReadWorkflow/TestCanWriteWorkflow,
// applied to the Access Network domain's access-control table ("apply
// the standard RBAC matrix").
func TestCanReadAccessNetwork(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        true,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanReadAccessNetwork(role); got != want {
			t.Errorf("CanReadAccessNetwork(%q) = %v, want %v", role, got, want)
		}
	}
}

func TestCanWriteAccessNetwork(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        false,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanWriteAccessNetwork(role); got != want {
			t.Errorf("CanWriteAccessNetwork(%q) = %v, want %v", role, got, want)
		}
	}
}

// TestCanReadAccessTopology and TestCanWriteAccessTopology are the same
// direct proof as TestCanReadAccessNetwork/TestCanWriteAccessNetwork,
// applied to the Access Topology domain's access-control table ("apply
// the standard RBAC matrix").
func TestCanReadAccessTopology(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        true,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanReadAccessTopology(role); got != want {
			t.Errorf("CanReadAccessTopology(%q) = %v, want %v", role, got, want)
		}
	}
}

func TestCanWriteAccessTopology(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        false,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanWriteAccessTopology(role); got != want {
			t.Errorf("CanWriteAccessTopology(%q) = %v, want %v", role, got, want)
		}
	}
}

// TestCanReadServiceProfiles and TestCanWriteServiceProfiles are the
// same direct proof as TestCanReadAccessTopology/TestCanWriteAccessTopology,
// applied to the Service Profile domain's access-control table ("apply
// the standard RBAC matrix").
func TestCanReadServiceProfiles(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        true,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanReadServiceProfiles(role); got != want {
			t.Errorf("CanReadServiceProfiles(%q) = %v, want %v", role, got, want)
		}
	}
}

func TestCanWriteServiceProfiles(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        false,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanWriteServiceProfiles(role); got != want {
			t.Errorf("CanWriteServiceProfiles(%q) = %v, want %v", role, got, want)
		}
	}
}

// TestCanRunDiagnostics is the same direct proof as every other
// capability's own test, applied to the Diagnostics framework's
// access-control rule ("apply the standard RBAC matrix"). Unlike the
// Read/Write pairs above, there is only one capability to test here —
// see CanRunDiagnostics's doc comment for why.
func TestCanRunDiagnostics(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        false,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanRunDiagnostics(role); got != want {
			t.Errorf("CanRunDiagnostics(%q) = %v, want %v", role, got, want)
		}
	}
}

func TestCanReadAuthentication(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        true,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanReadAuthentication(role); got != want {
			t.Errorf("CanReadAuthentication(%q) = %v, want %v", role, got, want)
		}
	}
}

func TestCanWriteAuthentication(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        false,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanWriteAuthentication(role); got != want {
			t.Errorf("CanWriteAuthentication(%q) = %v, want %v", role, got, want)
		}
	}
}

func TestCanReadConnectionProfiles(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        true,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanReadConnectionProfiles(role); got != want {
			t.Errorf("CanReadConnectionProfiles(%q) = %v, want %v", role, got, want)
		}
	}
}

func TestCanWriteConnectionProfiles(t *testing.T) {
	cases := map[auth.Role]bool{
		auth.RoleAdministrator: true,
		auth.RoleOperator:      true,
		auth.RoleViewer:        false,
		auth.Role("Nonsense"):  false,
		auth.Role(""):          false,
	}

	for role, want := range cases {
		if got := authz.CanWriteConnectionProfiles(role); got != want {
			t.Errorf("CanWriteConnectionProfiles(%q) = %v, want %v", role, got, want)
		}
	}
}

// TestNoAdministratorExclusiveCapabilityForSitesOrCustomers is the direct
// check behind "no Site endpoint should require Administrator
// exclusively" (goal 4) and its Customer, Location, Catalog, Service,
// Service Equipment, Provisioning, Access Network, Access Topology,
// Service Profile, Diagnostics, Authentication, and Connection Profile
// equivalents: for every capability a Site, Customer, Location,
// Catalog/Product, Service, Service Equipment, Provisioning, Access
// Network/OLT/PONPort, Access Interface/Access Attachment, Service
// Profile, Diagnostics, Authentication, or Connection Profile endpoint
// actually uses, at least one non-Administrator role must also satisfy
// it.
func TestNoAdministratorExclusiveCapabilityForSitesOrCustomers(t *testing.T) {
	capabilities := map[string]func(auth.Role) bool{
		"CanReadInventory":           authz.CanReadInventory,
		"CanWriteInventory":          authz.CanWriteInventory,
		"CanReadCustomers":           authz.CanReadCustomers,
		"CanWriteCustomers":          authz.CanWriteCustomers,
		"CanReadLocations":           authz.CanReadLocations,
		"CanWriteLocations":          authz.CanWriteLocations,
		"CanReadCatalog":             authz.CanReadCatalog,
		"CanWriteCatalog":            authz.CanWriteCatalog,
		"CanReadServices":            authz.CanReadServices,
		"CanWriteServices":           authz.CanWriteServices,
		"CanReadServiceEquipment":    authz.CanReadServiceEquipment,
		"CanWriteServiceEquipment":   authz.CanWriteServiceEquipment,
		"CanReadWorkflow":        authz.CanReadWorkflow,
		"CanWriteWorkflow":       authz.CanWriteWorkflow,
		"CanReadAccessNetwork":       authz.CanReadAccessNetwork,
		"CanWriteAccessNetwork":      authz.CanWriteAccessNetwork,
		"CanReadAccessTopology":      authz.CanReadAccessTopology,
		"CanWriteAccessTopology":     authz.CanWriteAccessTopology,
		"CanReadServiceProfiles":     authz.CanReadServiceProfiles,
		"CanWriteServiceProfiles":    authz.CanWriteServiceProfiles,
		"CanRunDiagnostics":          authz.CanRunDiagnostics,
		"CanReadAuthentication":      authz.CanReadAuthentication,
		"CanWriteAuthentication":     authz.CanWriteAuthentication,
		"CanReadConnectionProfiles":  authz.CanReadConnectionProfiles,
		"CanWriteConnectionProfiles": authz.CanWriteConnectionProfiles,
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
