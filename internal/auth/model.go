// Package auth models Palladium's authentication identity: the User
// record, password hashing, JWT issuance/validation, and the AuthService
// that ties them together for a login flow.
//
// This is authentication only — establishing who a caller is. It
// deliberately does not decide what an authenticated caller is allowed to
// do: no roles, no permissions, no authorization. Those are a separate,
// later concern (see TASKS.md's Phase 2, where they are tracked as
// distinct items from User model and JWT). Keeping the two apart from the
// start avoids the JWT claims, the User model, and this package's public
// API all needing to change shape again when authorization arrives.
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
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
