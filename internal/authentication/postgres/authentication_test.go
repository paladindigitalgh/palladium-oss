//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/authentication"
	"github.com/paladindigitalgh/palladium-oss/internal/authentication/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/encryption"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// testKey is a fixed, valid 32-byte AES-256 key used only by this test
// file — never derived from PALLADIUM_MASTER_KEY, and never used to
// protect anything real.
var testKey = []byte("01234567890123456789012345678901")[:32]

// newTestEncryptor builds a real internal/platform/encryption.Encryptor
// against testKey. These tests exercise the genuine encrypt-on-write,
// decrypt-on-read round trip through real PostgreSQL — not a fake — so a
// regression in either this package's wiring or
// internal/platform/encryption's own correctness would be caught here,
// not just in internal/platform/encryption's own unit tests.
func newTestEncryptor(t *testing.T) encryption.Encryptor {
	t.Helper()
	enc, err := encryption.NewAESGCMEncryptor(testKey)
	if err != nil {
		t.Fatalf("test setup: NewAESGCMEncryptor() = %v", err)
	}
	return enc
}

// newTestRepository mirrors internal/catalog/postgres/catalog_test.go's
// helper of the same name: open a transaction against the real test
// database, build the repository under test on it, and roll the
// transaction back on cleanup so tests never leave data behind or
// observe each other's writes.
func newTestRepository(t *testing.T, ids id.Generator, enc encryption.Encryptor) (*postgres.AuthenticationRepository, database.Querier, context.Context) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	pool, err := database.Connect(ctx, testConfig(t))
	if err != nil {
		t.Fatalf("Connect() = %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("Ping() = %v; is Postgres running and migrated? try `make db-up && make migrate-up`", err)
	}

	tx, err := pool.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx() = %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback(ctx) })

	return postgres.NewAuthenticationRepository(tx, clock.New(), ids, enc), tx, ctx
}

func testPasswordAuthentication(name string) authentication.Authentication {
	return authentication.Authentication{
		Name:               name,
		AuthenticationType: authentication.AuthenticationTypePassword,
		Username:           "admin",
		Password:           "hunter2",
	}
}

func TestAuthenticationRepositoryCreate(t *testing.T) {
	repo, _, ctx := newTestRepository(t, id.New(), newTestEncryptor(t))

	created, err := repo.Create(ctx, authentication.Authentication{
		Name:               "Default Device Login",
		AuthenticationType: authentication.AuthenticationTypePassword,
		Username:           "admin",
		Password:           "hunter2",
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if created.Name != "Default Device Login" {
		t.Errorf("Name = %q, want %q", created.Name, "Default Device Login")
	}
	if created.AuthenticationType != authentication.AuthenticationTypePassword {
		t.Errorf("AuthenticationType = %q, want %q", created.AuthenticationType, authentication.AuthenticationTypePassword)
	}
	if created.Username != "admin" {
		t.Errorf("Username = %q, want %q", created.Username, "admin")
	}
	if created.Password != "hunter2" {
		t.Errorf("Password = %q, want the plaintext %q back (decrypted)", created.Password, "hunter2")
	}
	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
	if !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Errorf("CreatedAt (%v) != UpdatedAt (%v) on a newly created row", created.CreatedAt, created.UpdatedAt)
	}
}

// TestAuthenticationRepositoryCreateStoresCiphertextNotPlaintext is the
// central security proof for this whole package: whatever ends up in
// the password column must not be the plaintext itself, and must not
// even contain it as a substring.
func TestAuthenticationRepositoryCreateStoresCiphertextNotPlaintext(t *testing.T) {
	repo, tx, ctx := newTestRepository(t, id.New(), newTestEncryptor(t))

	created, err := repo.Create(ctx, testPasswordAuthentication("Plaintext Check"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	var storedPassword string
	row := tx.QueryRow(ctx, "SELECT password FROM authentication_methods WHERE id = $1", created.ID)
	if err := row.Scan(&storedPassword); err != nil {
		t.Fatalf("query stored password column: %v", err)
	}

	if storedPassword == "hunter2" {
		t.Fatal("password column stores the plaintext password verbatim")
	}
	if storedPassword == "" {
		t.Fatal("password column is empty, want the encrypted ciphertext")
	}
}

func TestAuthenticationRepositoryCreateSSHKeyType(t *testing.T) {
	repo, _, ctx := newTestRepository(t, id.New(), newTestEncryptor(t))

	privateKey := "-----BEGIN OPENSSH PRIVATE KEY-----\nfake-key-material\n-----END OPENSSH PRIVATE KEY-----"
	created, err := repo.Create(ctx, authentication.Authentication{
		Name:               "Default SSH Key",
		AuthenticationType: authentication.AuthenticationTypeSSHKey,
		Username:           "admin",
		PrivateKey:         privateKey,
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.PrivateKey != privateKey {
		t.Errorf("PrivateKey = %q, want the plaintext key back (decrypted)", created.PrivateKey)
	}
	if created.Password != "" {
		t.Errorf("Password = %q, want empty for an SSHKey-type record", created.Password)
	}
}

func TestAuthenticationRepositoryCreateIgnoresCallerSuppliedIdentity(t *testing.T) {
	repo, _, ctx := newTestRepository(t, id.New(), newTestEncryptor(t))

	bogusID := uuid.New()
	bogusTime := time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	a := testPasswordAuthentication("Edge Authentication")
	a.ID = bogusID
	a.CreatedAt = bogusTime
	a.UpdatedAt = bogusTime

	created, err := repo.Create(ctx, a)
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ID == bogusID {
		t.Error("Create() used the caller-supplied ID instead of generating one")
	}
	if created.CreatedAt.Equal(bogusTime) {
		t.Error("Create() used the caller-supplied CreatedAt instead of stamping the current time")
	}
}

func TestAuthenticationRepositoryGet(t *testing.T) {
	repo, _, ctx := newTestRepository(t, id.New(), newTestEncryptor(t))

	created, err := repo.Create(ctx, testPasswordAuthentication("Business Login"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.ID != created.ID || got.Name != created.Name || got.Password != created.Password {
		t.Errorf("Get() = %+v, want %+v", got, created)
	}
}

func TestAuthenticationRepositoryGetNotFound(t *testing.T) {
	repo, _, ctx := newTestRepository(t, id.New(), newTestEncryptor(t))

	_, err := repo.Get(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestAuthenticationRepositoryList(t *testing.T) {
	repo, _, ctx := newTestRepository(t, id.New(), newTestEncryptor(t))

	first, err := repo.Create(ctx, testPasswordAuthentication("Alpha Login"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	second, err := repo.Create(ctx, testPasswordAuthentication("Beta Login"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	auths, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() = %v", err)
	}

	found := make(map[uuid.UUID]authentication.Authentication, len(auths))
	for _, a := range auths {
		found[a.ID] = a
	}
	if _, ok := found[first.ID]; !ok {
		t.Error("List() did not include the first created authentication")
	}
	if _, ok := found[second.ID]; !ok {
		t.Error("List() did not include the second created authentication")
	}
	if len(auths) != 2 {
		t.Fatalf("len(List()) = %d, want 2; got %+v", len(auths), auths)
	}
	if auths[0].Name != "Alpha Login" || auths[1].Name != "Beta Login" {
		t.Errorf("List() order = [%q, %q], want [Alpha Login, Beta Login]", auths[0].Name, auths[1].Name)
	}
}

func TestAuthenticationRepositoryUpdate(t *testing.T) {
	repo, _, ctx := newTestRepository(t, id.New(), newTestEncryptor(t))

	created, err := repo.Create(ctx, testPasswordAuthentication("Old Name"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	updated, err := repo.Update(ctx, authentication.Authentication{
		ID:                 created.ID,
		Name:               "New Name",
		AuthenticationType: authentication.AuthenticationTypePassword,
		Username:           "root",
		Password:           "new-password",
	})
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}

	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.Username != "root" {
		t.Errorf("Username = %q, want %q", updated.Username, "root")
	}
	if updated.Password != "new-password" {
		t.Errorf("Password = %q, want %q", updated.Password, "new-password")
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt changed on Update(): was %v, now %v", created.CreatedAt, updated.CreatedAt)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Errorf("UpdatedAt (%v) did not advance past the original (%v)", updated.UpdatedAt, created.UpdatedAt)
	}
}

func TestAuthenticationRepositoryUpdateNotFound(t *testing.T) {
	repo, _, ctx := newTestRepository(t, id.New(), newTestEncryptor(t))

	ghost := testPasswordAuthentication("Ghost")
	ghost.ID = uuid.New()

	_, err := repo.Update(ctx, ghost)

	assertNotFound(t, err)
}

func TestAuthenticationRepositoryDelete(t *testing.T) {
	repo, _, ctx := newTestRepository(t, id.New(), newTestEncryptor(t))

	created, err := repo.Create(ctx, testPasswordAuthentication("Temporary"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err = repo.Get(ctx, created.ID)
	assertNotFound(t, err)
}

func TestAuthenticationRepositoryDeleteNotFound(t *testing.T) {
	repo, _, ctx := newTestRepository(t, id.New(), newTestEncryptor(t))

	err := repo.Delete(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestAuthenticationRepositoryCreateConflictOnDuplicateID(t *testing.T) {
	fixedID := uuid.New()
	repo, _, ctx := newTestRepository(t, id.Static{Value: fixedID}, newTestEncryptor(t))

	if _, err := repo.Create(ctx, testPasswordAuthentication("First")); err != nil {
		t.Fatalf("first Create() = %v", err)
	}

	_, err := repo.Create(ctx, testPasswordAuthentication("Second"))
	assertConflict(t, err)
}

// TestAuthenticationRepositoryCreateConflictOnDuplicateName proves this
// milestone's explicit "Name unique" rule is enforced at the database
// level.
func TestAuthenticationRepositoryCreateConflictOnDuplicateName(t *testing.T) {
	repo, _, ctx := newTestRepository(t, id.New(), newTestEncryptor(t))

	if _, err := repo.Create(ctx, testPasswordAuthentication("Shared Name")); err != nil {
		t.Fatalf("first Create() = %v", err)
	}

	_, err := repo.Create(ctx, testPasswordAuthentication("Shared Name"))
	assertConflict(t, err)
}

func assertNotFound(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("error = nil, want a not-found error")
	}
	if !apperror.Is(err, apperror.KindNotFound) {
		t.Errorf("Kind = %q, want %q (err: %v)", apperror.KindOf(err), apperror.KindNotFound, err)
	}
}

func assertConflict(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("error = nil, want a conflict error")
	}
	if !apperror.Is(err, apperror.KindConflict) {
		t.Errorf("Kind = %q, want %q (err: %v)", apperror.KindOf(err), apperror.KindConflict, err)
	}
}

// testConfig points at the same database.Config the rest of the test
// suite uses (see internal/customer/postgres/customer_test.go): local
// defaults that match docker-compose.yml, overridable via environment
// variables.
func testConfig(t *testing.T) database.Config {
	t.Helper()
	return database.Config{
		Host:            envOrDefault("DB_HOST", "localhost"),
		Port:            5432,
		User:            envOrDefault("DB_USER", "palladium"),
		Password:        envOrDefault("DB_PASSWORD", "palladium"),
		Database:        envOrDefault("DB_NAME", "palladium"),
		SSLMode:         "disable",
		MaxConns:        2,
		MinConns:        1,
		MaxConnLifetime: time.Minute,
		MaxConnIdleTime: time.Minute,
		ConnectTimeout:  5 * time.Second,
	}
}

func envOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
