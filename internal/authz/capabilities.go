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
