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

// CanReadContacts reports whether role may read Contact data. All three
// built-in roles can — identical to CanReadLocations's rule today.
//
// This is a separate function from CanReadLocations/CanReadCustomers, not
// a call to either, for the same reasoning CanReadLocations's own doc
// comment gives for not being implemented in terms of CanReadCustomers:
// Contacts, Locations, and Customers are different resources answering
// different questions, and today's identical answer is a coincidence of
// this being RBAC v1, not a reason to wire them together.
func CanReadContacts(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator, auth.RoleViewer:
		return true
	default:
		return false
	}
}

// CanWriteContacts reports whether role may create, update, or delete
// Contact data. Administrator and Operator can; Viewer cannot. See
// CanReadContacts's doc comment for why this is not implemented in terms
// of CanWriteLocations despite the identical rule today.
func CanWriteContacts(role auth.Role) bool {
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

// CanReadWorkflow reports whether role may read Workflow Instance
// data — the orchestration record for a request to provision, modify,
// suspend, resume, disconnect, or synchronize a Service (see
// internal/workflow). All three built-in roles can — identical to
// CanReadServiceEquipment's rule today.
//
// This is a separate function from CanReadServices, not a call to it: a
// WorkflowInstance references a Service, but "who can see a Service" and
// "who can see the operational history of attempts to change it" are
// different questions that happen to share an answer today for the same
// reason every other capability pair in this file does — a future
// requirement scoped specifically to workflow execution (e.g. a
// field-operations role that can drive workflows without general
// Service visibility) must never require touching Service's code, and
// vice versa.
func CanReadWorkflow(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator, auth.RoleViewer:
		return true
	default:
		return false
	}
}

// CanWriteWorkflow reports whether role may create, update
// (including driving state transitions), or delete Workflow Instance
// data. Administrator and Operator can; Viewer cannot. See
// CanReadWorkflow's doc comment for why this is not implemented in
// terms of CanWriteServices despite the identical rule today.
func CanWriteWorkflow(role auth.Role) bool {
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
// built-in roles can — identical to CanReadWorkflow's rule today.
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
// function from CanReadServices/CanReadServiceEquipment/CanReadWorkflow,
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

// CanReadAccessTopology reports whether role may read Access Topology
// data — AccessInterface and AccessAttachment records alike (see
// internal/accessinterface, internal/accessattachment). All three
// built-in roles can — identical to CanReadAccessNetwork's rule today.
//
// A single capability pair (CanReadAccessTopology/CanWriteAccessTopology)
// guards both resources, the same reasoning authz.CanReadAccessNetwork's
// doc comment gives for AccessNetwork/OLT/PONPort: an AccessAttachment
// only exists nested inside an AccessInterface (see
// accessattachment.AccessAttachment's required AccessInterfaceID), so
// "who can see an interface" and "who can see what's attached to it" are
// the same question asked at two levels of one domain, not two domains
// that happen to share a rule today.
//
// This is a deliberately separate capability from
// CanReadAccessNetwork/CanReadServiceEquipment, not a call to either,
// even though AccessInterface references a PONPort (Access Network's
// domain) and AccessAttachment references a ServiceEquipment record
// (Service Equipment's domain) — the same reasoning
// authz.CanReadServiceEquipment's own doc comment gives for not
// deferring to the domains it merely references: "who can see the
// physical access network topology and what's plugged into it" is its
// own access question, not a restatement of "who can see an OLT" or "who
// can see a piece of subscriber equipment." A future requirement scoped
// specifically to access topology (e.g. a field-operations role that can
// see interface-to-equipment mappings without full Access Network or
// Service Equipment visibility) must never require touching either of
// those domains' code, and vice versa.
func CanReadAccessTopology(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator, auth.RoleViewer:
		return true
	default:
		return false
	}
}

// CanWriteAccessTopology reports whether role may create, update, or
// delete Access Topology data — AccessInterface and AccessAttachment
// records alike. Administrator and Operator can; Viewer cannot. See
// CanReadAccessTopology's doc comment for why one capability pair guards
// both resources, and for why this is not implemented in terms of
// CanWriteAccessNetwork/CanWriteServiceEquipment despite the identical
// rule today.
func CanWriteAccessTopology(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator:
		return true
	default:
		return false
	}
}

// CanReadServiceProfiles reports whether role may read Service Profile
// data — the named, reusable description of a Service's operational
// intent (see internal/serviceprofile). All three built-in roles can —
// identical to CanReadCatalog's rule today.
//
// This is a separate function from CanReadCatalog, not a call to it,
// per this milestone's explicit instruction ("do not reuse Product
// permissions"). A Service references both a Product (see
// internal/product, guarded by CanReadCatalog) and a ServiceProfile —
// "what was sold" and "how it is meant to operate" are different
// business concepts that merely happen to both be referenced by Service
// (see service.Service's doc comment), the same reasoning
// CanReadServices's own doc comment gives for not deferring to
// CanReadCatalog or CanReadLocations. Today's identical answer is a
// coincidence of this being RBAC v1, not a reason to wire the two
// together — a future requirement specific to Service Profiles (e.g. a
// network-engineering role that curates profiles without full Catalog
// visibility) must never require touching Catalog's or Product's code,
// and vice versa.
func CanReadServiceProfiles(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator, auth.RoleViewer:
		return true
	default:
		return false
	}
}

// CanWriteServiceProfiles reports whether role may create, update, or
// delete Service Profile data. Administrator and Operator can; Viewer
// cannot. See CanReadServiceProfiles's doc comment for why this is not
// implemented in terms of CanWriteCatalog despite the identical rule
// today.
func CanWriteServiceProfiles(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator:
		return true
	default:
		return false
	}
}

// CanRunDiagnostics reports whether role may execute a diagnostic (see
// internal/diagnostics) — today, POST /api/v1/diagnostics/basic-onu-check,
// and any future diagnostic endpoint built against the same framework.
// Administrator and Operator can; Viewer cannot.
//
// This is a single capability, not a Read/Write pair like almost every
// other capability in this file — deliberately, because running a
// diagnostic is neither: it is not "read" in the sense of retrieving
// stored data (this milestone's framework has no persistence at all —
// see internal/diagnostics/service's doc comment), and it is not "write"
// in the sense of creating, updating, or deleting a Palladium record.
// It is an active operation that, once real diagnostics exist (SSH
// sessions, CLI commands — explicitly out of scope for this milestone,
// see internal/diagnostics's package doc comment), will reach out and
// interact with live network equipment. That makes it categorically
// closer to every "write" capability's Administrator-and-Operator rule
// than to any "read" capability's three-role rule: a Viewer's defining
// property throughout this codebase is that they can see but never act,
// and running a diagnostic — even today's placeholder — is an action.
func CanRunDiagnostics(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator:
		return true
	default:
		return false
	}
}

// CanReadAuthentication reports whether role may read Authentication data
// — the reusable credential records future infrastructure components
// (Connection Profiles today; SSH-driven vendor adapters later) resolve
// by ID (see internal/authentication). All three built-in roles can —
// identical to CanReadServiceProfiles's rule today.
//
// This is a separate function from every other capability in this file,
// not a call to any of them, for the same reason CanReadServiceProfiles's
// own doc comment gives: a Connection Profile references an Authentication
// record (see connectionprofile.ConnectionProfile's AuthenticationID), but
// "who can see a stored credential" and "who can see a connection's
// non-secret settings" are different questions that merely happen to
// share an answer today — see CanReadConnectionProfiles's doc comment for
// the same reasoning applied in the other direction. A future requirement
// specific to credential visibility (e.g. restricting who can even see
// that a credential exists, independent of Connection Profile access)
// must never require touching Connection Profile's code, and vice versa.
//
// Note this governs only whether the Authentication record itself (Name,
// AuthenticationType, Username, HasPassword/HasPrivateKey — see
// internal/authentication/httpapi's response DTO) can be read, never the
// plaintext Password or PrivateKey: those are never returned over HTTP to
// any role, by design (see internal/authentication/httpapi/dto.go).
func CanReadAuthentication(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator, auth.RoleViewer:
		return true
	default:
		return false
	}
}

// CanWriteAuthentication reports whether role may create, update, or
// delete Authentication data. Administrator and Operator can; Viewer
// cannot. See CanReadAuthentication's doc comment for why this is not
// implemented in terms of any other domain's write capability despite the
// identical rule today.
func CanWriteAuthentication(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator:
		return true
	default:
		return false
	}
}

// CanReadConnectionProfiles reports whether role may read Connection
// Profile data — the reusable, non-secret connection settings (protocol,
// port, timeout, host key policy) future infrastructure components will
// use, together with an optional reference to an Authentication record
// (see internal/connectionprofile). All three built-in roles can —
// identical to CanReadAuthentication's rule today.
//
// This is a separate function from CanReadAuthentication, not a call to
// it, even though a ConnectionProfile references an Authentication record
// by ID — see CanReadAuthentication's doc comment for the reasoning in
// the other direction. A future requirement specific to Connection
// Profile visibility (e.g. a network-engineering role that manages
// connection settings without being able to see which credentials exist)
// must never require touching Authentication's code, and vice versa.
func CanReadConnectionProfiles(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator, auth.RoleViewer:
		return true
	default:
		return false
	}
}

// CanWriteConnectionProfiles reports whether role may create, update, or
// delete Connection Profile data. Administrator and Operator can; Viewer
// cannot. See CanReadConnectionProfiles's doc comment for why this is not
// implemented in terms of CanWriteAuthentication despite the identical
// rule today.
func CanWriteConnectionProfiles(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator:
		return true
	default:
		return false
	}
}

// CanReadEvents reports whether role may read Event data — the immutable
// operational history behind Timeline sections (see internal/event). All
// three built-in roles can: an Event only ever describes something that
// already happened, mirroring the read side of every other domain's
// capability pair. There is no CanWriteEvents: events are written
// internally by domain/workflow code, never through a public write route
// (see internal/event/httpapi's package doc comment), so there is
// nothing for a write capability to guard.
func CanReadEvents(role auth.Role) bool {
	switch role {
	case auth.RoleAdministrator, auth.RoleOperator, auth.RoleViewer:
		return true
	default:
		return false
	}
}
