//go:build integration

// Integration tests for SubtaskService. UpdateSubtask reads, merges in Go, then writes;
// that shape only misbehaves under concurrency, so it needs overlapping requests
// against a real Postgres.
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

func positionsOf(t *testing.T, f *subtaskFixture) []float64 {
	t.Helper()
	list, err := f.svc.GetSubtasksByCardID(context.Background(), f.cardID)
	require.NoError(t, err)
	out := make([]float64, len(list))
	for i, st := range list {
		out[i] = st.Position
	}
	return out
}

// The bug that shipped: the client sent count+1, which repeats a position once a middle
// subtask is deleted, and tied rows then swapped places on every update.
func TestCreateSubtask_AfterDeletingMiddle_AppendsAfterLast(t *testing.T) {
	ctx := context.Background()
	f := newSubtaskFixture(t) // seeds one subtask at position 1

	second, err := f.svc.CreateSubtask(ctx, f.cardID, "second")
	require.NoError(t, err)
	third, err := f.svc.CreateSubtask(ctx, f.cardID, "third")
	require.NoError(t, err)
	require.NoError(t, f.svc.DeleteSubtask(ctx, second.ID))

	added, err := f.svc.CreateSubtask(ctx, f.cardID, "added")
	require.NoError(t, err)

	assert.Greater(t, added.Position, third.Position, "a new subtask goes after the last one, never beside it")
	assert.Equal(t, []float64{1, 3, 4}, positionsOf(t, f))
}

func TestCreateSubtask_ConcurrentAppends_GetDistinctPositions(t *testing.T) {
	ctx := context.Background()
	f := newSubtaskFixture(t)

	const n = 10
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := f.svc.CreateSubtask(ctx, f.cardID, "concurrent")
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err, "the card row lock must serialize appends, not fail them")
	}

	positions := positionsOf(t, f)
	require.Len(t, positions, n+1)
	for i, p := range positions {
		assert.Equal(t, float64(i+1), p)
	}
}

func TestCardSubtasks_DuplicatePosition_RejectedByDatabase(t *testing.T) {
	ctx := context.Background()
	f := newSubtaskFixture(t)

	_, err := f.queries.CreateSubtask(ctx, db.CreateSubtaskParams{CardID: f.cardID, Title: "dup", Position: 1})

	require.Error(t, err, "UNIQUE(card_id, position) is the backstop if any code path skips AppendSubtask")
}

func TestCreateSubtask_Success(t *testing.T) {
	ctx := context.Background()
	f := newSubtaskFixture(t)

	st, err := f.svc.CreateSubtask(ctx, f.cardID, "New step")
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

// Documents a known lost-update (audit finding T3, not fixed here): two concurrent
// edits to different fields both merge from the same stale read, so the second write
// restores the first one's field. A single UPDATE with COALESCE would not.
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
	// No assertion: which write wins depends on scheduling, and asserting an order
	// would make the test flaky. Logging makes a lost edit visible when it happens.
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
