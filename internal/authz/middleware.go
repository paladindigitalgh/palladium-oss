package authz

import (
	"net/http"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// Capability is a predicate over a Role — CanReadInventory,
// CanWriteInventory, or CanManageUsers, from capabilities.go. It exists as
// a named type only so Middleware.Require's signature reads clearly; it
// is still just `func(auth.Role) bool` underneath, not an opening for
// arbitrary runtime-supplied logic (see capabilities.go's doc comment on
// "no policy evaluation").
type Capability func(auth.Role) bool

// Middleware builds authorization middleware for specific capabilities.
//
// It depends on auth.UserRepository so it can look up the authenticated
// caller's current Role. This is a real, deliberate cost, not an
// oversight: goal 6 ("JWT claims remain unchanged, do not add roles to
// JWT yet") means Role cannot be read from the token the way UserID and
// Email already are (see auth.Claims) — a database lookup is the only
// place left to get it. The upside of that constraint is correctness a
// token-embedded Role would not have: if an administrator changes a
// user's Role, the change takes effect on that user's very next request,
// rather than only after their current token expires (this codebase has
// no refresh tokens, so a stale token-embedded Role could otherwise
// persist for the full JWT_EXPIRATION window — 24h by default). A future
// milestone might reconsider embedding Role in the token as a performance
// optimization once that trade-off matters; this one takes the correct,
// simple option.
type Middleware struct {
	users auth.UserRepository
}

// NewMiddleware builds a Middleware.
func NewMiddleware(users auth.UserRepository) *Middleware {
	return &Middleware{users: users}
}

// Require returns HTTP middleware that allows a request only if the
// authenticated caller's Role satisfies capability, and responds 403
// Forbidden otherwise.
//
// It reuses auth.Middleware for authentication rather than duplicating
// any JWT validation (goal 3): it reads the Claims auth.Middleware already
// stored in the request context via auth.ClaimsFromContext, so this
// middleware must run after auth.Middleware in the chain — see
// internal/server/router.go's /sites wiring for how the two compose. If
// no Claims are present (auth.Middleware was skipped or never ran), this
// fails closed with 401 rather than treating the request as anonymous.
func (m *Middleware) Require(capability Capability) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.ClaimsFromContext(r.Context())
			if !ok {
				httpx.WriteError(w, apperror.Unauthorized("authentication required"))
				return
			}

			user, err := m.users.GetByID(r.Context(), claims.UserID)
			if err != nil {
				httpx.WriteError(w, err)
				return
			}

			if !capability(user.Role) {
				httpx.WriteError(w, apperror.Forbidden("you do not have permission to perform this action"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireInventoryRead returns middleware allowing any Role that
// CanReadInventory (Administrator, Operator, Viewer).
func (m *Middleware) RequireInventoryRead() func(http.Handler) http.Handler {
	return m.Require(CanReadInventory)
}

// RequireInventoryWrite returns middleware allowing any Role that
// CanWriteInventory (Administrator, Operator).
func (m *Middleware) RequireInventoryWrite() func(http.Handler) http.Handler {
	return m.Require(CanWriteInventory)
}

// RequireUserManagement returns middleware allowing any Role that
// CanManageUsers (Administrator only). Unused today (see
// CanManageUsers's doc comment) — provided now so the eventual user
// management endpoints only need to apply it, not invent it.
func (m *Middleware) RequireUserManagement() func(http.Handler) http.Handler {
	return m.Require(CanManageUsers)
}
