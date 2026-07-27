package auth

import (
	"context"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// AuthService authenticates users and issues JWTs. It is the one place
// that combines a UserRepository lookup, password verification, and token
// issuance into a single login flow; HashPassword, VerifyPassword, and
// TokenIssuer each stay independently usable by other future callers
// (e.g. a registration flow needs HashPassword and UserRepository.Create,
// not this service) — see password.go and token.go.
//
// Scope note: this milestone's goal for AuthService is authenticating
// email/password and generating JWTs specifically. It has no
// Register/CreateUser method: creating a User is just HashPassword
// followed by UserRepository.Create, which does not need a service to
// coordinate, and adding one here would be scope beyond what was asked.
type AuthService struct {
	users  UserRepository
	tokens *TokenIssuer
}

// NewAuthService builds an AuthService.
func NewAuthService(users UserRepository, tokens *TokenIssuer) *AuthService {
	return &AuthService{users: users, tokens: tokens}
}

// Authenticate looks up the User with the given email, verifies password
// against its stored hash, and returns a signed JWT on success.
//
// An unknown email and a correct email with the wrong password both return
// the exact same apperror.KindUnauthorized error. This is deliberate:
// returning a different error for "no such user" than for "wrong password"
// lets a caller enumerate which email addresses are registered, a
// well-known authentication information leak. The two cases run through
// completely different internal logic (a UserRepository NotFound vs a
// failed VerifyPassword), but produce one indistinguishable outcome.
func (s *AuthService) Authenticate(ctx context.Context, email, password string) (string, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if apperror.Is(err, apperror.KindNotFound) {
			return "", errInvalidCredentials()
		}
		return "", err
	}

	if !VerifyPassword(user.PasswordHash, password) {
		return "", errInvalidCredentials()
	}

	return s.tokens.IssueToken(user)
}

func errInvalidCredentials() error {
	return apperror.Unauthorized("invalid email or password")
}
