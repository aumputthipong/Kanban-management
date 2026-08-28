//go:build integration

// Package testutil spins up a real Postgres for integration tests.
//
// Design: one testcontainers Postgres per test binary (sync.Once), one
// "template" database with migrations applied once, and each NewTestDB(t)
// call clones that template via `CREATE DATABASE ... TEMPLATE template`.
// Postgres template-clone is near-instant (~10ms), so per-test isolation
// stays cheap even with the full migration set applied.
//
// Tests opt in via the `integration` build tag — `make test` (the fast
// path mirroring CI) skips this package; `make test-integration` runs it.
package testutil

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/migrate"
)

const templateDB = "turtask_template"

var (
	once        sync.Once
	container   *tcpg.PostgresContainer
	adminDSN    string // connection string to "postgres" admin DB
	templateErr error
)

// NewTestDB returns a *pgxpool.Pool pointing at a fresh database cloned
// from the migrated template. The DB is dropped on test cleanup.
//
// Race tests and concurrent commits work because the returned pool talks
// to a real Postgres instance — no in-memory shortcuts, no shared state
// with other tests.
func NewTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	once.Do(func() { templateErr = bootstrap(ctx) })
	if templateErr != nil {
		t.Fatalf("testutil bootstrap: %v", templateErr)
	}

	dbName := "test_" + sanitize(uuid.NewString())

	admin, err := pgxpool.New(ctx, adminDSN)
	if err != nil {
		t.Fatalf("testutil admin pool: %v", err)
	}
	defer admin.Close()

	// CREATE DATABASE doesn't accept parameters; safe here because dbName
	// is uuid-derived and sanitize() keeps only [a-z0-9_].
	if _, err := admin.Exec(ctx, fmt.Sprintf(
		"CREATE DATABASE %s TEMPLATE %s", dbName, templateDB,
	)); err != nil {
		t.Fatalf("testutil clone template: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsnForDB(dbName))
	if err != nil {
		t.Fatalf("testutil test pool: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		// Reconnect to admin to drop the test DB. Best-effort — a leaked
		// DB just costs memory in the (ephemeral) container.
		dropAdmin, derr := pgxpool.New(context.Background(), adminDSN)
		if derr != nil {
			t.Logf("testutil drop admin: %v", derr)
			return
		}
		defer dropAdmin.Close()
		if _, derr := dropAdmin.Exec(context.Background(),
			fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", dbName),
		); derr != nil {
			t.Logf("testutil drop %s: %v", dbName, derr)
		}
	})

	return pool
}

// bootstrap starts the shared Postgres container and prepares the template
// database. Runs exactly once per test binary.
func bootstrap(ctx context.Context) error {
	c, err := tcpg.Run(ctx,
		"postgres:15-alpine",
		tcpg.WithDatabase("postgres"),
		tcpg.WithUsername("turtask"),
		tcpg.WithPassword("turtask"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return fmt.Errorf("start postgres: %w", err)
	}
	container = c

	base, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return fmt.Errorf("connection string: %w", err)
	}
	// base ends in "/postgres?..."; we use it as admin and derive per-db
	// DSNs by swapping the path segment.
	adminDSN = base

	// Create template DB then run migrations against it.
	adminPool, err := pgxpool.New(ctx, adminDSN)
	if err != nil {
		return fmt.Errorf("admin pool: %w", err)
	}
	defer adminPool.Close()

	if _, err := adminPool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", templateDB)); err != nil {
		return fmt.Errorf("create template: %w", err)
	}

	// Mirror production's bootstrap exactly (internal/migrate.Bootstrap):
	// schema.sql already reflects every migration's end state, so on this
	// fresh template DB it applies schema.sql and stamps the version table
	// to latest — it does NOT also replay the migrations. Calling
	// applySchemaFile followed by migrate.Run here used to do both
	// unconditionally, which re-ran every migration against a DB that
	// schema.sql had already brought to the same end state, and failed on
	// the first CREATE TABLE schema.sql and a migration both contain
	// (e.g. "tags" already exists, added in schema.sql and again in
	// migration 000004).
	if err := migrate.Bootstrap(ctx, dsnForDB(templateDB), schemaFilePath(), migrationsDir()); err != nil {
		return fmt.Errorf("template bootstrap: %w", err)
	}

	// Mark template as a template so CREATE DATABASE ... TEMPLATE is allowed
	// without superuser privileges on the source DB beyond what we already
	// have. ALLOW_CONNECTIONS=false + datistemplate prevents writes after.
	if _, err := adminPool.Exec(ctx, fmt.Sprintf(
		"UPDATE pg_database SET datistemplate=true, datallowconn=false WHERE datname='%s'",
		templateDB,
	)); err != nil {
		return fmt.Errorf("mark template: %w", err)
	}

	return nil
}

// migrationsDir resolves backend/database/migrations regardless of where
// the test is invoked from. runtime.Caller gives the source path of this
// file (testutil/db.go), and migrations live two directories up.
func migrationsDir() string {
	return filepath.Join(backendDir(), "database", "migrations")
}

// schemaFilePath returns backend/database/schema.sql — the pre-migration
// baseline.
func schemaFilePath() string {
	return filepath.Join(backendDir(), "database", "schema.sql")
}

// backendDir returns the absolute path to the backend module root. Uses
// runtime.Caller so it's robust to the test's working directory.
func backendDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("testutil: cannot locate own source file")
	}
	// file = .../backend/internal/testutil/db.go → backend is two dirs up
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// dsnForDB rewrites the admin DSN to point at a specific database. The
// testcontainers connection string format is
// "postgres://user:pass@host:port/<dbname>?<params>".
func dsnForDB(name string) string {
	// Find "/postgres?" or "/postgres" at the end-ish of adminDSN and swap.
	// Simpler: testcontainers gives us a known shape; reconstruct manually.
	host, err := container.Host(context.Background())
	if err != nil {
		panic(fmt.Sprintf("testutil dsn host: %v", err))
	}
	port, err := container.MappedPort(context.Background(), "5432/tcp")
	if err != nil {
		panic(fmt.Sprintf("testutil dsn port: %v", err))
	}
	return fmt.Sprintf("postgres://turtask:turtask@%s:%s/%s?sslmode=disable",
		host, port.Port(), name)
}

// sanitize strips characters not allowed in unquoted Postgres identifiers.
// We only need lowercase ascii + digits + underscore; uuid hyphens go.
func sanitize(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			out = append(out, c)
		case c >= 'A' && c <= 'Z':
			out = append(out, c+('a'-'A'))
		}
	}
	return string(out)
}
