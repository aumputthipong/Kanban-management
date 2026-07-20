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

// Bootstrap prepares a database for the app.
//
// sqlc is generated from schema.sql (see sqlc.yaml), so schema.sql — not the
// migrations — is the source of truth for the current schema. The migrations
// are the historical, incremental path used to evolve an existing database.
//
// On a FRESH database it applies schema.sql and stamps the migration version to
// the latest, so the historical migrations aren't replayed on top (which would
// conflict, since schema.sql already reflects their end state). On an EXISTING
// database it just applies any pending migrations.
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
	m, err := migrate.New("file://"+sourcePath, normalizeDBURL(dbURL))
	if err != nil {
		return fmt.Errorf("schema: open migrate: %w", err)
	}
	defer m.Close()
	if err := m.Force(v); err != nil {
		return fmt.Errorf("schema: force version %d: %w", v, err)
	}
	return nil
}
