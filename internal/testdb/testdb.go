// Package testdb provides shared test helpers for integration tests that
// need a real Postgres. Tests opt in by calling NewPool(t); if no test DB is
// configured, the test is skipped rather than failing, so plain `go test
// ./...` works on a developer box without docker.
package testdb

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vantutran2k1/SignalFlow/internal/database"
	"github.com/vantutran2k1/SignalFlow/migrations"
)

// allTables is the list of tables truncated between tests. Order matters only
// for foreign-key reasons, but we use TRUNCATE ... CASCADE so order is safe.
var allTables = []string{"notifications", "executions", "jobs", "channels", "users"}

var (
	migrateOnce sync.Once
	migrateErr  error
)

// URL returns the configured test database URL, or empty if integration tests
// should be skipped. Honors TEST_DATABASE_URL first; falls back to
// DATABASE_URL when running in CI (which sets the latter to a disposable DB).
func URL() string {
	if v := os.Getenv("TEST_DATABASE_URL"); v != "" {
		return v
	}
	if v := os.Getenv("DATABASE_URL"); v != "" && strings.Contains(v, "test") {
		// Only borrow DATABASE_URL if it clearly points at a test database;
		// otherwise we'd happily nuke a developer's dev data.
		return v
	}
	return ""
}

// NewPool returns a pgx pool to a freshly-truncated test database. Migrations
// are applied once per process; subsequent calls just truncate. If no test DB
// is configured, the test is skipped.
func NewPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	url := URL()
	if url == "" {
		t.Skip("set TEST_DATABASE_URL (or DATABASE_URL containing 'test') to run integration tests")
	}

	migrateOnce.Do(func() {
		migrateErr = database.Migrate(url, migrations.FS)
	})
	if migrateErr != nil {
		t.Fatalf("migrate: %v", migrateErr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := Truncate(ctx, pool); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return pool
}

// Truncate wipes all tables in dependency-safe order.
func Truncate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx,
		"TRUNCATE "+strings.Join(allTables, ", ")+" RESTART IDENTITY CASCADE")
	return err
}
