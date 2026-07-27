//go:build integration

package bootstrap_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/paladindigitalgh/palladium-oss/internal/auth"
	"github.com/paladindigitalgh/palladium-oss/internal/auth/bootstrap"
	"github.com/paladindigitalgh/palladium-oss/internal/auth/postgres"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// newTestRepository mirrors internal/auth/postgres/user_test.go's helper
// of the same name: a transaction rolled back on cleanup, so tests never
// leave data behind or observe each other's writes — important here
// specifically because Administrator.Create's whole behavior hinges on
// how many rows already exist in the table.
func newTestRepository(t *testing.T) (*postgres.UserRepository, context.Context) {
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

	return postgres.NewUserRepository(tx, clock.New(), id.New()), ctx
}

// TestAdministratorCreateAgainstRealDatabaseRefusesWhenUserExists proves
// the refusal check against the real Count() SQL (SELECT count(*) FROM
// users), not just an in-memory fake's len() — the unit test in
// bootstrap_test.go already covers the orchestration logic in isolation,
// this proves the query behind it is actually correct.
func TestAdministratorCreateAgainstRealDatabaseRefusesWhenUserExists(t *testing.T) {
	repo, ctx := newTestRepository(t)

	if _, err := repo.Create(ctx, auth.User{
		Email:        "existing-admin@example.com",
		PasswordHash: "$2a$10$examplehashexamplehashexampleu",
	}); err != nil {
		t.Fatalf("fixture Create() = %v", err)
	}

	admin := bootstrap.NewAdministrator(repo)
	_, err := admin.Create(ctx, "second-admin@example.com", "some password")

	if !apperror.Is(err, apperror.KindConflict) {
		t.Fatalf("Kind = %q, want %q (err: %v)", apperror.KindOf(err), apperror.KindConflict, err)
	}
}

// TestBootstrapCreatesAUsableLogin is the literal, end-to-end version of
// this milestone's "bootstrap creates a usable login" requirement: it
// creates an administrator through bootstrap.Administrator against a real
// UserRepository, then — using nothing but that same repository, exactly
// as cmd/server wires it in production — builds a real auth.AuthService
// and proves the created account can actually log in and receive a valid
// token. It also proves the wrong password still fails against the
// freshly bootstrapped account, so "usable login" means the real
// credential, not merely "some row exists in the table."
func TestBootstrapCreatesAUsableLogin(t *testing.T) {
	repo, ctx := newTestRepository(t)

	admin := bootstrap.NewAdministrator(repo)
	created, err := admin.Create(ctx, "admin@example.com", "correct horse battery staple")
	if err != nil {
		t.Fatalf("Administrator.Create() = %v", err)
	}
	if created.Role != auth.RoleAdministrator {
		t.Fatalf("Role = %q, want %q (goal 5: bootstrap must grant Administrator)", created.Role, auth.RoleAdministrator)
	}

	// Round-trip through the real database, not just the value Create()
	// returned in memory: proves the role column is actually persisted
	// and read back correctly, not merely passed through in the Go value.
	stored, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID() = %v", err)
	}
	if stored.Role != auth.RoleAdministrator {
		t.Fatalf("stored Role = %q, want %q", stored.Role, auth.RoleAdministrator)
	}

	tokens := auth.NewTokenIssuer([]byte("test-secret"), time.Hour, clock.New())
	authService := auth.NewAuthService(repo, tokens)

	token, err := authService.Authenticate(ctx, "admin@example.com", "correct horse battery staple")
	if err != nil {
		t.Fatalf("Authenticate() with the bootstrapped credentials = %v", err)
	}
	if token == "" {
		t.Fatal("Authenticate() returned an empty token")
	}

	claims, err := tokens.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() = %v", err)
	}
	if claims.UserID != created.ID {
		t.Errorf("UserID = %v, want %v", claims.UserID, created.ID)
	}

	if _, err := authService.Authenticate(ctx, "admin@example.com", "wrong password"); err == nil {
		t.Error("Authenticate() with the wrong password succeeded, want an error")
	}
}

// testConfig points at the same database.Config the rest of the test
// suite uses (see internal/auth/postgres/user_test.go): local defaults
// that match docker-compose.yml, overridable via environment variables.
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
