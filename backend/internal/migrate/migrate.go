// Package migrate runs SQL migrations on application startup using golang-migrate.
package migrate

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// fileSourceURL turns an OS filesystem path into a "file://" URL golang-
// migrate's source driver can parse. A naive "file://" + path breaks on
// Windows: a path like `C:\Users\...\migrations` has backslashes url.Parse
// treats as opaque, so the whole path — including the "C:" — gets read as
// the URL authority, and Parse tries (and fails) to read what follows the
// colon as a port number.
//
// filepath.ToSlash is the whole fix. It turns the Windows path into
// `C:/Users/...`; url.Parse then reads "C:" as the host and "/Users/..." as
// the path, and golang-migrate's own parseURL concatenates host+path back
// into the original "C:/Users/..." — which os.DirFS accepts. On Unix
// ToSlash is a no-op (already forward slashes), so this changes nothing
// there: an absolute path already starts with "/", giving the same
// "file:///home/..." this produced before.
func fileSourceURL(sourcePath string) string {
	return "file://" + filepath.ToSlash(sourcePath)
}

// Run applies all pending up-migrations from the given source path against
// the given Postgres URL. It is idempotent — already-applied migrations are skipped.
//
// sourcePath: filesystem path to the migrations directory (e.g. "database/migrations").
// dbURL:      Postgres connection URL. Will be normalized to the pgx/v5 driver.
func Run(sourcePath, dbURL string) error {
	migrationsURL := fileSourceURL(sourcePath)
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
