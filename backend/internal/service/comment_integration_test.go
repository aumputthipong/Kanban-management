//go:build integration

// Integration tests for the planning comment thread. The mock-level tests
// in handler/planning_comment_handler_test.go cover the permission matrix;
// these tests prove the SQL contract — soft delete preserves position,
// list returns deleted rows, and edit-on-already-deleted returns the
// sentinel.
package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/testutil"
)

func TestComments_SoftDeletePreservesPosition(t *testing.T) {
	// A delete in the middle of a thread must NOT shift later comments
	// up — the thread's reading order is part of the conversation. The
	// list endpoint returns deleted rows (with body intact at the SQL
	// level; the handler nils it out before the response) so the UI can
	// render them as "deleted" placeholders in their original slot.
	ctx := context.Background()
	pool := testutil.NewTestDB(t)
	seed := testutil.NewSeed(t, pool)
	owner := seed.User(ctx)
	boardID := seed.Board(ctx, owner)
	sessID := seed.PlanningSession(ctx, boardID, owner)
	itemID := seed.PlanningItem(ctx, sessID)
	svc := service.NewPlanningService(pool, db.New(pool))

	c1, err := svc.CreateComment(ctx, itemID, owner, "first")
	require.NoError(t, err)
	c2, err := svc.CreateComment(ctx, itemID, owner, "second")
	require.NoError(t, err)
	c3, err := svc.CreateComment(ctx, itemID, owner, "third")
	require.NoError(t, err)

	require.NoError(t, svc.DeleteComment(ctx, c2.ID))

	rows, err := svc.ListItemComments(ctx, itemID)
	require.NoError(t, err)
	require.Len(t, rows, 3, "deleted comments must still appear in the list — UI renders the tombstone")
	assert.Equal(t, c1.ID, rows[0].ID)
	assert.Equal(t, c2.ID, rows[1].ID)
	assert.Equal(t, c3.ID, rows[2].ID)
	require.NotNil(t, rows[1].DeletedAt, "middle row must carry the soft-delete marker")
	assert.Nil(t, rows[0].DeletedAt)
	assert.Nil(t, rows[2].DeletedAt)
}

func TestEditComment_OnSoftDeleted_ReturnsSentinel(t *testing.T) {
	// The UpdatePlanningItemComment query is guarded by
	// "deleted_at IS NULL" so an edit landing after a delete falls
	// through to zero rows updated. The service maps that to a typed
	// sentinel so handlers can return 409 cleanly — without the guard,
	// an edit would resurrect the body on a tombstoned row.
	ctx := context.Background()
	pool := testutil.NewTestDB(t)
	seed := testutil.NewSeed(t, pool)
	owner := seed.User(ctx)
	boardID := seed.Board(ctx, owner)
	sessID := seed.PlanningSession(ctx, boardID, owner)
	itemID := seed.PlanningItem(ctx, sessID)
	svc := service.NewPlanningService(pool, db.New(pool))

	c, err := svc.CreateComment(ctx, itemID, owner, "to be deleted then edited")
	require.NoError(t, err)
	require.NoError(t, svc.DeleteComment(ctx, c.ID))

	_, err = svc.EditComment(ctx, c.ID, "ghost edit")
	require.ErrorIs(t, err, service.ErrPlanningCommentDeleted)
}
