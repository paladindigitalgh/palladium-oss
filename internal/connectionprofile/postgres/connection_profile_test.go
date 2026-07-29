//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/authentication"
	authenticationpostgres "github.com/paladindigitalgh/palladium-oss/internal/authentication/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/connectionprofile"
	"github.com/paladindigitalgh/palladium-oss/internal/connectionprofile/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/encryption"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// testKey is a fixed, valid 32-byte AES-256 key used only by this test
// file's Authentication fixture — never derived from
// PALLADIUM_MASTER_KEY, and never used to protect anything real.
var testKey = []byte("01234567890123456789012345678901")[:32]

// newTestQuerier opens a transaction against the real test database,
// rolled back automatically on cleanup — the same pattern as
// internal/olt/postgres/olt_test.go. A ConnectionProfile's
// AuthenticationID is optional, so most tests here need no fixture at
// all; the tests that do exercise it use createTestAuthentication.
func newTestQuerier(t *testing.T) (database.Querier, context.Context) {
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

	return tx, ctx
}

// createTestAuthentication creates a real Authentication row through
// internal/authentication/postgres — not
// internal/connectionprofile/postgres — so a ConnectionProfile fixture
// failure surfaces as a clear failure of Authentication's own Create,
// not a confusing failure somewhere else. This is the one place this
// package imports internal/authentication at all: the domain model
// (internal/connectionprofile) never does (see its package doc
// comment), only this test, which genuinely needs a real
// authentication_methods row for the foreign key to reference.
func createTestAuthentication(t *testing.T, ctx context.Context, q database.Querier) authentication.Authentication {
	t.Helper()

	enc, err := encryption.NewAESGCMEncryptor(testKey)
	if err != nil {
		t.Fatalf("fixture: build encryptor: %v", err)
	}
	repo := authenticationpostgres.NewAuthenticationRepository(q, clock.New(), id.New(), enc)
	a, err := repo.Create(ctx, authentication.Authentication{
		Name:               "Fixture Authentication " + uuid.NewString(),
		AuthenticationType: authentication.AuthenticationTypePassword,
		Username:           "admin",
		Password:           "hunter2",
	})
	if err != nil {
		t.Fatalf("fixture: create authentication: %v", err)
	}
	return a
}

func testConnectionProfile(name string) connectionprofile.ConnectionProfile {
	return connectionprofile.ConnectionProfile{
		Name:          name,
		HostKeyPolicy: connectionprofile.HostKeyPolicyStrict,
	}
}

func TestConnectionProfileRepositoryCreate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	authID := createTestAuthentication(t, ctx, q).ID
	repo := postgres.NewConnectionProfileRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, connectionprofile.ConnectionProfile{
		Name:             "Standard SSH",
		Protocol:         "SSH",
		Port:             22,
		AuthenticationID: &authID,
		Timeout:          30 * time.Second,
		HostKeyPolicy:    connectionprofile.HostKeyPolicyStrict,
		Description:      "Standard SSH profile for lab OLTs",
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.ID == uuid.Nil {
		t.Error("Create() did not assign an ID")
	}
	if created.Name != "Standard SSH" {
		t.Errorf("Name = %q, want %q", created.Name, "Standard SSH")
	}
	if created.Protocol != "SSH" {
		t.Errorf("Protocol = %q, want %q", created.Protocol, "SSH")
	}
	if created.Port != 22 {
		t.Errorf("Port = %d, want 22", created.Port)
	}
	if created.AuthenticationID == nil || *created.AuthenticationID != authID {
		t.Errorf("AuthenticationID = %v, want %v", created.AuthenticationID, authID)
	}
	if created.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", created.Timeout)
	}
	if created.HostKeyPolicy != connectionprofile.HostKeyPolicyStrict {
		t.Errorf("HostKeyPolicy = %q, want %q", created.HostKeyPolicy, connectionprofile.HostKeyPolicyStrict)
	}
	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
	if !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Errorf("CreatedAt (%v) != UpdatedAt (%v) on a newly created row", created.CreatedAt, created.UpdatedAt)
	}
}

// TestConnectionProfileRepositoryCreateWithoutAuthenticationID proves a
// ConnectionProfile can exist as a template, with no AuthenticationID
// bound — this milestone's Rules section does not require it (see
// connectionprofile.ConnectionProfile's own doc comment).
func TestConnectionProfileRepositoryCreateWithoutAuthenticationID(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewConnectionProfileRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testConnectionProfile("Template Only"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if created.AuthenticationID != nil {
		t.Errorf("AuthenticationID = %v, want nil", created.AuthenticationID)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.AuthenticationID != nil {
		t.Errorf("Get() AuthenticationID = %v, want nil", got.AuthenticationID)
	}
}

func TestConnectionProfileRepositoryCreateIgnoresCallerSuppliedIdentity(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewConnectionProfileRepository(q, clock.New(), id.New())

	bogusID := uuid.New()
	bogusTime := time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	p := testConnectionProfile("Edge Profile")
	p.ID = bogusID
	p.CreatedAt = bogusTime
	p.UpdatedAt = bogusTime

	created, err := repo.Create(ctx, p)
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

func TestConnectionProfileRepositoryCreateFailsWhenAuthenticationDoesNotExist(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewConnectionProfileRepository(q, clock.New(), id.New())

	nonexistent := uuid.New()
	p := testConnectionProfile("Orphan Profile")
	p.AuthenticationID = &nonexistent

	_, err := repo.Create(ctx, p)

	assertConflict(t, err)
}

func TestConnectionProfileRepositoryGet(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewConnectionProfileRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testConnectionProfile("Get Target"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.ID != created.ID || got.Name != created.Name {
		t.Errorf("Get() = %+v, want %+v", got, created)
	}
}

func TestConnectionProfileRepositoryGetNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewConnectionProfileRepository(q, clock.New(), id.New())

	_, err := repo.Get(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestConnectionProfileRepositoryList(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewConnectionProfileRepository(q, clock.New(), id.New())

	first, err := repo.Create(ctx, testConnectionProfile("Alpha Profile"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	second, err := repo.Create(ctx, testConnectionProfile("Beta Profile"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	profiles, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() = %v", err)
	}

	found := make(map[uuid.UUID]connectionprofile.ConnectionProfile, len(profiles))
	for _, p := range profiles {
		found[p.ID] = p
	}
	if _, ok := found[first.ID]; !ok {
		t.Error("List() did not include the first created profile")
	}
	if _, ok := found[second.ID]; !ok {
		t.Error("List() did not include the second created profile")
	}
	if len(profiles) != 2 {
		t.Fatalf("len(List()) = %d, want 2; got %+v", len(profiles), profiles)
	}
	if profiles[0].Name != "Alpha Profile" || profiles[1].Name != "Beta Profile" {
		t.Errorf("List() order = [%q, %q], want [Alpha Profile, Beta Profile]", profiles[0].Name, profiles[1].Name)
	}
}

func TestConnectionProfileRepositoryUpdate(t *testing.T) {
	q, ctx := newTestQuerier(t)
	authID := createTestAuthentication(t, ctx, q).ID
	repo := postgres.NewConnectionProfileRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testConnectionProfile("Old Name"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	updated, err := repo.Update(ctx, connectionprofile.ConnectionProfile{
		ID:               created.ID,
		Name:             "New Name",
		Protocol:         "SSH",
		Port:             2222,
		AuthenticationID: &authID,
		Timeout:          10 * time.Second,
		HostKeyPolicy:    connectionprofile.HostKeyPolicyInsecure,
		Description:      "New Description",
	})
	if err != nil {
		t.Fatalf("Update() = %v", err)
	}

	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.Port != 2222 {
		t.Errorf("Port = %d, want 2222", updated.Port)
	}
	if updated.AuthenticationID == nil || *updated.AuthenticationID != authID {
		t.Errorf("AuthenticationID = %v, want %v (AuthenticationID must be mutable via Update)", updated.AuthenticationID, authID)
	}
	if updated.Timeout != 10*time.Second {
		t.Errorf("Timeout = %v, want 10s", updated.Timeout)
	}
	if updated.HostKeyPolicy != connectionprofile.HostKeyPolicyInsecure {
		t.Errorf("HostKeyPolicy = %q, want %q", updated.HostKeyPolicy, connectionprofile.HostKeyPolicyInsecure)
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt changed on Update(): was %v, now %v", created.CreatedAt, updated.CreatedAt)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Errorf("UpdatedAt (%v) did not advance past the original (%v)", updated.UpdatedAt, created.UpdatedAt)
	}
}

func TestConnectionProfileRepositoryUpdateNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewConnectionProfileRepository(q, clock.New(), id.New())

	ghost := testConnectionProfile("Ghost")
	ghost.ID = uuid.New()

	_, err := repo.Update(ctx, ghost)

	assertNotFound(t, err)
}

func TestConnectionProfileRepositoryDelete(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewConnectionProfileRepository(q, clock.New(), id.New())

	created, err := repo.Create(ctx, testConnectionProfile("Temporary"))
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() = %v", err)
	}

	_, err = repo.Get(ctx, created.ID)
	assertNotFound(t, err)
}

func TestConnectionProfileRepositoryDeleteNotFound(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewConnectionProfileRepository(q, clock.New(), id.New())

	err := repo.Delete(ctx, uuid.New())

	assertNotFound(t, err)
}

func TestConnectionProfileRepositoryCreateConflictOnDuplicateID(t *testing.T) {
	fixedID := uuid.New()
	q, ctx := newTestQuerier(t)
	repo := postgres.NewConnectionProfileRepository(q, clock.New(), id.Static{Value: fixedID})

	if _, err := repo.Create(ctx, testConnectionProfile("First")); err != nil {
		t.Fatalf("first Create() = %v", err)
	}

	_, err := repo.Create(ctx, testConnectionProfile("Second"))
	assertConflict(t, err)
}

// TestConnectionProfileRepositoryCreateConflictOnDuplicateName proves
// this milestone's explicit "Name unique" rule is enforced at the
// database level.
func TestConnectionProfileRepositoryCreateConflictOnDuplicateName(t *testing.T) {
	q, ctx := newTestQuerier(t)
	repo := postgres.NewConnectionProfileRepository(q, clock.New(), id.New())

	if _, err := repo.Create(ctx, testConnectionProfile("Shared Name")); err != nil {
		t.Fatalf("first Create() = %v", err)
	}

	_, err := repo.Create(ctx, testConnectionProfile("Shared Name"))
	assertConflict(t, err)
}

// TestAuthenticationRepositoryDeleteBlockedByExistingConnectionProfile
// lives here, not in internal/authentication/postgres, so that
// package's existing test files stay untouched — the same reasoning
// internal/olt/postgres/olt_test.go already documents for why its
// equivalent test lives with the child, not the parent. It exercises
// AuthenticationRepository.Delete against the foreign key this
// migration adds.
func TestAuthenticationRepositoryDeleteBlockedByExistingConnectionProfile(t *testing.T) {
	q, ctx := newTestQuerier(t)
	authID := createTestAuthentication(t, ctx, q).ID
	repo := postgres.NewConnectionProfileRepository(q, clock.New(), id.New())

	p := testConnectionProfile("Blocking Profile")
	p.AuthenticationID = &authID
	if _, err := repo.Create(ctx, p); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	enc, err := encryption.NewAESGCMEncryptor(testKey)
	if err != nil {
		t.Fatalf("test setup: build encryptor: %v", err)
	}
	authRepo := authenticationpostgres.NewAuthenticationRepository(q, clock.New(), id.New(), enc)

	err = authRepo.Delete(ctx, authID)

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
