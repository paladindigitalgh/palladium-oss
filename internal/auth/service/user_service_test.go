package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/auth/service"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeUserRepository is an in-memory auth.UserRepository. Like
// internal/provider/service/provider_service_test.go's
// fakeProviderRepository, it exists so UserManagementService's business
// logic — the last-active-administrator guard, hashing on Create — is
// tested without a real database; internal/auth/postgres/user_test.go
// covers the repository itself against real PostgreSQL.
type fakeUserRepository struct {
	byID map[uuid.UUID]auth.User
}

func newFakeUserRepository(users ...auth.User) *fakeUserRepository {
	f := &fakeUserRepository{byID: make(map[uuid.UUID]auth.User)}
	for _, u := range users {
		f.byID[u.ID] = u
	}
	return f
}

func (f *fakeUserRepository) GetByID(_ context.Context, id uuid.UUID) (auth.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return auth.User{}, apperror.NotFound("user not found")
	}
	return u, nil
}

func (f *fakeUserRepository) GetByEmail(_ context.Context, email string) (auth.User, error) {
	for _, u := range f.byID {
		if u.Email == email {
			return u, nil
		}
	}
	return auth.User{}, apperror.NotFound("user not found")
}

func (f *fakeUserRepository) List(_ context.Context) ([]auth.User, error) {
	users := make([]auth.User, 0, len(f.byID))
	for _, u := range f.byID {
		users = append(users, u)
	}
	return users, nil
}

func (f *fakeUserRepository) Create(_ context.Context, u auth.User) (auth.User, error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	f.byID[u.ID] = u
	return u, nil
}

func (f *fakeUserRepository) UpdatePasswordHash(_ context.Context, id uuid.UUID, passwordHash string) (auth.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return auth.User{}, apperror.NotFound("user not found")
	}
	u.PasswordHash = passwordHash
	f.byID[id] = u
	return u, nil
}

func (f *fakeUserRepository) UpdateRole(_ context.Context, id uuid.UUID, role auth.Role) (auth.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return auth.User{}, apperror.NotFound("user not found")
	}
	u.Role = role
	f.byID[id] = u
	return u, nil
}

func (f *fakeUserRepository) UpdateStatus(_ context.Context, id uuid.UUID, status auth.UserStatus) (auth.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return auth.User{}, apperror.NotFound("user not found")
	}
	u.Status = status
	f.byID[id] = u
	return u, nil
}

func (f *fakeUserRepository) Count(_ context.Context) (int, error) {
	return len(f.byID), nil
}

var _ auth.UserRepository = (*fakeUserRepository)(nil)

func TestUserManagementServiceCreateHashesPasswordAndForcesActiveStatus(t *testing.T) {
	repo := newFakeUserRepository()
	svc := service.NewUserManagementService(repo)

	created, err := svc.Create(context.Background(), "jane@example.com", "correct horse battery staple", auth.RoleOperator)
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.PasswordHash == "correct horse battery staple" {
		t.Fatal("Create() stored the plaintext password instead of a hash")
	}
	if !auth.VerifyPassword(created.PasswordHash, "correct horse battery staple") {
		t.Error("the stored hash does not verify against the password that was provided")
	}
	if created.Status != auth.UserStatusActive {
		t.Errorf("Status = %q, want %q — a new account is never caller-configurable to start deactivated", created.Status, auth.UserStatusActive)
	}
	if created.Role != auth.RoleOperator {
		t.Errorf("Role = %q, want %q", created.Role, auth.RoleOperator)
	}
}

func TestUserManagementServiceCreateRejectsInvalidRole(t *testing.T) {
	repo := newFakeUserRepository()
	svc := service.NewUserManagementService(repo)

	_, err := svc.Create(context.Background(), "jane@example.com", "some password", auth.Role("SuperAdmin"))

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindInvalid)
	}
}

func activeAdmin(email string) auth.User {
	return auth.User{ID: uuid.New(), Email: email, PasswordHash: "$2a$10$examplehash", Role: auth.RoleAdministrator, Status: auth.UserStatusActive}
}

// TestUserManagementServiceUpdateRoleRejectedWhenLastActiveAdministrator
// covers both self-demotion and demoting someone else: the guard is the
// same invariant check regardless of who initiated it (see
// guardLastActiveAdministrator's doc comment).
func TestUserManagementServiceUpdateRoleRejectedWhenLastActiveAdministrator(t *testing.T) {
	admin := activeAdmin("admin@example.com")
	repo := newFakeUserRepository(admin)
	svc := service.NewUserManagementService(repo)

	_, err := svc.UpdateRole(context.Background(), admin.ID, auth.RoleOperator)

	if !apperror.Is(err, apperror.KindConflict) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindConflict)
	}

	// The change must not have been applied.
	unchanged, _ := repo.GetByID(context.Background(), admin.ID)
	if unchanged.Role != auth.RoleAdministrator {
		t.Errorf("Role = %q, want unchanged %q after a rejected update", unchanged.Role, auth.RoleAdministrator)
	}
}

func TestUserManagementServiceUpdateRoleAllowedWhenAnotherActiveAdministratorRemains(t *testing.T) {
	admin1 := activeAdmin("admin1@example.com")
	admin2 := activeAdmin("admin2@example.com")
	repo := newFakeUserRepository(admin1, admin2)
	svc := service.NewUserManagementService(repo)

	updated, err := svc.UpdateRole(context.Background(), admin1.ID, auth.RoleOperator)
	if err != nil {
		t.Fatalf("UpdateRole() = %v", err)
	}
	if updated.Role != auth.RoleOperator {
		t.Errorf("Role = %q, want %q", updated.Role, auth.RoleOperator)
	}
}

// TestUserManagementServiceUpdateRoleToAdministratorNeverGuarded proves
// the guard only fires when a change would remove an active
// Administrator, never when it adds one.
func TestUserManagementServiceUpdateRoleToAdministratorNeverGuarded(t *testing.T) {
	viewer := auth.User{ID: uuid.New(), Email: "viewer@example.com", Role: auth.RoleViewer, Status: auth.UserStatusActive}
	repo := newFakeUserRepository(viewer)
	svc := service.NewUserManagementService(repo)

	updated, err := svc.UpdateRole(context.Background(), viewer.ID, auth.RoleAdministrator)
	if err != nil {
		t.Fatalf("UpdateRole() = %v", err)
	}
	if updated.Role != auth.RoleAdministrator {
		t.Errorf("Role = %q, want %q", updated.Role, auth.RoleAdministrator)
	}
}

func TestUserManagementServiceDeactivateRejectedWhenLastActiveAdministrator(t *testing.T) {
	admin := activeAdmin("admin@example.com")
	repo := newFakeUserRepository(admin)
	svc := service.NewUserManagementService(repo)

	_, err := svc.Deactivate(context.Background(), admin.ID)

	if !apperror.Is(err, apperror.KindConflict) {
		t.Fatalf("Kind = %q, want %q", apperror.KindOf(err), apperror.KindConflict)
	}
}

func TestUserManagementServiceDeactivateAllowedWhenAnotherActiveAdministratorRemains(t *testing.T) {
	admin1 := activeAdmin("admin1@example.com")
	admin2 := activeAdmin("admin2@example.com")
	repo := newFakeUserRepository(admin1, admin2)
	svc := service.NewUserManagementService(repo)

	updated, err := svc.Deactivate(context.Background(), admin1.ID)
	if err != nil {
		t.Fatalf("Deactivate() = %v", err)
	}
	if updated.Status != auth.UserStatusInactive {
		t.Errorf("Status = %q, want %q", updated.Status, auth.UserStatusInactive)
	}
}

// TestUserManagementServiceDeactivateNeverGuardsNonAdministrator proves
// the guard never blocks deactivating an Operator or Viewer, even when
// they are the only account of that role — the invariant only protects
// the count of active Administrators.
func TestUserManagementServiceDeactivateNeverGuardsNonAdministrator(t *testing.T) {
	operator := auth.User{ID: uuid.New(), Email: "operator@example.com", Role: auth.RoleOperator, Status: auth.UserStatusActive}
	repo := newFakeUserRepository(operator)
	svc := service.NewUserManagementService(repo)

	updated, err := svc.Deactivate(context.Background(), operator.ID)
	if err != nil {
		t.Fatalf("Deactivate() = %v", err)
	}
	if updated.Status != auth.UserStatusInactive {
		t.Errorf("Status = %q, want %q", updated.Status, auth.UserStatusInactive)
	}
}

// TestUserManagementServiceReactivateNeverGuarded proves Reactivate never
// consults the last-active-administrator invariant: reactivating a User
// can only increase the number of active Administrators, never reduce
// it, so there is nothing to guard against.
func TestUserManagementServiceReactivateNeverGuarded(t *testing.T) {
	admin := activeAdmin("admin@example.com")
	admin.Status = auth.UserStatusInactive
	repo := newFakeUserRepository(admin)
	svc := service.NewUserManagementService(repo)

	updated, err := svc.Reactivate(context.Background(), admin.ID)
	if err != nil {
		t.Fatalf("Reactivate() = %v", err)
	}
	if updated.Status != auth.UserStatusActive {
		t.Errorf("Status = %q, want %q", updated.Status, auth.UserStatusActive)
	}
}
