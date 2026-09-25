//go:build integration

package migrate_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/require"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/testutil"
)

// make test only runs forward, so a broken DOWN migration stays invisible until a real rollback.
func TestMigrations_DownUpRoundTrip(t *testing.T) {
	pool := testutil.NewTestDB(t)

	m, err := migrate.New(toFileURL(migrationsDir()), toPgx5(pool.Config().ConnString()))
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = m.Close() })

	require.NoError(t, m.Down(), "down migrations must all revert cleanly")
	require.NoError(t, m.Up(), "re-applying up after a full down must succeed")
}

// ToSlash is load-bearing on Windows — see migrate.fileSourceURL.
func toFileURL(path string) string {
	return "file://" + filepath.ToSlash(path)
}

func toPgx5(dsn string) string {
	for _, p := range []string{"postgres://", "postgresql://"} {
		if strings.HasPrefix(dsn, p) {
			return "pgx5://" + strings.TrimPrefix(dsn, p)
		}
	}
	return dsn
}

func migrationsDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot locate own source file")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "database", "migrations")
}
