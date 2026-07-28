// Package authz decides what an auth.Role is allowed to do. It is
// RBAC v1: intentionally the simplest thing that can work, per this
// milestone's explicit constraints.
//
//   - "Do not build a generic permission system": there is no table of
//     permission strings, no lookup by name, no way to define a new
//     capability without writing a new Go function. Each capability below
//     is a concrete, named, statically-typed predicate. Adding one means
//     editing this file, not configuring data — which is the point: a
//     change to who-can-do-what is a reviewable code change, not a runtime
//     configuration that can drift from what the code actually enforces.
//   - "Do not build policy evaluation": there is no expression language, no
//     rule engine, nothing "evaluated" — just switch statements returning
//     bool.
//   - "No role hierarchy": Administrator is not implemented as "Operator
//     plus more" or via inheritance; every capability's switch lists every
//     Role that has it explicitly, even when that means repeating
//     RoleAdministrator in every case. A hierarchy is a reasonable thing to
//     want eventually, but it is a design decision this milestone was
//     explicitly told not to make, and listing roles out longhand costs
//     nothing at three roles and two capabilities.
//
// Business code asks these questions (CanReadInventory(role), etc.); it
// never compares a Role against a string literal itself — see this
// package's own doc goal: "do not expose raw string comparisons
// throughout the application." That comparison happens in exactly one
// place per capability, here.
package authz

import "github.com/paladindigitalgh/palladium-oss/internal/auth"

// CanReadInventory reports whether role may read Inventory data (Sites
// today; the same rule will apply to Buildings, Rooms, Racks, and Devices
// once they have HTTP endpoints). All three built-in roles can.
func CanReadInventory(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator, auth.RoleViewer:
		return true
	default:
		return false
	}
}

// CanWriteInventory reports whether role may create, update, or delete
// Inventory data. Administrator and Operator can; Viewer cannot — a
// Viewer can see the inventory but never change it, which is the entire
// reason that role exists as distinct from Operator.
func CanWriteInventory(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator:
		return true
	default:
		return false
	}
}

// CanManageUsers reports whether role may administer user accounts. Only
// Administrator can. Nothing calls this yet — there is no user management
// API (explicitly out of scope for this milestone) — but goal 2 asks for
// it by name as a capability this package answers, so it exists as the
// building block a future user management endpoint will guard itself
// with, exactly like CanReadInventory/CanWriteInventory guard the Site
// endpoints today.
func CanManageUsers(role auth.Role) bool {
	return role == auth.RoleAdministrator
}

// CanReadCustomers reports whether role may read Customer data. All three
// built-in roles can — identical to CanReadInventory's rule today.
//
// This is a separate function from CanReadInventory, not a call to it,
// even though the two currently return the same answer for every Role.
// Customers and Inventory are different resources (see
// internal/customer's package doc comment on why a Customer never
// references Inventory), and "who can read Customer data" is a different
// question from "who can read Inventory data" that only happens to share
// an answer right now — e.g. a future privacy requirement could restrict
// Customer reads to Administrator alone without that having any business
// touching Inventory's rule at all. Reusing CanReadInventory here would
// wire those two unrelated questions together by accident.
func CanReadCustomers(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator, auth.RoleViewer:
		return true
	default:
		return false
	}
}

// CanWriteCustomers reports whether role may create, update, or delete
// Customer data. Administrator and Operator can; Viewer cannot. See
// CanReadCustomers's doc comment for why this is not implemented in terms
// of CanWriteInventory despite the identical rule today.
func CanWriteCustomers(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator:
		return true
	default:
		return false
	}
}

// CanReadLocations reports whether role may read Location data. All three
// built-in roles can — identical to CanReadCustomers's rule today.
//
// This is a separate function from CanReadCustomers, not a call to it,
// even though the two currently return the same answer for every Role —
// the Location milestone's explicit instruction, matching the reasoning
// CanReadCustomers's own doc comment already gives for not being
// implemented in terms of CanReadInventory: Locations and Customers are
// different resources answering different questions ("who can see where
// a Customer's equipment lives" is not the same question as "who can see
// a Customer's own record"), and today's identical answer is a
// coincidence of this being RBAC v1, not a reason to wire the two
// together. A future requirement affecting one must never have to touch
// the other's code to stay correct.
func CanReadLocations(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator, auth.RoleViewer:
		return true
	default:
		return false
	}
}

// CanWriteLocations reports whether role may create, update, or delete
// Location data. Administrator and Operator can; Viewer cannot. See
// CanReadLocations's doc comment for why this is not implemented in terms
// of CanWriteCustomers despite the identical rule today.
func CanWriteLocations(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator:
		return true
	default:
		return false
	}
}

// CanReadCatalog reports whether role may read Product Catalog data —
// both ProductCatalog and Product records (see internal/catalog and
// internal/product). All three built-in roles can — identical to
// CanReadLocations's rule today.
//
// A single capability pair (CanReadCatalog/CanWriteCatalog) guards both
// resources, unlike Location's own dedicated pair separate from
// Customer's: ProductCatalog and Product are not two different domains
// that happen to share a rule the way Location and Customer are — a
// Product only exists nested inside a ProductCatalog (see
// product.Product's required CatalogID), so "who can see the catalog" and
// "who can see a product in it" are the same question asked at two
// levels of one domain, not two questions that could plausibly diverge.
// This is a separate function from CanReadCustomers/CanReadLocations/
// CanReadInventory, not a call to any of them, per this milestone's
// explicit instruction — even though today's answer is identical, the
// Catalog domain answering a future access question differently (e.g. a
// distributor-facing read-only integration) must never require touching
// Customer's, Location's, or Inventory's code.
func CanReadCatalog(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator, auth.RoleViewer:
		return true
	default:
		return false
	}
}

// CanWriteCatalog reports whether role may create, update, or delete
// Product Catalog data — both ProductCatalog and Product records.
// Administrator and Operator can; Viewer cannot. See CanReadCatalog's doc
// comment for why one capability pair guards both resources, and for why
// this is not implemented in terms of CanWriteCustomers/CanWriteLocations/
// CanWriteInventory despite the identical rule today.
func CanWriteCatalog(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator:
		return true
	default:
		return false
	}
}

// CanReadServices reports whether role may read Service data — a
// subscriber's purchased Service (see internal/service). All three
// built-in roles can — identical to CanReadCatalog's rule today.
//
// This is a separate function from CanReadCatalog/CanReadLocations/
// CanReadCustomers/CanReadInventory, not a call to any of them, per this
// milestone's explicit instruction. Unlike Catalog and Product, which
// share one capability because a Product only exists nested inside a
// ProductCatalog (see CanReadCatalog's doc comment), a Service is not
// "part of" a Location or a Product the way a Product is part of a
// Catalog — a Service is its own resource that happens to reference both
// (see internal/service's package doc comment on why it has no
// CustomerID: the Customer relationship is a join through Location, not
// ownership). "Who can see a subscriber's purchased Service" is
// genuinely its own access question, not a restatement of "who can see a
// Location" or "who can see a Product" — a future requirement (e.g.
// restricting Service visibility for billing-sensitivity reasons) must
// never require touching Location's, Product's, Customer's, or
// Inventory's code.
func CanReadServices(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator, auth.RoleViewer:
		return true
	default:
		return false
	}
}

// CanWriteServices reports whether role may create, update, or delete
// Service data. Administrator and Operator can; Viewer cannot. See
// CanReadServices's doc comment for why this is not implemented in terms
// of CanWriteCatalog/CanWriteLocations/CanWriteCustomers/CanWriteInventory
// despite the identical rule today.
func CanWriteServices(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator:
		return true
	default:
		return false
	}
}

// CanReadServiceEquipment reports whether role may read Service
// Equipment data — the link between a Service and the inventory.Device
// delivering it (see internal/serviceequipment). All three built-in
// roles can — identical to CanReadServices's rule today.
//
// This is a separate function from CanReadServices, not a call to it, per
// this milestone's explicit instruction ("do not reuse Service
// capabilities"). Service Equipment sits at the intersection of two
// domains — Service and Inventory — and answering "who can see this link"
// by deferring to either one's rule would wire Service Equipment's access
// question to a domain it merely references, the same reasoning
// CanReadServices's own doc comment gives for not deferring to Location's
// or Product's rule. A future requirement specific to physical equipment
// visibility (e.g. field technicians needing read access without full
// Service visibility) must never require touching Service's code, and
// vice versa.
func CanReadServiceEquipment(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator, auth.RoleViewer:
		return true
	default:
		return false
	}
}

// CanWriteServiceEquipment reports whether role may create, update, or
// delete Service Equipment data. Administrator and Operator can; Viewer
// cannot. See CanReadServiceEquipment's doc comment for why this is not
// implemented in terms of CanWriteServices despite the identical rule
// today.
func CanWriteServiceEquipment(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator:
		return true
	default:
		return false
	}
}

// CanReadProvisioning reports whether role may read Provisioning Job
// data — the orchestration record for a request to provision, modify,
// suspend, resume, disconnect, or synchronize a Service (see
// internal/provisioning). All three built-in roles can — identical to
// CanReadServiceEquipment's rule today.
//
// This is a separate function from CanReadServices, not a call to it, per
// this milestone's explicit instruction ("do not reuse Service
// permissions"). Provisioning is called out explicitly as "an
// operational concern [that] deserves its own authorization boundary":
// a ProvisioningJob references a Service, but "who can see a Service"
// and "who can see the operational history of attempts to provision
// it" are different questions that happen to share an answer today for
// the same reason every other capability pair in this file does — a
// future requirement scoped specifically to provisioning (e.g. a
// field-operations role that can drive provisioning without general
// Service visibility) must never require touching Service's code, and
// vice versa.
func CanReadProvisioning(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator, auth.RoleViewer:
		return true
	default:
		return false
	}
}

// CanWriteProvisioning reports whether role may create, update
// (including driving state transitions), or delete Provisioning Job
// data. Administrator and Operator can; Viewer cannot. See
// CanReadProvisioning's doc comment for why this is not implemented in
// terms of CanWriteServices despite the identical rule today.
func CanWriteProvisioning(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator:
		return true
	default:
		return false
	}
}

// CanReadAccessNetwork reports whether role may read Access Network
// data — AccessNetwork, OLT, and PONPort records alike (see
// internal/accessnetwork, internal/olt, internal/ponport). All three
// built-in roles can — identical to CanReadProvisioning's rule today.
//
// A single capability pair (CanReadAccessNetwork/CanWriteAccessNetwork)
// guards all three resources, the same reasoning
// authz.CanReadCatalog's doc comment gives for Catalog and Product: an
// OLT only exists nested inside an AccessNetwork (see olt.OLT's required
// AccessNetworkID), and a PONPort only exists nested inside an OLT (see
// ponport.PONPort's required OLTID) — so "who can see the access
// network," "who can see an OLT in it," and "who can see a port on that
// OLT" are the same question asked at three levels of one domain, not
// three domains that happen to share a rule today. This is a separate
// function from CanReadServices/CanReadServiceEquipment/CanReadProvisioning,
// not a call to any of them, for the same reason those are each separate
// from one another — a future access requirement specific to the
// physical access network (e.g. field technicians needing OLT
// visibility without Service visibility) must never require touching
// another domain's code.
func CanReadAccessNetwork(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator, auth.RoleViewer:
		return true
	default:
		return false
	}
}

// CanWriteAccessNetwork reports whether role may create, update, or
// delete Access Network data — AccessNetwork, OLT, and PONPort records
// alike. Administrator and Operator can; Viewer cannot. See
// CanReadAccessNetwork's doc comment for why one capability pair guards
// all three resources, and for why this is not implemented in terms of
// any other domain's capability despite the identical rule today.
func CanWriteAccessNetwork(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator:
		return true
	default:
		return false
	}
}
