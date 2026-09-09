//go:build integration

// Integration tests for SubtaskService. UpdateSubtask does a
// read-then-merge-then-write in Go rather than letting the SQL's own
// COALESCE resolve unset fields (see the file's own comment) — that shape
// only shows a problem under concurrency, so this needs real overlapping
// requests against a real Postgres, not a mock.
package service_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/dto"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/testutil"
)

type subtaskFixture struct {
	queries   *db.Queries
	svc       *service.SubtaskService
	cardID    string
	subtaskID string
}

func newSubtaskFixture(t *testing.T) *subtaskFixture {
	t.Helper()
	ctx := context.Background()
	pool := testutil.NewTestDB(t)
	queries := db.New(pool)
	seed := testutil.NewSeed(t, pool)
	columnID := seed.Column(ctx, seed.Board(ctx, seed.User(ctx)), "TODO", 1)
	cardID := seed.Card(ctx, columnID)
	return &subtaskFixture{
		queries:   queries,
		svc:       service.NewSubtaskService(pool),
		cardID:    cardID,
		subtaskID: seed.Subtask(ctx, cardID),
	}
}

func TestCreateSubtask_Success(t *testing.T) {
	ctx := context.Background()
	f := newSubtaskFixture(t)

	st, err := f.svc.CreateSubtask(ctx, db.CreateSubtaskParams{CardID: f.cardID, Title: "New step", Position: 1})
	require.NoError(t, err)
	assert.Equal(t, "New step", st.Title)
	assert.False(t, st.IsDone)
}

// A partial update (title only) must leave is_done/position as they were.
// This currently works — but via the Go-side read-then-merge, not the SQL's
// COALESCE, which is why the next test demonstrates where that shape breaks.
func TestUpdateSubtask_PartialUpdate_PreservesOtherFields(t *testing.T) {
	ctx := context.Background()
	f := newSubtaskFixture(t)
	newTitle := "Renamed"

	updated, err := f.svc.UpdateSubtask(ctx, f.subtaskID, dto.UpdateSubtaskRequest{Title: &newTitle})
	require.NoError(t, err)
	assert.Equal(t, "Renamed", updated.Title)
	assert.False(t, updated.IsDone)
}

// Demonstrates the cost of the read-then-merge shape (not a fix): two callers
// editing DIFFERENT fields of the same subtask "at the same time" each start
// from the same stale snapshot. Whichever writes second overwrites the
// first's change to the field IT didn't touch, because its own merge still
// carries the old value for that field. A single UPDATE ... SET x =
// COALESCE(...) statement (bypassing the Go-side read) would not have this
// problem — this is the T3 finding from the earlier audit.
func TestUpdateSubtask_ConcurrentDifferentFieldEdits_OneEditIsLost(t *testing.T) {
	ctx := context.Background()
	f := newSubtaskFixture(t)

	var wg sync.WaitGroup
	start := make(chan struct{})
	newTitle := "Title changed by A"
	done := true

	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, _ = f.svc.UpdateSubtask(ctx, f.subtaskID, dto.UpdateSubtaskRequest{Title: &newTitle})
	}()
	go func() {
		defer wg.Done()
		<-start
		_, _ = f.svc.UpdateSubtask(ctx, f.subtaskID, dto.UpdateSubtaskRequest{IsDone: &done})
	}()
	close(start)
	wg.Wait()

	final, err := f.queries.GetSubtask(ctx, f.subtaskID)
	require.NoError(t, err)
	// Not asserting a specific outcome — which write "won" depends on
	// goroutine scheduling. The point this test exists to make is that BOTH
	// edits together are not guaranteed to land: log the observed state so a
	// lost edit is visible when it happens, without making the test flaky
	// by asserting a fixed order.
	t.Logf("T3: after concurrent edits, title=%q is_done=%v (a correct fix would guarantee both land: title=%q is_done=true)",
		final.Title, final.IsDone, newTitle)
}

func TestDeleteSubtask_Success(t *testing.T) {
	ctx := context.Background()
	f := newSubtaskFixture(t)

	require.NoError(t, f.svc.DeleteSubtask(ctx, f.subtaskID))

	_, err := f.queries.GetSubtask(ctx, f.subtaskID)
	assert.Error(t, err)
}

func TestGetSubtaskByID_NotFound_Errors(t *testing.T) {
	ctx := context.Background()
	f := newSubtaskFixture(t)

	_, err := f.svc.GetSubtaskByID(ctx, "00000000-0000-0000-0000-000000000000")
	assert.Error(t, err)
}
