package auth_test

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
)

var fixedNow = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func TestIssueTokenAndParseTokenRoundTrip(t *testing.T) {
	issuer := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(fixedNow))
	user := auth.User{ID: uuid.New(), Email: "jane@example.com"}

	token, err := issuer.IssueToken(user)
	if err != nil {
		t.Fatalf("IssueToken() = %v", err)
	}
	if token == "" {
		t.Fatal("IssueToken() returned an empty string")
	}

	claims, err := issuer.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() = %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("UserID = %v, want %v", claims.UserID, user.ID)
	}
	if claims.Email != user.Email {
		t.Errorf("Email = %q, want %q", claims.Email, user.Email)
	}
	if !claims.IssuedAt.Time.Equal(fixedNow) {
		t.Errorf("IssuedAt = %v, want %v", claims.IssuedAt.Time, fixedNow)
	}
	wantExpiry := fixedNow.Add(time.Hour)
	if !claims.ExpiresAt.Time.Equal(wantExpiry) {
		t.Errorf("ExpiresAt = %v, want %v", claims.ExpiresAt.Time, wantExpiry)
	}
}

// TestIssueTokenClaimsContainOnlyUserIDEmailIatExp decodes the raw JWT
// payload and checks its key set directly, rather than only checking the
// Claims struct's fields, so it actually proves what goes out over the
// wire — not just what Go's json.Unmarshal happens to populate. This is
// the concrete check behind "do not include roles or permissions."
func TestIssueTokenClaimsContainOnlyUserIDEmailIatExp(t *testing.T) {
	issuer := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(fixedNow))
	user := auth.User{ID: uuid.New(), Email: "jane@example.com"}

	token, err := issuer.IssueToken(user)
	if err != nil {
		t.Fatalf("IssueToken() = %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token has %d dot-separated segments, want 3", len(parts))
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload segment: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	wantKeys := []string{"user_id", "email", "iat", "exp"}
	if len(raw) != len(wantKeys) {
		t.Fatalf("claim keys = %v, want exactly %v", keysOf(raw), wantKeys)
	}
	for _, key := range wantKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("claims missing key %q; got keys %v", key, keysOf(raw))
		}
	}
}

func keysOf(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestParseTokenRejectsExpiredToken(t *testing.T) {
	secret := []byte("test-secret")
	issuedAt := fixedNow
	afterExpiry := fixedNow.Add(2 * time.Hour)

	issuer := auth.NewTokenIssuer(secret, time.Hour, clock.NewFrozen(issuedAt))
	token, err := issuer.IssueToken(auth.User{ID: uuid.New(), Email: "jane@example.com"})
	if err != nil {
		t.Fatalf("IssueToken() = %v", err)
	}

	// A second issuer, same secret, whose clock is frozen after the first
	// token's expiration — this is what makes the expiry check
	// deterministic and sleep-free (see token.go's ParseToken doc comment
	// on jwt.WithTimeFunc).
	laterIssuer := auth.NewTokenIssuer(secret, time.Hour, clock.NewFrozen(afterExpiry))

	_, err = laterIssuer.ParseToken(token)
	assertUnauthorized(t, err)
}

func TestParseTokenRejectsWrongSecret(t *testing.T) {
	issuer := auth.NewTokenIssuer([]byte("secret-a"), time.Hour, clock.NewFrozen(fixedNow))
	token, err := issuer.IssueToken(auth.User{ID: uuid.New(), Email: "jane@example.com"})
	if err != nil {
		t.Fatalf("IssueToken() = %v", err)
	}

	otherIssuer := auth.NewTokenIssuer([]byte("secret-b"), time.Hour, clock.NewFrozen(fixedNow))

	_, err = otherIssuer.ParseToken(token)
	assertUnauthorized(t, err)
}

func TestParseTokenRejectsMalformedToken(t *testing.T) {
	issuer := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.NewFrozen(fixedNow))

	_, err := issuer.ParseToken("this.is.not-a-valid-jwt")

	assertUnauthorized(t, err)
}

func assertUnauthorized(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("error = nil, want an unauthorized error")
	}
	if !apperror.Is(err, apperror.KindUnauthorized) {
		t.Errorf("Kind = %q, want %q (err: %v)", apperror.KindOf(err), apperror.KindUnauthorized, err)
	}
}
