//go:build integration

// Integration tests for BoardService's board-level methods. CreateBoard is transactional
// (board + 4 columns + owner) and the read paths hit *db.Queries with aggregation and
// gate logic a mock cannot verify — rollback, COALESCE defaults, WHERE-clause gating.
package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/testutil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/util"
)

type boardFixture struct {
	pool    *pgxpool.Pool
	queries *db.Queries
	seed    *testutil.SeedHelper
	svc     *service.BoardService
	userID  string
}

func newBoardFixture(t *testing.T) *boardFixture {
	t.Helper()
	ctx := context.Background()
	pool := testutil.NewTestDB(t)
	queries := db.New(pool)
	seed := testutil.NewSeed(t, pool)
	return &boardFixture{
		pool:    pool,
		queries: queries,
		seed:    seed,
		svc:     service.NewBoardService(pool, queries),
		userID:  seed.User(ctx),
	}
}

// ────────────────────────────────────────────────
// CreateBoard
// ────────────────────────────────────────────────

func TestCreateBoard_EmptyTitle_Rejected(t *testing.T) {
	ctx := context.Background()
	f := newBoardFixture(t)

	_, err := f.svc.CreateBoard(ctx, "   ", nil, nil, nil, f.userID)
	assert.Error(t, err)
}

func TestCreateBoard_Success_CreatesFourDefaultColumnsAndOwnerMember(t *testing.T) {
	ctx := context.Background()
	f := newBoardFixture(t)

	boardID, err := f.svc.CreateBoard(ctx, "New Board", nil, nil, nil, f.userID)
	require.NoError(t, err)

	cols, err := f.queries.GetColumnsByBoardID(ctx, boardID)
	require.NoError(t, err)
	require.Len(t, cols, 4, "CreateBoard must seed exactly the 4 default columns")
	todoCount, doneCount := 0, 0
	for _, c := range cols {
		if c.Category == "DONE" {
			doneCount++
		} else if c.Category == "TODO" {
			todoCount++
		}
	}
	assert.Equal(t, 3, todoCount)
	assert.Equal(t, 1, doneCount)

	role, err := f.queries.GetBoardMemberRole(ctx, db.GetBoardMemberRoleParams{BoardID: boardID, UserID: f.userID})
	require.NoError(t, err)
	assert.Equal(t, "owner", role, "the creator must be added as owner in the same transaction")
}

// description/color/icon are optional at create time; a nil narg falls back
// to the column default in SQL via COALESCE. Pins the three documented
// defaults so a schema change that touches them gets caught here.
func TestCreateBoard_NilAppearance_FallsBackToColumnDefaults(t *testing.T) {
	ctx := context.Background()
	f := newBoardFixture(t)

	boardID, err := f.svc.CreateBoard(ctx, "Defaults Board", nil, nil, nil, f.userID)
	require.NoError(t, err)

	board, err := f.queries.GetBoardByID(ctx, boardID)
	require.NoError(t, err)
	assert.Equal(t, "", board.Description)
	assert.Equal(t, "#1E40AF", board.Color)
	assert.Equal(t, "board", board.Icon)
}

// AddBoardMember has an FK on user_id, so a nonexistent owner fails after the board and
// all four columns are already inserted. That is why CreateBoard needs a transaction:
// otherwise a caller is left with an orphan board nobody can see.
func TestCreateBoard_NonExistentOwner_RollsBackBoardAndColumnsToo(t *testing.T) {
	ctx := context.Background()
	f := newBoardFixture(t)

	_, err := f.svc.CreateBoard(ctx, "Should Not Exist", nil, nil, nil, "00000000-0000-0000-0000-000000000000")
	assert.Error(t, err)

	var count int
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM boards WHERE title = 'Should Not Exist'`,
	).Scan(&count))
	assert.Equal(t, 0, count, "the board row must not survive when adding the owner fails later in the same tx")
}

// ────────────────────────────────────────────────
// UpdateBoard
// ────────────────────────────────────────────────

func TestUpdateBoard_NilFields_PreserveExistingValues(t *testing.T) {
	ctx := context.Background()
	f := newBoardFixture(t)
	boardID, err := f.svc.CreateBoard(ctx, "Original Title", util.StringToPtr("Original desc"), nil, nil, f.userID)
	require.NoError(t, err)

	newTitle := "Renamed"
	updated, err := f.svc.UpdateBoard(ctx, boardID, &newTitle, nil, nil, nil, nil)
	require.NoError(t, err)

	assert.Equal(t, "Renamed", updated.Title)
	assert.Equal(t, "Original desc", updated.Description, "nil description must leave the existing value untouched")
}

func TestUpdateBoard_GivenFields_Overwrite(t *testing.T) {
	ctx := context.Background()
	f := newBoardFixture(t)
	boardID, err := f.svc.CreateBoard(ctx, "Board", nil, nil, nil, f.userID)
	require.NoError(t, err)

	newColor := "#FF0000"
	updated, err := f.svc.UpdateBoard(ctx, boardID, nil, nil, nil, &newColor, nil)
	require.NoError(t, err)
	assert.Equal(t, "#FF0000", updated.Color)
}

// ────────────────────────────────────────────────
// GetBoardWithCards
// ────────────────────────────────────────────────

func TestGetBoardWithCards_NoColumns_ReturnsEmptySliceNotNil(t *testing.T) {
	ctx := context.Background()
	f := newBoardFixture(t)
	boardID := f.seed.Board(ctx, f.userID) // testutil's Board seeds no columns

	result, err := f.svc.GetBoardWithCards(ctx, boardID)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestGetBoardWithCards_ColumnWithNoCards_CardsIsEmptySliceNotNil(t *testing.T) {
	ctx := context.Background()
	f := newBoardFixture(t)
	boardID := f.seed.Board(ctx, f.userID)
	f.seed.Column(ctx, boardID, "TODO", 1)

	result, err := f.svc.GetBoardWithCards(ctx, boardID)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.NotNil(t, result[0].Cards, "a column with zero cards must serialise as [] to the frontend, not null")
	assert.Empty(t, result[0].Cards)
}

func TestGetBoardWithCards_AttachesTagsToTheRightCard(t *testing.T) {
	ctx := context.Background()
	f := newBoardFixture(t)
	boardID := f.seed.Board(ctx, f.userID)
	columnID := f.seed.Column(ctx, boardID, "TODO", 1)
	cardA := f.seed.Card(ctx, columnID)
	cardB := f.seed.Card(ctx, columnID)
	tag := f.seed.Tag(ctx, boardID, "urgent")
	require.NoError(t, f.queries.InsertCardTag(ctx, db.InsertCardTagParams{CardID: cardA, TagID: tag}))

	result, err := f.svc.GetBoardWithCards(ctx, boardID)
	require.NoError(t, err)
	require.Len(t, result, 1)

	byID := map[string]service.CardData{}
	for _, c := range result[0].Cards {
		byID[c.ID] = c
	}
	assert.Len(t, byID[cardA].Tags, 1, "the tagged card must carry the tag")
	assert.Empty(t, byID[cardB].Tags, "the untagged card must not inherit the other card's tag")
}

// ────────────────────────────────────────────────
// CompleteMyTask
// ────────────────────────────────────────────────

func TestCompleteMyTask_Assignee_MarksDoneAndMovesToDoneColumn(t *testing.T) {
	ctx := context.Background()
	f := newBoardFixture(t)
	boardID := f.seed.Board(ctx, f.userID)
	todoCol := f.seed.Column(ctx, boardID, "TODO", 1)
	doneCol := f.seed.Column(ctx, boardID, "DONE", 2)
	cardID := f.seed.Card(ctx, todoCol)
	require.NoError(t, f.pool.QueryRow(ctx,
		`UPDATE cards SET assignee_id = $1 WHERE id = $2 RETURNING id`, f.userID, cardID,
	).Scan(new(string)))

	result, err := f.svc.CompleteMyTask(ctx, cardID, f.userID)
	require.NoError(t, err)
	assert.True(t, result.OK)
	assert.Equal(t, boardID, result.BoardID)

	card, err := f.queries.GetCard(ctx, cardID)
	require.NoError(t, err)
	assert.True(t, card.IsDone)
	assert.Equal(t, doneCol, card.ColumnID)
}

// The assignee gate is enforced by the SQL's WHERE assignee_id = $3, not by
// a Go-level check the caller could bypass — this confirms the DB itself
// rejects it, not just that the service happens to be called correctly.
func TestCompleteMyTask_NotTheAssignee_ReturnsOKFalseAndLeavesCardUntouched(t *testing.T) {
	ctx := context.Background()
	f := newBoardFixture(t)
	boardID := f.seed.Board(ctx, f.userID)
	todoCol := f.seed.Column(ctx, boardID, "TODO", 1)
	f.seed.Column(ctx, boardID, "DONE", 2)
	cardID := f.seed.Card(ctx, todoCol) // left unassigned

	someoneElse := f.seed.User(ctx)
	result, err := f.svc.CompleteMyTask(ctx, cardID, someoneElse)
	require.NoError(t, err)
	assert.False(t, result.OK, "a non-assignee's complete attempt must not report success")

	card, err := f.queries.GetCard(ctx, cardID)
	require.NoError(t, err)
	assert.False(t, card.IsDone, "the card must be untouched, not silently completed")
}

// ────────────────────────────────────────────────
// GetMyWork
// ────────────────────────────────────────────────

// Counts must reflect the full inbox even when Filter narrows the returned
// card list — this is what lets the frontend show filter-chip totals that
// don't change as you switch chips.
func TestGetMyWork_CountsCoverFullInbox_RegardlessOfFilter(t *testing.T) {
	ctx := context.Background()
	f := newBoardFixture(t)
	boardID := f.seed.Board(ctx, f.userID)
	columnID := f.seed.Column(ctx, boardID, "TODO", 1)

	today := service.MyWorkToday(time.Now(), "Asia/Bangkok")
	for i := 0; i < 3; i++ {
		cardID := f.seed.Card(ctx, columnID)
		require.NoError(t, f.pool.QueryRow(ctx,
			`UPDATE cards SET assignee_id = $1, due_date = $2 WHERE id = $3 RETURNING id`,
			f.userID, today, cardID,
		).Scan(new(string)))
	}

	result, err := f.svc.GetMyWork(ctx, service.MyWorkOptions{
		UserID: f.userID, Today: today, Filter: service.MyWorkFilterOverdue,
	})
	require.NoError(t, err)

	assert.Empty(t, result.Cards, "filtering to 'overdue' must return none of the 3 due-today cards")
	assert.Equal(t, 3, result.Counts.Today, "but the today-count must still reflect all 3, independent of the filter")
	assert.Equal(t, 3, result.Counts.Total)
}

func TestGetMyWork_UnassignedCard_ExcludedUnlessIncludeUnassigned(t *testing.T) {
	ctx := context.Background()
	f := newBoardFixture(t)
	boardID := f.seed.Board(ctx, f.userID)
	columnID := f.seed.Column(ctx, boardID, "TODO", 1)
	f.seed.Card(ctx, columnID) // unassigned

	today := service.MyWorkToday(time.Now(), "Asia/Bangkok")

	excluded, err := f.svc.GetMyWork(ctx, service.MyWorkOptions{UserID: f.userID, Today: today, IncludeUnassigned: false})
	require.NoError(t, err)
	assert.Equal(t, 0, excluded.Counts.Total)

	included, err := f.svc.GetMyWork(ctx, service.MyWorkOptions{UserID: f.userID, Today: today, IncludeUnassigned: true})
	require.NoError(t, err)
	assert.Equal(t, 1, included.Counts.Total, "with IncludeUnassigned, an unassigned card on the caller's own board must count")
}
