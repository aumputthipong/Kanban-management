package migrate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5"
)

// Bootstrap prepares a database for the app. schema.sql is the source of truth (sqlc
// generates from it); migrations are the historical path for an existing database. A
// FRESH database gets schema.sql plus a version stamp so history is not replayed over
// its own end state; an EXISTING one just gets pending migrations.
func Bootstrap(ctx context.Context, dbURL, schemaPath, migrationsPath string) error {
	fresh, err := applyBaseSchema(ctx, dbURL, schemaPath)
	if err != nil {
		return err
	}
	if fresh {
		v, err := latestVersion(migrationsPath)
		if err != nil {
			return err
		}
		return stampVersion(migrationsPath, dbURL, v)
	}
	return Run(migrationsPath, dbURL)
}

// applyBaseSchema applies schema.sql when the database has no base tables yet.
// Returns true when it applied (the database was fresh).
func applyBaseSchema(ctx context.Context, dbURL, schemaPath string) (bool, error) {
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		return false, fmt.Errorf("schema: connect: %w", err)
	}
	defer conn.Close(ctx)

	var existing *string
	if err := conn.QueryRow(ctx, "SELECT to_regclass('public.boards')::text").Scan(&existing); err != nil {
		return false, fmt.Errorf("schema: probe: %w", err)
	}
	if existing != nil {
		return false, nil // base tables already present
	}

	sql, err := os.ReadFile(schemaPath)
	if err != nil {
		return false, fmt.Errorf("schema: read %s: %w", schemaPath, err)
	}
	if _, err := conn.Exec(ctx, string(sql)); err != nil {
		return false, fmt.Errorf("schema: apply: %w", err)
	}
	return true, nil
}

// latestVersion returns the highest NNNNNN migration number in sourcePath.
func latestVersion(sourcePath string) (int, error) {
	entries, err := os.ReadDir(sourcePath)
	if err != nil {
		return 0, fmt.Errorf("schema: read migrations dir: %w", err)
	}
	max := 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		i := strings.IndexByte(name, '_')
		if i <= 0 {
			continue
		}
		if n, err := strconv.Atoi(name[:i]); err == nil && n > max {
			max = n
		}
	}
	if max == 0 {
		return 0, errors.New("schema: no migrations found")
	}
	return max, nil
}

// stampVersion marks the database as being at version v (clean) without running
// any migration — used after applying the schema.sql baseline on a fresh DB.
func stampVersion(sourcePath, dbURL string, v int) error {
	m, err := migrate.New(fileSourceURL(sourcePath), normalizeDBURL(dbURL))
	if err != nil {
		return fmt.Errorf("schema: open migrate: %w", err)
	}
	defer m.Close()
	if err := m.Force(v); err != nil {
		return fmt.Errorf("schema: force version %d: %w", v, err)
	}
	return nil
}
