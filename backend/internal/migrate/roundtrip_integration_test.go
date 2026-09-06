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

// `make test` only ever runs the schema forward, so a broken or out-of-order DOWN
// migration stays invisible until a real rollback. This walks every migration all the
// way down and back up against a real Postgres, failing if any step errors.
func TestMigrations_DownUpRoundTrip(t *testing.T) {
	pool := testutil.NewTestDB(t)

	m, err := migrate.New(toFileURL(migrationsDir()), toPgx5(pool.Config().ConnString()))
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = m.Close() })

	require.NoError(t, m.Down(), "down migrations must all revert cleanly")
	require.NoError(t, m.Up(), "re-applying up after a full down must succeed")
}

// toFileURL mirrors migrate.fileSourceURL, which is unexported. filepath.ToSlash is
// load-bearing on Windows — see that function's doc for why.
func toFileURL(path string) string {
	return "file://" + filepath.ToSlash(path)
}

// toPgx5 swaps the postgres:// scheme for the pgx5:// scheme golang-migrate's
// pgx/v5 driver expects (mirrors migrate.normalizeDBURL, which is unexported).
func toPgx5(dsn string) string {
	for _, p := range []string{"postgres://", "postgresql://"} {
		if strings.HasPrefix(dsn, p) {
			return "pgx5://" + strings.TrimPrefix(dsn, p)
		}
	}
	return dsn
}

// migrationsDir resolves backend/database/migrations from this file's location
// (backend/internal/migrate/), independent of the test's working directory.
func migrationsDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot locate own source file")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "database", "migrations")
}
