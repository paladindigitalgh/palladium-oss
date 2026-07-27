package bootstrap_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/auth/bootstrap"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeUserRepository is an in-memory auth.UserRepository, the same
// pattern internal/auth/service_test.go already uses for AuthService's
// unit tests: it lets Administrator's orchestration logic (refuse if
// count > 0, hash before create, validate before create) be tested
// without a real database. internal/auth/bootstrap_integration_test.go
// covers the same operation against real PostgreSQL, including proving
// Count's SQL is correct and that the created account can actually log
// in.
type fakeUserRepository struct {
	users []auth.User
}

func (f *fakeUserRepository) GetByID(_ context.Context, id uuid.UUID) (auth.User, error) {
	for _, u := range f.users {
		if u.ID == id {
			return u, nil
		}
	}
	return auth.User{}, apperror.NotFound("user not found")
}

func (f *fakeUserRepository) GetByEmail(_ context.Context, email string) (auth.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return auth.User{}, apperror.NotFound("user not found")
}

func (f *fakeUserRepository) Create(_ context.Context, user auth.User) (auth.User, error) {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	f.users = append(f.users, user)
	return user, nil
}

func (f *fakeUserRepository) UpdatePasswordHash(_ context.Context, id uuid.UUID, passwordHash string) (auth.User, error) {
	for i, u := range f.users {
		if u.ID == id {
			f.users[i].PasswordHash = passwordHash
			return f.users[i], nil
		}
	}
	return auth.User{}, apperror.NotFound("user not found")
}

func (f *fakeUserRepository) Count(context.Context) (int, error) {
	return len(f.users), nil
}

var _ auth.UserRepository = (*fakeUserRepository)(nil)

func TestAdministratorCreateSucceedsWhenNoUsersExist(t *testing.T) {
	repo := &fakeUserRepository{}
	admin := bootstrap.NewAdministrator(repo)

	user, err := admin.Create(context.Background(), "admin@example.com", "correct horse battery staple")
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if user.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if user.Email != "admin@example.com" {
		t.Errorf("Email = %q, want %q", user.Email, "admin@example.com")
	}
	if user.Role != auth.RoleAdministrator {
		t.Errorf("Role = %q, want %q (goal 5: bootstrap must grant Administrator)", user.Role, auth.RoleAdministrator)
	}
}

func TestAdministratorCreateHashesThePassword(t *testing.T) {
	repo := &fakeUserRepository{}
	admin := bootstrap.NewAdministrator(repo)

	user, err := admin.Create(context.Background(), "admin@example.com", "correct horse battery staple")
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if user.PasswordHash == "correct horse battery staple" {
		t.Fatal("Create() stored the plaintext password instead of a hash")
	}
	if !auth.VerifyPassword(user.PasswordHash, "correct horse battery staple") {
		t.Error("the stored hash does not verify against the password that was provided")
	}
}

func TestAdministratorCreateRefusesWhenUserAlreadyExists(t *testing.T) {
	repo := &fakeUserRepository{users: []auth.User{
		{ID: uuid.New(), Email: "existing@example.com", PasswordHash: "$2a$10$examplehashexamplehashexampleu"},
	}}
	admin := bootstrap.NewAdministrator(repo)

	_, err := admin.Create(context.Background(), "new-admin@example.com", "some password")

	if err == nil {
		t.Fatal("Create() = nil, want a conflict error since a user already exists")
	}
	if !apperror.Is(err, apperror.KindConflict) {
		t.Errorf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindConflict)
	}
	if len(repo.users) != 1 {
		t.Errorf("len(repo.users) = %d, want 1 (no new user should have been created)", len(repo.users))
	}
}

func TestAdministratorCreateRejectsEmptyPassword(t *testing.T) {
	repo := &fakeUserRepository{}
	admin := bootstrap.NewAdministrator(repo)

	_, err := admin.Create(context.Background(), "admin@example.com", "")

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if len(repo.users) != 0 {
		t.Error("Create() persisted a user despite an empty password")
	}
}

func TestAdministratorCreateRejectsMalformedEmail(t *testing.T) {
	repo := &fakeUserRepository{}
	admin := bootstrap.NewAdministrator(repo)

	_, err := admin.Create(context.Background(), "not-an-email", "some password")

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
	if len(repo.users) != 0 {
		t.Error("Create() persisted a user despite a malformed email")
	}
}
