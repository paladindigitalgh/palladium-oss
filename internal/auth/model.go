// Package auth models Palladium's authentication identity: the User
// record (including its Role, see role.go), password hashing, JWT
// issuance/validation, and the AuthService that ties them together for a
// login flow.
//
// This package establishes who a caller is and, as of RBAC v1, what
// single Role they hold — but it does not decide what that Role is
// allowed to do. That is internal/authz's job, kept deliberately separate
// so authentication (identity) and authorization (permission) stay two
// different questions even though a User now carries both a credential
// and a Role: this package could not answer "can this caller delete a
// Site?" without importing every domain that has an opinion about that,
// which would turn a small identity package into a hub every feature
// depends on. internal/authz depends on this package for the Role type;
// this package depends on nothing authorization-related.
package auth

import (
	"time"

	"github.com/google/uuid"
)

// User is an authentication identity. Email is the unique identity a
// caller logs in with; PasswordHash is a bcrypt hash (see password.go) —
// this package never stores or handles a plaintext password beyond the
// single call that hashes or verifies it.
//
// This does not embed inventory.Metadata or reuse any Inventory type:
// auth and inventory are separate domains (CLAUDE.md: "Never couple
// inventory directly to customers" — the same separation of concerns
// applies here), and User's fields don't match Metadata's shape (Email/
// PasswordHash, not Name/Description) even if they did share a package.
// Role holds exactly one value (see role.go) — not a set, not a slice.
// "Store exactly one Role on each User" is this milestone's explicit
// scope: no multiple roles per user, no role hierarchy. If that ever
// needs to change, it is a deliberate, visible schema and model change,
// not something that can silently happen by appending to a collection.
//
// Status (see status.go) was added alongside User Management: whether a
// User may authenticate is a separate question from what Role they hold,
// the same separation Role itself draws from authz's capabilities.
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Role         Role
	Status       UserStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
