package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// This file is the one place in this package that touches net/http. That
// mirrors internal/health and internal/inventory, which each pair a pure
// domain concern with a thin HTTP adapter in the same small package,
// rather than spinning up a separate package for one file's worth of
// glue. internal/server (the router) only ever imports and mounts this;
// it never contains authentication logic itself, keeping it "no business
// logic" as its own doc comment already states.

type contextKey int

const claimsContextKey contextKey = iota

// ContextWithClaims returns a copy of ctx carrying claims. Exported so
// tests can build a context as if a request had already passed through
// Middleware, without needing a real token.
func ContextWithClaims(ctx context.Context, claims Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

// ClaimsFromContext retrieves the Claims Middleware stored, if any.
func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(Claims)
	return claims, ok
}

// Middleware returns HTTP middleware that requires a valid JWT — signed by
// tokens, not expired — on every request it wraps, and rejects everything
// else with 401. On success, it stores the token's Claims in the request
// context for downstream handlers to read via ClaimsFromContext.
//
// This is authentication only: establishing who is calling, and refusing
// the request outright if that cannot be established. It does not decide
// what that caller is allowed to do — there is no role or permission
// check, and deliberately no UserRepository dependency here: unlike an
// authorization check, which would need to look up the user's current
// permissions from storage, authentication only needs what the token
// itself already asserts. That is also why this takes *TokenIssuer, not
// *AuthService: AuthService additionally knows how to verify a
// email/password login, which a request that already carries a token has
// no use for.
func Middleware(tokens *TokenIssuer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				httpx.WriteError(w, apperror.Unauthorized("missing or malformed Authorization header"))
				return
			}

			claims, err := tokens.ParseToken(token)
			if err != nil {
				httpx.WriteError(w, err)
				return
			}

			next.ServeHTTP(w, r.WithContext(ContextWithClaims(r.Context(), claims)))
		})
	}
}

func bearerToken(r *http.Request) (string, bool) {
	const prefix = "Bearer "

	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}

	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", false
	}
	return token, true
}
