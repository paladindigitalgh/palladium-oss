package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
)

// authService is the seam LoginHandler depends on instead of a concrete
// *auth.AuthService — the same reasoning internal/inventory/httpapi's
// siteService interface documents: it lets handler tests exercise HTTP
// behavior (status codes, JSON shape, error mapping) against a fake, with
// no real service, repository, or database involved.
// internal/auth/service_test.go already covers AuthService.Authenticate's
// own behavior (unknown email, wrong password, success) at the business
// logic layer — reusing that instead of duplicating it here is exactly
// this milestone's "reuse the existing AuthService" (goal 3) and "do not
// move business logic into HTTP handlers".
type authService interface {
	Authenticate(ctx context.Context, email, password string) (string, error)
}

// LoginHandler serves POST /api/v1/auth/login.
type LoginHandler struct {
	auth      authService
	expiresIn time.Duration
}

// NewLoginHandler builds a LoginHandler. expiresIn is the same duration
// the injected authService's underlying TokenIssuer was configured with
// (see cmd/server/main.go's wiring) — LoginHandler does not parse the
// issued token to discover its own expiration; it is simply told what
// that configuration already is, so the response's expiresIn field is
// exact without this handler needing to know anything about JWTs.
func NewLoginHandler(auth authService, expiresIn time.Duration) *LoginHandler {
	return &LoginHandler{auth: auth, expiresIn: expiresIn}
}

// Login handles POST /api/v1/auth/login. It stays thin (goal 4): decode,
// delegate to AuthService, translate the result — no business logic here.
// AuthService.Authenticate already returns the same error for an unknown
// email and an incorrect password (see its doc comment in
// internal/auth/service.go), so this handler needs no special-casing to
// keep that guarantee: whatever Kind comes back is translated by
// httpx.WriteError exactly like any other error.
func (h *LoginHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	token, err := h.auth.Authenticate(r.Context(), req.Email, req.Password)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, loginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int(h.expiresIn.Seconds()),
	})
}
