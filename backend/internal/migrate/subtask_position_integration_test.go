//go:build integration

package migrate_test

import (
	"context"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/testutil"
)

// Migration 000019 must renumber the duplicate positions real boards already have, in the
// order users currently see them, before the unique constraint can be added.
func TestMigration000019_RenumbersDuplicatePositionsInDisplayOrder(t *testing.T) {
	ctx := context.Background()
	pool := testutil.NewTestDB(t)
	seed := testutil.NewSeed(t, pool)
	cardID := seed.Card(ctx, seed.Column(ctx, seed.Board(ctx, seed.User(ctx)), "TODO", 1))

	m, err := migrate.New(toFileURL(migrationsDir()), toPgx5(pool.Config().ConnString()))
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = m.Close() })
	// Absolute versions, not Steps(±1): later migrations must not shift what this targets.
	require.NoError(t, m.Migrate(18), "step back to before 000019")

	// The real shape from production data: a gap at 2, then two rows sharing 3.
	_, err = pool.Exec(ctx, `
		INSERT INTO card_subtasks (card_id, title, position, created_at) VALUES
		($1, 'Member Filter', 1, '2026-04-08 10:47:23+00'),
		($1, 'Estimate',      3, '2026-04-08 12:23:17+00'),
		($1, 'g',             3, '2026-09-17 05:31:40+00')`, cardID)
	require.NoError(t, err)

	require.NoError(t, m.Migrate(19), "000019 up must succeed on data with duplicates")

	rows, err := pool.Query(ctx, `SELECT title, position FROM card_subtasks WHERE card_id = $1 ORDER BY position`, cardID)
	require.NoError(t, err)
	defer rows.Close()
	var got []string
	var positions []float64
	for rows.Next() {
		var title string
		var pos float64
		require.NoError(t, rows.Scan(&title, &pos))
		got = append(got, title)
		positions = append(positions, pos)
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, []string{"Member Filter", "Estimate", "g"}, got, "older duplicate stays first")
	assert.Equal(t, []float64{1, 2, 3}, positions)

	_, err = pool.Exec(ctx, `INSERT INTO card_subtasks (card_id, title, position) VALUES ($1, 'dup', 2)`, cardID)
	require.Error(t, err, "the unique constraint must now reject a repeated position")
}
