//go:build integration

// One Postgres container per test binary; each NewTestDB clones a migrated template. Build tag: integration.
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
	adminDSN    string
	templateErr error
)

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

	// CREATE DATABASE takes no parameters; dbName is sanitized to [a-z0-9_].
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
		// Best-effort: a leaked DB only costs memory in the container.
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
	adminDSN = base

	adminPool, err := pgxpool.New(ctx, adminDSN)
	if err != nil {
		return fmt.Errorf("admin pool: %w", err)
	}
	defer adminPool.Close()

	if _, err := adminPool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", templateDB)); err != nil {
		return fmt.Errorf("create template: %w", err)
	}

	// Mirror production bootstrap — don't also call migrate.Run, replaying fails on existing tables.
	if err := migrate.Bootstrap(ctx, dsnForDB(templateDB), schemaFilePath(), migrationsDir()); err != nil {
		return fmt.Errorf("template bootstrap: %w", err)
	}

	// Template: CREATE DATABASE ... TEMPLATE without superuser; datallowconn=false blocks writes.
	if _, err := adminPool.Exec(ctx, fmt.Sprintf(
		"UPDATE pg_database SET datistemplate=true, datallowconn=false WHERE datname='%s'",
		templateDB,
	)); err != nil {
		return fmt.Errorf("mark template: %w", err)
	}

	return nil
}

func migrationsDir() string {
	return filepath.Join(backendDir(), "database", "migrations")
}

func schemaFilePath() string {
	return filepath.Join(backendDir(), "database", "schema.sql")
}

func backendDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("testutil: cannot locate own source file")
	}
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func dsnForDB(name string) string {
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
