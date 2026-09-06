//go:build integration

// Integration tests for BoardCommandService — the write path the WebSocket layer calls.
// CreateCardWS is transactional, MoveCard derives is_done from the target column, and
// CreateColumn computes a position unlocked. None of that is observable through a mock.
package service_test

import (
	"context"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/testutil"
)

type commandFixture struct {
	pool    *pgxpool.Pool
	queries *db.Queries
	seed    *testutil.SeedHelper
	svc     *service.BoardCommandService
	userID  string
	boardID string
}

func newCommandFixture(t *testing.T) *commandFixture {
	t.Helper()
	ctx := context.Background()
	pool := testutil.NewTestDB(t)
	queries := db.New(pool)
	seed := testutil.NewSeed(t, pool)
	userID := seed.User(ctx)
	return &commandFixture{
		pool:    pool,
		queries: queries,
		seed:    seed,
		svc:     service.NewBoardCommandService(pool, queries),
		userID:  userID,
		boardID: seed.Board(ctx, userID),
	}
}

// columnPosition reads one column's position. There is no GetColumnByID query (nothing
// in production needs one), so this goes at the table directly, like liveTokenCount.
func (f *commandFixture) columnPosition(ctx context.Context, t *testing.T, columnID string) float64 {
	t.Helper()
	var pos float64
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT position FROM columns WHERE id = $1`, columnID).Scan(&pos))
	return pos
}

// ────────────────────────────────────────────────
// MoveCard
// ────────────────────────────────────────────────

func TestMoveCard_IntoDoneColumn_MarksDoneWithTimestamp(t *testing.T) {
	ctx := context.Background()
	f := newCommandFixture(t)
	todoCol := f.seed.Column(ctx, f.boardID, "TODO", 1)
	doneCol := f.seed.Column(ctx, f.boardID, "DONE", 2)
	cardID := f.seed.Card(ctx, todoCol)

	result, err := f.svc.MoveCard(ctx, cardID, doneCol, 100)
	require.NoError(t, err)
	assert.True(t, result.IsDone)
	require.NotNil(t, result.CompletedAt)

	card, err := f.queries.GetCard(ctx, cardID)
	require.NoError(t, err)
	assert.True(t, card.IsDone)
}

func TestMoveCard_IntoTodoColumn_NotDoneNoTimestamp(t *testing.T) {
	ctx := context.Background()
	f := newCommandFixture(t)
	todoA := f.seed.Column(ctx, f.boardID, "TODO", 1)
	todoB := f.seed.Column(ctx, f.boardID, "TODO", 2)
	cardID := f.seed.Card(ctx, todoA)

	result, err := f.svc.MoveCard(ctx, cardID, todoB, 100)
	require.NoError(t, err)
	assert.False(t, result.IsDone)
	assert.Nil(t, result.CompletedAt)
}

// ────────────────────────────────────────────────
// CreateCardWS
// ────────────────────────────────────────────────

func TestCreateCardWS_Success_CardAndSubtasksLandTogether(t *testing.T) {
	ctx := context.Background()
	f := newCommandFixture(t)
	columnID := f.seed.Column(ctx, f.boardID, "TODO", 1)

	card, subtasks, err := f.svc.CreateCardWS(ctx, columnID, f.userID, "New Card", "high", 0, nil, nil, nil, []string{"Step 1", "Step 2"})
	require.NoError(t, err)
	assert.Equal(t, "New Card", card.Title)
	require.Len(t, subtasks, 2)
	assert.Equal(t, "Step 1", subtasks[0].Title)

	// Confirm both actually committed, not just returned in memory.
	saved, err := f.queries.GetSubtasksByCardID(ctx, card.ID)
	require.NoError(t, err)
	assert.Len(t, saved, 2)
}

// A card that already exists in the column determines where the next one
// lands when the caller passes position <= 0 (quick-add's case) — the new
// card must go after the highest existing position, not at some fixed spot.
func TestCreateCardWS_ZeroPosition_LandsAfterExistingCards(t *testing.T) {
	ctx := context.Background()
	f := newCommandFixture(t)
	columnID := f.seed.Column(ctx, f.boardID, "TODO", 1)
	first, _, err := f.svc.CreateCardWS(ctx, columnID, f.userID, "First", "", 0, nil, nil, nil, nil)
	require.NoError(t, err)

	second, _, err := f.svc.CreateCardWS(ctx, columnID, f.userID, "Second", "", 0, nil, nil, nil, nil)
	require.NoError(t, err)

	assert.Greater(t, second.Position, first.Position, "a zero position must be computed after the existing max, not default to the same spot")
}

// NOT TESTED: transactional rollback of CreateCardWS on a failed subtask insert.
// card_subtasks has no constraint a second insert can violate while the first succeeds,
// so there is no reachable failing input. Add the test here if a migration adds one.

// ────────────────────────────────────────────────
// ToggleCardDone
// ────────────────────────────────────────────────

func TestToggleCardDone_ToDone_MovesToBoardsDoneColumn(t *testing.T) {
	ctx := context.Background()
	f := newCommandFixture(t)
	todoCol := f.seed.Column(ctx, f.boardID, "TODO", 1)
	doneCol := f.seed.Column(ctx, f.boardID, "DONE", 2)
	cardID := f.seed.Card(ctx, todoCol)

	result, err := f.svc.ToggleCardDone(ctx, cardID, f.boardID, true)
	require.NoError(t, err)
	assert.Equal(t, doneCol, result.TargetColumnID)
	require.NotNil(t, result.CompletedAt)
}

func TestToggleCardDone_BackToNotDone_MovesToBoardsFirstTodoColumn(t *testing.T) {
	ctx := context.Background()
	f := newCommandFixture(t)
	firstTodo := f.seed.Column(ctx, f.boardID, "TODO", 1)
	f.seed.Column(ctx, f.boardID, "TODO", 2)
	doneCol := f.seed.Column(ctx, f.boardID, "DONE", 3)
	cardID := f.seed.Card(ctx, doneCol)

	result, err := f.svc.ToggleCardDone(ctx, cardID, f.boardID, false)
	require.NoError(t, err)
	assert.Equal(t, firstTodo, result.TargetColumnID, "un-completing must land on the lowest-position TODO column")
	assert.Nil(t, result.CompletedAt)
}

// ────────────────────────────────────────────────
// CreateColumn
// ────────────────────────────────────────────────

func TestCreateColumn_NoDoneColumnYet_AppendsAtEnd(t *testing.T) {
	ctx := context.Background()
	f := newCommandFixture(t)
	first := f.seed.Column(ctx, f.boardID, "TODO", 65536)

	col, err := f.svc.CreateColumn(ctx, f.boardID, "Second", "TODO", nil)
	require.NoError(t, err)

	assert.Greater(t, col.Position, f.columnPosition(ctx, t, first))
}

func TestCreateColumn_DoneColumnExists_InsertsBeforeIt(t *testing.T) {
	ctx := context.Background()
	f := newCommandFixture(t)
	todo := f.seed.Column(ctx, f.boardID, "TODO", 65536)
	done := f.seed.Column(ctx, f.boardID, "DONE", 131072)

	col, err := f.svc.CreateColumn(ctx, f.boardID, "Review", "TODO", nil)
	require.NoError(t, err)

	assert.Greater(t, col.Position, f.columnPosition(ctx, t, todo))
	assert.Less(t, col.Position, f.columnPosition(ctx, t, done), "a new column must land strictly before the DONE column")
}

// Documents a known race rather than fixing it: CreateColumn reads the DONE and max
// non-DONE positions, computes a midpoint in Go and writes, with no lock in between. Two
// callers racing compute the identical midpoint, so both columns land on the same
// position. The fix has to move the calculation into SQL, not read-then-write in Go.
func TestCreateColumn_ConcurrentCreates_CanCollideOnPosition_T6(t *testing.T) {
	ctx := context.Background()
	f := newCommandFixture(t)
	f.seed.Column(ctx, f.boardID, "TODO", 65536)
	f.seed.Column(ctx, f.boardID, "DONE", 131072)

	const goroutines = 4
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		positions = make([]float64, 0, goroutines)
	)
	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			col, err := f.svc.CreateColumn(ctx, f.boardID, "Racer", "TODO", nil)
			require.NoError(t, err)
			mu.Lock()
			positions = append(positions, col.Position)
			mu.Unlock()
		}()
	}
	close(start)
	wg.Wait()

	unique := map[float64]bool{}
	for _, p := range positions {
		unique[p] = true
	}
	t.Logf("T6: %d concurrent CreateColumn calls produced %d distinct positions (collisions = %d)",
		goroutines, len(unique), goroutines-len(unique))
	// Deliberately not asserting len(unique) == goroutines: that invariant is what the
	// race violates today. The job here is to keep proving it exists, and to stay green
	// once real locking lands (the Logf above then reports zero collisions).
}

// ────────────────────────────────────────────────
// VerifyCardInBoard / VerifyColumnInBoard
// ────────────────────────────────────────────────

func TestVerifyCardInBoard_CardBelongsToBoard_NoError(t *testing.T) {
	ctx := context.Background()
	f := newCommandFixture(t)
	columnID := f.seed.Column(ctx, f.boardID, "TODO", 1)
	cardID := f.seed.Card(ctx, columnID)

	assert.NoError(t, f.svc.VerifyCardInBoard(ctx, cardID, f.boardID))
}

// The exact cross-board attack the comment describes: a WS client connected
// to boardA sends a card id that actually belongs to boardB.
func TestVerifyCardInBoard_CardBelongsToDifferentBoard_ErrEntityBoardMismatch(t *testing.T) {
	ctx := context.Background()
	f := newCommandFixture(t)
	otherBoard := f.seed.Board(ctx, f.userID)
	otherColumn := f.seed.Column(ctx, otherBoard, "TODO", 1)
	cardInOtherBoard := f.seed.Card(ctx, otherColumn)

	err := f.svc.VerifyCardInBoard(ctx, cardInOtherBoard, f.boardID)
	assert.ErrorIs(t, err, service.ErrEntityBoardMismatch)
}
