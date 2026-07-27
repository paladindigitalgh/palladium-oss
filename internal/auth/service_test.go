package auth_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
)

// fakeUserRepository is an in-memory auth.UserRepository. AuthService's
// logic (branch on NotFound vs a failed password check, then issue a
// token) is independent of storage, so its unit tests use this rather than
// the real PostgreSQL implementation in internal/auth/postgres — that one
// has its own integration tests (user_test.go) that exercise the database
// itself. This fake still returns apperror.NotFound exactly like the real
// repository does, so AuthService is tested against the same contract it
// would see in production, not a shortcut around it.
type fakeUserRepository struct {
	mu      sync.Mutex
	byEmail map[string]auth.User
}

func newFakeUserRepository(users ...auth.User) *fakeUserRepository {
	f := &fakeUserRepository{byEmail: make(map[string]auth.User)}
	for _, u := range users {
		f.byEmail[u.Email] = u
	}
	return f
}

func (f *fakeUserRepository) GetByID(_ context.Context, id uuid.UUID) (auth.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.byEmail {
		if u.ID == id {
			return u, nil
		}
	}
	return auth.User{}, apperror.NotFound("user not found")
}

func (f *fakeUserRepository) GetByEmail(_ context.Context, email string) (auth.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byEmail[email]
	if !ok {
		return auth.User{}, apperror.NotFound("user not found")
	}
	return u, nil
}

func (f *fakeUserRepository) Create(_ context.Context, user auth.User) (auth.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	f.byEmail[user.Email] = user
	return user, nil
}

func (f *fakeUserRepository) UpdatePasswordHash(_ context.Context, id uuid.UUID, passwordHash string) (auth.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for email, u := range f.byEmail {
		if u.ID == id {
			u.PasswordHash = passwordHash
			f.byEmail[email] = u
			return u, nil
		}
	}
	return auth.User{}, apperror.NotFound("user not found")
}

func (f *fakeUserRepository) Count(context.Context) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.byEmail), nil
}

var _ auth.UserRepository = (*fakeUserRepository)(nil)

func newTestUser(t *testing.T, email, password string) auth.User {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() = %v", err)
	}
	return auth.User{ID: uuid.New(), Email: email, PasswordHash: hash}
}

func TestAuthServiceAuthenticateSucceeds(t *testing.T) {
	user := newTestUser(t, "jane@example.com", "correct password")
	repo := newFakeUserRepository(user)
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(fixedNow))
	service := auth.NewAuthService(repo, tokens)

	token, err := service.Authenticate(context.Background(), "jane@example.com", "correct password")
	if err != nil {
		t.Fatalf("Authenticate() = %v", err)
	}
	if token == "" {
		t.Fatal("Authenticate() returned an empty token")
	}

	claims, err := tokens.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() = %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("UserID = %v, want %v", claims.UserID, user.ID)
	}
	if claims.Email != user.Email {
		t.Errorf("Email = %q, want %q", claims.Email, user.Email)
	}
}

func TestAuthServiceAuthenticateFailsForUnknownEmail(t *testing.T) {
	repo := newFakeUserRepository() // empty
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(fixedNow))
	service := auth.NewAuthService(repo, tokens)

	_, err := service.Authenticate(context.Background(), "nobody@example.com", "whatever")

	assertUnauthorized(t, err)
}

func TestAuthServiceAuthenticateFailsForWrongPassword(t *testing.T) {
	user := newTestUser(t, "jane@example.com", "correct password")
	repo := newFakeUserRepository(user)
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(fixedNow))
	service := auth.NewAuthService(repo, tokens)

	_, err := service.Authenticate(context.Background(), "jane@example.com", "wrong password")

	assertUnauthorized(t, err)
}

// TestAuthServiceAuthenticateDoesNotRevealWhichCaseOccurred is the concrete
// check behind AuthService.Authenticate's doc comment: an unknown email and
// a known email with the wrong password must be indistinguishable to the
// caller, so neither response leaks which registered email addresses
// exist.
func TestAuthServiceAuthenticateDoesNotRevealWhichCaseOccurred(t *testing.T) {
	user := newTestUser(t, "jane@example.com", "correct password")
	repo := newFakeUserRepository(user)
	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(fixedNow))
	service := auth.NewAuthService(repo, tokens)

	_, unknownEmailErr := service.Authenticate(context.Background(), "nobody@example.com", "whatever")
	_, wrongPasswordErr := service.Authenticate(context.Background(), "jane@example.com", "wrong password")

	if unknownEmailErr == nil || wrongPasswordErr == nil {
		t.Fatal("expected both authentication attempts to fail")
	}
	if unknownEmailErr.Error() != wrongPasswordErr.Error() {
		t.Errorf("error messages differ: unknown email = %q, wrong password = %q; a caller could tell them apart",
			unknownEmailErr.Error(), wrongPasswordErr.Error())
	}
}
