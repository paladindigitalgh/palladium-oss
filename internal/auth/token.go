package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
)

// Claims are the JWT claims Palladium issues: user ID, email, issued-at,
// and expiration — nothing else. Embedding jwt.RegisteredClaims (rather
// than hand-implementing the jwt.Claims interface) and setting only
// IssuedAt and ExpiresAt is what keeps this minimal in practice: every
// other RegisteredClaims field (iss, sub, aud, nbf, jti) is tagged
// `omitempty` by the library, so leaving them unset means they are absent
// from the encoded token entirely, not merely empty. Deliberately no
// roles, no permissions: this milestone is authentication (who is this?),
// not authorization (what can they do?) — see model.go.
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims
}

// TokenIssuer issues and validates JWTs.
//
// Unlike HashPassword/VerifyPassword, this is a constructor-injected type,
// not plain functions: it needs configuration — a signing secret and an
// expiration duration, both of which vary per environment and per this
// milestone's "make expiration configurable" goal — plus a clock.Clock for
// the same reason SiteRepository injects one (internal/inventory/postgres/
// site.go): issued-at and expiration are timestamps, and a test that wants
// to assert an exact expiration or exercise "this token has expired"
// deterministically needs to control what TokenIssuer considers "now",
// both when it issues a token and when it later validates one.
type TokenIssuer struct {
	secret     []byte
	expiration time.Duration
	clock      clock.Clock
}

// NewTokenIssuer builds a TokenIssuer. secret signs and verifies every
// token issued or parsed through it; expiration controls how long an
// issued token remains valid from the moment it is issued.
func NewTokenIssuer(secret []byte, expiration time.Duration, clock clock.Clock) *TokenIssuer {
	return &TokenIssuer{secret: secret, expiration: expiration, clock: clock}
}

// IssueToken returns a signed JWT for user, valid for the configured
// expiration duration starting now.
func (i *TokenIssuer) IssueToken(user User) (string, error) {
	now := i.clock.Now()

	claims := Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(i.expiration)),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(i.secret)
	if err != nil {
		return "", fmt.Errorf("auth: sign token: %w", err)
	}
	return signed, nil
}

// ParseToken validates token's signature and expiration and returns its
// claims, or an apperror.KindUnauthorized error if the token is malformed,
// unsigned by this issuer's secret, or expired.
//
// jwt.WithValidMethods pins acceptable signing algorithms to HS256
// explicitly. This is a deliberate defense, not boilerplate: JWT libraries
// that trust the "alg" header found inside the token itself are vulnerable
// to algorithm-confusion attacks (e.g. a token claiming "alg: none", or one
// signed with a different algorithm than the server expects); pinning the
// method here means a token can only ever be accepted if it was signed
// with exactly the algorithm this issuer uses.
//
// jwt.WithTimeFunc(i.clock.Now) makes expiration validation use the same
// injected clock as IssueToken, rather than the wall clock — required for
// ParseToken's expired-token behavior to be testable without sleeping.
func (i *TokenIssuer) ParseToken(token string) (Claims, error) {
	var claims Claims

	_, err := jwt.ParseWithClaims(token, &claims,
		func(*jwt.Token) (any, error) { return i.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		jwt.WithTimeFunc(i.clock.Now),
	)
	if err != nil {
		return Claims{}, apperror.Wrap(apperror.KindUnauthorized, "invalid token", err)
	}

	return claims, nil
}
