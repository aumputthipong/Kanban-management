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

// filepath.ToSlash must stay: with backslashes url.Parse reads the drive letter as host:port.
func fileSourceURL(sourcePath string) string {
	return "file://" + filepath.ToSlash(sourcePath)
}

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
