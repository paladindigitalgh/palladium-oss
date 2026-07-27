package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// bcryptCost is the work factor passed to bcrypt. bcrypt.DefaultCost (the
// library's own default, currently 10) is used rather than a custom value:
// picking a "stronger" cost without a specific throughput/latency budget in
// mind is just guessing, and the library's default is a deliberately
// chosen, still-current baseline maintained by the Go team.
const bcryptCost = bcrypt.DefaultCost

// HashPassword returns the bcrypt hash of password, suitable for storing
// in User.PasswordHash. Never store a plaintext password.
//
// This and VerifyPassword are plain functions, not a constructor-injected
// type: unlike TokenIssuer (token.go), they need no configuration (no
// secret, no expiration) and no swappable implementation — the task
// specifies bcrypt, not an algorithm to be chosen at runtime — so wrapping
// them in an interface would be exactly the unnecessary abstraction
// CLAUDE.md warns against.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("auth: hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword reports whether password matches hash, as produced by
// HashPassword.
//
// It returns a plain bool, not an error: a non-matching password and a
// malformed hash are both simply "not verified" to the caller. Collapsing
// them is deliberate, not a loss of information — a login flow must not
// distinguish "wrong password" from "the stored hash is unreadable" in its
// response, since doing so is itself a potential information leak, and
// the caller (AuthService) has nothing different to do for either case:
// both mean authentication failed.
func VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
