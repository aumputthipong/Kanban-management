//go:build integration

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

// Regression: client-side count+1 repeated a position after a middle delete.
func TestCreateSubtask_AfterDeletingMiddle_AppendsAfterLast(t *testing.T) {
	ctx := context.Background()
	f := newSubtaskFixture(t)

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

func TestUpdateSubtask_PartialUpdate_PreservesOtherFields(t *testing.T) {
	ctx := context.Background()
	f := newSubtaskFixture(t)
	newTitle := "Renamed"

	updated, err := f.svc.UpdateSubtask(ctx, f.subtaskID, dto.UpdateSubtaskRequest{Title: &newTitle})
	require.NoError(t, err)
	assert.Equal(t, "Renamed", updated.Title)
	assert.False(t, updated.IsDone)
}

// Repeated because one round can pass by luck.
func TestUpdateSubtask_ConcurrentDifferentFieldEdits_BothLand(t *testing.T) {
	ctx := context.Background()
	f := newSubtaskFixture(t)

	for round := 0; round < 20; round++ {
		reset, notDone := "original", false
		_, err := f.svc.UpdateSubtask(ctx, f.subtaskID, dto.UpdateSubtaskRequest{Title: &reset, IsDone: &notDone})
		require.NoError(t, err)

		var wg sync.WaitGroup
		start := make(chan struct{})
		newTitle, done := "Title changed by A", true
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
		require.Equal(t, newTitle, final.Title, "round %d: title edit lost", round)
		require.True(t, final.IsDone, "round %d: is_done edit lost", round)
	}
}

func TestUpdateSubtask_NotFound_Errors(t *testing.T) {
	f := newSubtaskFixture(t)
	title := "x"

	_, err := f.svc.UpdateSubtask(context.Background(), "00000000-0000-0000-0000-000000000000", dto.UpdateSubtaskRequest{Title: &title})
	assert.Error(t, err)
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
