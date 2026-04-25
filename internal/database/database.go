package database

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	migrateMaxAttempts  = 30
	migrateRetryBackoff = 2 * time.Second
)

func Connect(ctx context.Context, databaseUrl string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseUrl)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	return pool, nil
}

func Migrate(databaseUrl string, migrationsFS fs.FS) error {
	src, err := iofs.New(migrationsFS, ".")
	if err != nil {
		return fmt.Errorf("creating iofs source: %w", err)
	}

	var m *migrate.Migrate
	for attempt := 1; attempt <= migrateMaxAttempts; attempt++ {
		m, err = migrate.NewWithSourceInstance("iofs", src, databaseUrl)
		if err == nil {
			break
		}
		if attempt == migrateMaxAttempts {
			return fmt.Errorf("creating migrator after %d attempts: %w", attempt, err)
		}
		slog.Warn("database not ready, retrying",
			"attempt", attempt,
			"max", migrateMaxAttempts,
			"error", err)
		time.Sleep(migrateRetryBackoff)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("running migrations: %w", err)
	}
	return nil
}
