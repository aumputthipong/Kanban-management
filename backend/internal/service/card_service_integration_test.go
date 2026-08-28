//go:build integration

// Integration tests for BoardService's card methods (card_service.go).
// UpdateCard runs a transaction (card fields + replacing tags) and its SQL
// mixes two update strategies on purpose — most columns are overwritten
// outright, but acceptance_criteria/implementation_note use COALESCE so an
// unrelated edit can't wipe what PromoteItem copied in. Mocking *db.Queries
// can't verify either the transaction's atomicity or that COALESCE actually
// behaves as documented, so this needs a real Postgres.
package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/testutil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/util"
)

type cardFixture struct {
	pool     *pgxpool.Pool
	queries  *db.Queries
	svc      *service.BoardService
	boardID  string
	columnID string
	cardID   string
}

func newCardFixture(t *testing.T) *cardFixture {
	t.Helper()
	ctx := context.Background()
	pool := testutil.NewTestDB(t)
	queries := db.New(pool)
	seed := testutil.NewSeed(t, pool)

	userID := seed.User(ctx)
	boardID := seed.Board(ctx, userID)
	columnID := seed.Column(ctx, boardID, "TODO", 1)
	cardID := seed.Card(ctx, columnID)

	return &cardFixture{
		pool:     pool,
		queries:  queries,
		svc:      service.NewBoardService(pool, queries),
		boardID:  boardID,
		columnID: columnID,
		cardID:   cardID,
	}
}

func (f *cardFixture) cardTags(ctx context.Context, t *testing.T) []string {
	t.Helper()
	rows, err := f.queries.GetTagsByCardIDs(ctx, []string{f.cardID})
	require.NoError(t, err)
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	return ids
}

// ────────────────────────────────────────────────
// UpdateCard — basic fields
// ────────────────────────────────────────────────

func TestUpdateCard_Success_UpdatesGivenFields(t *testing.T) {
	ctx := context.Background()
	f := newCardFixture(t)

	updated, err := f.svc.UpdateCard(ctx, service.UpdateCardParams{
		ID:          f.cardID,
		Title:       "New Title",
		Description: util.StringToPtr("New description"),
	})
	require.NoError(t, err)
	assert.Equal(t, "New Title", updated.Title)
	require.NotNil(t, updated.Description)
	assert.Equal(t, "New description", *updated.Description)
}

// title/description/due_date/assignee_id/priority/estimated_hours are plain
// SET, not COALESCE (see the SQL's own comment) — the caller is expected to
// have already merged in the existing values for anything it isn't
// changing. This pins that behaviour: passing a nil Description really does
// null it out, it does not "leave description alone".
func TestUpdateCard_NilDescription_ClearsIt_NotCOALESCEd(t *testing.T) {
	ctx := context.Background()
	f := newCardFixture(t)

	_, err := f.svc.UpdateCard(ctx, service.UpdateCardParams{
		ID: f.cardID, Title: "Has a description", Description: util.StringToPtr("will be cleared"),
	})
	require.NoError(t, err)

	updated, err := f.svc.UpdateCard(ctx, service.UpdateCardParams{
		ID: f.cardID, Title: "Has a description", Description: nil,
	})
	require.NoError(t, err)
	assert.Nil(t, updated.Description, "unlike AcceptanceCriteria/ImplementationNote, Description has no COALESCE — nil overwrites, it doesn't preserve")
}

// The one pair of fields that IS COALESCE'd: an update that only touches
// the title must not wipe acceptance_criteria/implementation_note that
// PromoteItem copied in from a planning item. This is the exact bug the
// SQL comment says COALESCE exists to prevent.
func TestUpdateCard_AcceptanceCriteriaAndNote_PreservedWhenNotTouched(t *testing.T) {
	ctx := context.Background()
	f := newCardFixture(t)

	ac := "Given/When/Then..."
	note := "Watch for the race on X"
	_, err := f.svc.UpdateCard(ctx, service.UpdateCardParams{
		ID: f.cardID, Title: "Original", AcceptanceCriteria: &ac, ImplementationNote: &note,
	})
	require.NoError(t, err)

	// A later edit that only changes the title, and passes nil for both —
	// PATCH semantics: nil means "don't touch".
	updated, err := f.svc.UpdateCard(ctx, service.UpdateCardParams{
		ID: f.cardID, Title: "Renamed", AcceptanceCriteria: nil, ImplementationNote: nil,
	})
	require.NoError(t, err)

	assert.Equal(t, "Renamed", updated.Title)
	require.NotNil(t, updated.AcceptanceCriteria)
	assert.Equal(t, ac, *updated.AcceptanceCriteria, "AC must survive an edit that never touched it")
	require.NotNil(t, updated.ImplementationNote)
	assert.Equal(t, note, *updated.ImplementationNote, "dev note must survive an edit that never touched it")
}

// ────────────────────────────────────────────────
// UpdateCard — tags
// ────────────────────────────────────────────────

func TestUpdateCard_NilTagIDs_LeavesExistingTagsUntouched(t *testing.T) {
	ctx := context.Background()
	f := newCardFixture(t)
	seed := testutil.NewSeed(t, f.pool)
	tag := seed.Tag(ctx, f.boardID, "bug")
	require.NoError(t, f.queries.InsertCardTag(ctx, db.InsertCardTagParams{CardID: f.cardID, TagID: tag}))

	_, err := f.svc.UpdateCard(ctx, service.UpdateCardParams{
		ID: f.cardID, Title: "Renamed only", TagIDs: nil,
	})
	require.NoError(t, err)

	assert.Equal(t, []string{tag}, f.cardTags(ctx, t), "nil TagIDs must mean \"don't touch tags\", not \"clear them\"")
}

func TestUpdateCard_EmptyTagIDs_ClearsAllTags(t *testing.T) {
	ctx := context.Background()
	f := newCardFixture(t)
	seed := testutil.NewSeed(t, f.pool)
	tag := seed.Tag(ctx, f.boardID, "bug")
	require.NoError(t, f.queries.InsertCardTag(ctx, db.InsertCardTagParams{CardID: f.cardID, TagID: tag}))

	empty := []string{}
	_, err := f.svc.UpdateCard(ctx, service.UpdateCardParams{
		ID: f.cardID, Title: "Clearing tags", TagIDs: &empty,
	})
	require.NoError(t, err)

	assert.Empty(t, f.cardTags(ctx, t))
}

func TestUpdateCard_ReplaceTagIDs_SwapsToExactlyTheNewSet(t *testing.T) {
	ctx := context.Background()
	f := newCardFixture(t)
	seed := testutil.NewSeed(t, f.pool)
	oldTag := seed.Tag(ctx, f.boardID, "old")
	newTag := seed.Tag(ctx, f.boardID, "new")
	require.NoError(t, f.queries.InsertCardTag(ctx, db.InsertCardTagParams{CardID: f.cardID, TagID: oldTag}))

	newSet := []string{newTag}
	_, err := f.svc.UpdateCard(ctx, service.UpdateCardParams{
		ID: f.cardID, Title: "Swapping tags", TagIDs: &newSet,
	})
	require.NoError(t, err)

	assert.Equal(t, []string{newTag}, f.cardTags(ctx, t))
}

// The >5 check runs AFTER qtx.UpdateCard has already applied the title/field
// change, inside the same transaction — this is the whole reason UpdateCard
// needs a transaction at all. Confirms both halves of that: the tag-count
// error surfaces, AND the title change from earlier in the same call rolls
// back with it rather than silently landing on its own.
func TestUpdateCard_MoreThanFiveTags_ErrorsAndRollsBackEntireUpdate(t *testing.T) {
	ctx := context.Background()
	f := newCardFixture(t)
	seed := testutil.NewSeed(t, f.pool)

	tooMany := make([]string, 0, 6)
	for i := 0; i < 6; i++ {
		tooMany = append(tooMany, seed.Tag(ctx, f.boardID, fmt.Sprintf("tag-%d", i)))
	}

	_, err := f.svc.UpdateCard(ctx, service.UpdateCardParams{
		ID: f.cardID, Title: "Should not stick", TagIDs: &tooMany,
	})
	assert.Error(t, err)

	reloaded, gerr := f.queries.GetCard(ctx, f.cardID)
	require.NoError(t, gerr)
	assert.Equal(t, "Test Card", reloaded.Title, "the title change earlier in the same transaction must have rolled back too")
	assert.Empty(t, f.cardTags(ctx, t), "no tags should have been attached either")
}

// ────────────────────────────────────────────────
// GetCardDetail
// ────────────────────────────────────────────────

func TestGetCardDetail_AggregatesSubtasksAndTags(t *testing.T) {
	ctx := context.Background()
	f := newCardFixture(t)
	seed := testutil.NewSeed(t, f.pool)
	tag := seed.Tag(ctx, f.boardID, "feature")
	require.NoError(t, f.queries.InsertCardTag(ctx, db.InsertCardTagParams{CardID: f.cardID, TagID: tag}))
	_, err := f.queries.CreateSubtask(ctx, db.CreateSubtaskParams{CardID: f.cardID, Title: "Step 1", Position: 1})
	require.NoError(t, err)

	detail, err := f.svc.GetCardDetail(ctx, f.cardID)
	require.NoError(t, err)

	assert.Equal(t, f.cardID, detail.Card.ID)
	require.Len(t, detail.Subtasks, 1)
	assert.Equal(t, "Step 1", detail.Subtasks[0].Title)
	require.Len(t, detail.Tags, 1)
	assert.Equal(t, "feature", detail.Tags[0].Name)
	assert.Nil(t, detail.AssigneeName, "an unassigned card must report a nil assignee name, not error")
}
