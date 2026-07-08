// Package migrate runs SQL migrations on application startup using golang-migrate.
package migrate

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Run applies all pending up-migrations from the given source path against
// the given Postgres URL. It is idempotent — already-applied migrations are skipped.
//
// sourcePath: filesystem path to the migrations directory (e.g. "database/migrations").
// dbURL:      Postgres connection URL. Will be normalized to the pgx/v5 driver.
func Run(sourcePath, dbURL string) error {
	migrationsURL := "file://" + sourcePath
	driverURL := normalizeDBURL(dbURL)

	m, err := migrate.New(migrationsURL, driverURL)
	if err != nil {
		return fmt.Errorf("migrate: open: %w", err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			slog.Warn("migrate: source close failed", "err", srcErr)
		}
		if dbErr != nil {
			slog.Warn("migrate: db close failed", "err", dbErr)
		}
	}()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate: up: %w", err)
	}

	version, dirty, verr := m.Version()
	if verr != nil && !errors.Is(verr, migrate.ErrNilVersion) {
		slog.Warn("migrate: version probe failed", "err", verr)
	} else {
		slog.Info("migrate: applied", "version", version, "dirty", dirty)
	}
	return nil
}

// normalizeDBURL converts the common "postgres://" / "postgresql://" prefixes
// to the "pgx5://" scheme expected by the golang-migrate pgx/v5 driver.
func normalizeDBURL(dbURL string) string {
	const (
		postgres   = "postgres://"
		postgresql = "postgresql://"
		pgx5       = "pgx5://"
	)
	switch {
	case len(dbURL) >= len(postgres) && dbURL[:len(postgres)] == postgres:
		return pgx5 + dbURL[len(postgres):]
	case len(dbURL) >= len(postgresql) && dbURL[:len(postgresql)] == postgresql:
		return pgx5 + dbURL[len(postgresql):]
	default:
		return dbURL
	}
}
