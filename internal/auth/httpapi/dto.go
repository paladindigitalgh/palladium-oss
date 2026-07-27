// Package httpapi is the auth domain's REST layer. It mirrors
// internal/inventory/httpapi's shape (a small package holding DTOs plus
// one handler per resource/action, depending on a service via an
// unexported interface for testability) rather than living directly in
// internal/auth alongside auth.Middleware: the login endpoint has request
// and response DTOs and its own testable seam, structurally the same
// amount of HTTP-layer code as a full CRUD resource's handler, not the
// single self-contained function auth.Middleware is. Splitting it out
// keeps internal/auth itself focused on the domain (model, validation,
// hashing, tokens, the login business logic) rather than growing into a
// mixed bag of unrelated files.
package httpapi

// loginRequest is the JSON body for POST /api/v1/auth/login.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginResponse is returned on a successful login.
//
// Its JSON field names are camelCase (accessToken, tokenType, expiresIn),
// unlike internal/inventory/httpapi's snake_case DTOs (created_at,
// updated_at). That inconsistency is not a choice made here: it is
// dictated by this milestone's exact specification for this endpoint, and
// is reproduced deliberately rather than "corrected" to snake_case for
// internal consistency.
//
// PasswordHash never appears here, or anywhere in this package — the only
// domain type loginResponse is built from is the token string
// AuthService.Authenticate already returns, which never included the hash
// to begin with.
type loginResponse struct {
	AccessToken string `json:"accessToken"`
	TokenType   string `json:"tokenType"`
	ExpiresIn   int    `json:"expiresIn"`
}
