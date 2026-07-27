package auth_test

import (
	"strings"
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
)

func TestHashPasswordAndVerifyPasswordRoundTrip(t *testing.T) {
	hash, err := auth.HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword() = %v", err)
	}

	if hash == "correct horse battery staple" {
		t.Fatal("HashPassword() returned the plaintext password unchanged")
	}
	if !auth.VerifyPassword(hash, "correct horse battery staple") {
		t.Error("VerifyPassword() = false, want true for the matching password")
	}
}

func TestVerifyPasswordRejectsWrongPassword(t *testing.T) {
	hash, err := auth.HashPassword("correct password")
	if err != nil {
		t.Fatalf("HashPassword() = %v", err)
	}

	if auth.VerifyPassword(hash, "wrong password") {
		t.Error("VerifyPassword() = true, want false for a non-matching password")
	}
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	if auth.VerifyPassword("not-a-bcrypt-hash", "anything") {
		t.Error("VerifyPassword() = true, want false for a malformed hash")
	}
}

func TestHashPasswordProducesDifferentHashesForSameInput(t *testing.T) {
	// bcrypt salts every hash, so hashing the same password twice must
	// never produce the same stored value — otherwise two users with the
	// same password would be trivially detectable from the database alone.
	first, err := auth.HashPassword("same password")
	if err != nil {
		t.Fatalf("HashPassword() = %v", err)
	}
	second, err := auth.HashPassword("same password")
	if err != nil {
		t.Fatalf("HashPassword() = %v", err)
	}

	if first == second {
		t.Error("two hashes of the same password were identical; bcrypt should salt each one differently")
	}
	if !auth.VerifyPassword(first, "same password") || !auth.VerifyPassword(second, "same password") {
		t.Error("both differently-salted hashes should still verify the same password")
	}
}

func TestHashPasswordRejectsPasswordOverBcryptLimit(t *testing.T) {
	// bcrypt cannot hash inputs longer than 72 bytes; HashPassword should
	// surface that as an error rather than silently truncating.
	tooLong := strings.Repeat("a", 73)

	if _, err := auth.HashPassword(tooLong); err == nil {
		t.Error("HashPassword() = nil error, want an error for a 73-byte password")
	}
}
