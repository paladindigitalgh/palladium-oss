// Package bootstrap creates the first administrator account for a new
// Palladium installation.
//
// It is its own package, separate from cmd/bootstrap, specifically so the
// operation itself — refuse if a user already exists, hash the password,
// create the account — is unit-testable without a real terminal.
// cmd/bootstrap's main() only handles interactive prompting (which needs
// a real TTY for masked password input, see golang.org/x/term) and calls
// into this package with the plain strings it collected.
package bootstrap

import (
	"context"
	"fmt"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// Administrator creates the first administrator account. This tool is for
// initial installation only, not general user management (see
// cmd/bootstrap's doc comment): it has one method, and that method
// refuses outright once any User exists.
//
// The account it creates always receives auth.RoleAdministrator — that is
// the entire point of this tool (goal 5: "bootstrap administrator must
// automatically receive the Administrator role"), and is not a caller
// choice: Create takes only an email and a password, with no Role
// parameter for a caller to (mis)set. A brand-new installation has no
// User yet, so nothing could grant a lesser role even if this tool wanted
// to be asked.
type Administrator struct {
	users auth.UserRepository
}

// NewAdministrator builds an Administrator.
func NewAdministrator(users auth.UserRepository) *Administrator {
	return &Administrator{users: users}
}

// Create creates the first User with the given email and password, or
// returns an apperror.KindConflict error if a User already exists.
//
// The existence check (UserRepository.Count) and the Create that follows
// it are not wrapped in a transaction. That is a deliberate simplicity
// choice, not an oversight: this tool is run once, by hand, during initial
// installation (see the package doc comment) — a race between two
// concurrent bootstrap runs is not a scenario worth the added complexity
// of transactional locking, and the worst case if it ever happened is a
// second administrator account, not a security hole. A future general
// user-management feature that runs unattended and repeatedly would need
// to reconsider this; this one-shot tool does not.
func (a *Administrator) Create(ctx context.Context, email, password string) (auth.User, error) {
	if password == "" {
		return auth.User{}, apperror.Invalid("password must not be empty")
	}

	count, err := a.users.Count(ctx)
	if err != nil {
		return auth.User{}, err
	}
	if count > 0 {
		return auth.User{}, apperror.Conflict("refusing to bootstrap: a user account already exists")
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return auth.User{}, fmt.Errorf("bootstrap: hash password: %w", err)
	}

	user := auth.User{Email: email, PasswordHash: hash, Role: auth.RoleAdministrator}
	if err := user.Validate(); err != nil {
		return auth.User{}, err
	}

	return a.users.Create(ctx, user)
}
