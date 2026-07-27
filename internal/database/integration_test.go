//go:build integration

package database_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/paladindigitalgh/palladium-oss/internal/database"
)

func TestConnectAndHealthCheck(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.Connect(ctx, testConfig(t))
	if err != nil {
		t.Fatalf("Connect() = %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("Ping() = %v; is Postgres running? try `make db-up`", err)
	}

	checker := database.NewHealthChecker(pool)
	if got := checker.Name(); got != "database" {
		t.Errorf("Name() = %q, want %q", got, "database")
	}
	if err := checker.Check(ctx); err != nil {
		t.Errorf("Check() = %v", err)
	}
}

func TestRunInTxCommitsAndRollsBack(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.Connect(ctx, testConfig(t))
	if err != nil {
		t.Fatalf("Connect() = %v", err)
	}
	defer pool.Close()

	// Committed transaction: a value written inside should be visible after.
	err = database.RunInTx(ctx, pool, func(ctx context.Context, q database.Querier) error {
		_, execErr := q.Exec(ctx, "CREATE TEMP TABLE IF NOT EXISTS runintx_probe (n int)")
		return execErr
	})
	if err != nil {
		t.Fatalf("RunInTx() commit case = %v", err)
	}

	// Rolled-back transaction: an error from fn must undo its writes.
	sentinelErr := context.Canceled // any non-nil error stands in for a business failure
	err = database.RunInTx(ctx, pool, func(ctx context.Context, q database.Querier) error {
		if _, execErr := q.Exec(ctx, "INSERT INTO runintx_probe (n) VALUES (1)"); execErr != nil {
			return execErr
		}
		return sentinelErr
	})
	if err != sentinelErr {
		t.Fatalf("RunInTx() rollback case = %v, want %v", err, sentinelErr)
	}
}

// testConfig pins the pool to a single connection so that sequential
// transactions in the same test (e.g. the TEMP TABLE used by
// TestRunInTxCommitsAndRollsBack) observe a consistent session; TEMP TABLE
// visibility is connection-scoped in Postgres.
func testConfig(t *testing.T) database.Config {
	t.Helper()
	return database.Config{
		Host:            envOrDefault("DB_HOST", "localhost"),
		Port:            5432,
		User:            envOrDefault("DB_USER", "palladium"),
		Password:        envOrDefault("DB_PASSWORD", "palladium"),
		Database:        envOrDefault("DB_NAME", "palladium"),
		SSLMode:         "disable",
		MaxConns:        1,
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
