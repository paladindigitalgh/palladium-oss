package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/auth/service"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// fakeUserRepository (defined in user_service_test.go) is reused here:
// ProfileService depends on the exact same auth.UserRepository seam
// UserManagementService does, just a different subset of its methods.
func TestProfileServiceUpdateName(t *testing.T) {
	existing := auth.User{ID: uuid.New(), Email: "jane@example.com", PasswordHash: "$2a$10$examplehash", Role: auth.RoleOperator, Status: auth.UserStatusActive}
	repo := newFakeUserRepository(existing)
	svc := service.NewProfileService(repo)

	updated, err := svc.UpdateName(context.Background(), existing.ID, "Jane", "Doe")
	if err != nil {
		t.Fatalf("UpdateName() = %v", err)
	}
	if updated.FirstName != "Jane" || updated.LastName != "Doe" {
		t.Errorf("name = %q %q, want Jane Doe", updated.FirstName, updated.LastName)
	}
}

func TestProfileServiceGetReturnsCaller(t *testing.T) {
	existing := auth.User{ID: uuid.New(), Email: "jane@example.com", PasswordHash: "$2a$10$examplehash", Role: auth.RoleOperator, Status: auth.UserStatusActive}
	repo := newFakeUserRepository(existing)
	svc := service.NewProfileService(repo)

	got, err := svc.Get(context.Background(), existing.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.ID != existing.ID {
		t.Errorf("ID = %v, want %v", got.ID, existing.ID)
	}
}

func TestProfileServiceChangePasswordRequiresCorrectCurrentPassword(t *testing.T) {
	hash, err := auth.HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword() = %v", err)
	}
	existing := auth.User{ID: uuid.New(), Email: "jane@example.com", PasswordHash: hash, Role: auth.RoleOperator, Status: auth.UserStatusActive}
	repo := newFakeUserRepository(existing)
	svc := service.NewProfileService(repo)

	_, err = svc.ChangePassword(context.Background(), existing.ID, "wrong password", "a new password")

	if !apperror.Is(err, apperror.KindInvalid) {
		t.Fatalf("Kind = %q, want %q (err: %v)", apperror.KindOf(err), apperror.KindInvalid, err)
	}
}

func TestProfileServiceChangePasswordReplacesHash(t *testing.T) {
	hash, err := auth.HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword() = %v", err)
	}
	existing := auth.User{ID: uuid.New(), Email: "jane@example.com", PasswordHash: hash, Role: auth.RoleOperator, Status: auth.UserStatusActive}
	repo := newFakeUserRepository(existing)
	svc := service.NewProfileService(repo)

	updated, err := svc.ChangePassword(context.Background(), existing.ID, "correct horse battery staple", "a new password")
	if err != nil {
		t.Fatalf("ChangePassword() = %v", err)
	}

	if !auth.VerifyPassword(updated.PasswordHash, "a new password") {
		t.Error("the stored hash does not verify against the new password")
	}
	if auth.VerifyPassword(updated.PasswordHash, "correct horse battery staple") {
		t.Error("the old password still verifies against the stored hash")
	}
}
