//go:build integration

// Integration tests for BoardService's card methods. UpdateCard is transactional and merges
// every field in SQL, so an edit never overwrites a field it did not send — including under
// concurrency, which only a real Postgres can show.
package service_test

import (
	"context"
	"fmt"
	"sync/atomic"
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
		Title:       util.StringToPtr("New Title"),
		Description: util.StringToPtr("New description"),
	})
	require.NoError(t, err)
	assert.Equal(t, "New Title", updated.Card.Title)
	require.NotNil(t, updated.Card.Description)
	assert.Equal(t, "New description", *updated.Card.Description)
}

// nil means "no change" for every field; a field the caller did not send keeps its value.
func TestUpdateCard_NilFields_LeaveStoredValuesAlone(t *testing.T) {
	ctx := context.Background()
	f := newCardFixture(t)
	assignee := testutil.NewSeed(t, f.pool).User(ctx)
	due := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	hours := 3.5

	_, err := f.svc.UpdateCard(ctx, service.UpdateCardParams{
		ID: f.cardID, Description: util.StringToPtr("keep me"),
		DueDate:        service.FieldPatch[time.Time]{Set: true, Value: &due},
		AssigneeID:     service.FieldPatch[string]{Set: true, Value: &assignee},
		Priority:       service.FieldPatch[string]{Set: true, Value: util.StringToPtr("high")},
		EstimatedHours: service.FieldPatch[float64]{Set: true, Value: &hours},
	})
	require.NoError(t, err)

	updated, err := f.svc.UpdateCard(ctx, service.UpdateCardParams{ID: f.cardID, Title: util.StringToPtr("Renamed")})
	require.NoError(t, err)

	c := updated.Card
	assert.Equal(t, "Renamed", c.Title)
	require.NotNil(t, c.Description)
	assert.Equal(t, "keep me", *c.Description)
	require.NotNil(t, c.DueDate)
	assert.True(t, due.Equal(*c.DueDate))
	require.NotNil(t, c.AssigneeID)
	assert.Equal(t, assignee, *c.AssigneeID)
	require.NotNil(t, c.Priority)
	assert.Equal(t, "high", *c.Priority)
	require.NotNil(t, util.PgNumericToFloat64Ptr(c.EstimatedHours))
}

// Set with a nil Value is the only way to clear a nullable column.
func TestUpdateCard_SetNil_ClearsNullableColumns(t *testing.T) {
	ctx := context.Background()
	f := newCardFixture(t)
	assignee := testutil.NewSeed(t, f.pool).User(ctx)
	due := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	hours := 2.0
	_, err := f.svc.UpdateCard(ctx, service.UpdateCardParams{
		ID:             f.cardID,
		DueDate:        service.FieldPatch[time.Time]{Set: true, Value: &due},
		AssigneeID:     service.FieldPatch[string]{Set: true, Value: &assignee},
		Priority:       service.FieldPatch[string]{Set: true, Value: util.StringToPtr("low")},
		EstimatedHours: service.FieldPatch[float64]{Set: true, Value: &hours},
	})
	require.NoError(t, err)

	updated, err := f.svc.UpdateCard(ctx, service.UpdateCardParams{
		ID:             f.cardID,
		DueDate:        service.FieldPatch[time.Time]{Set: true},
		AssigneeID:     service.FieldPatch[string]{Set: true},
		Priority:       service.FieldPatch[string]{Set: true},
		EstimatedHours: service.FieldPatch[float64]{Set: true},
	})
	require.NoError(t, err)

	c := updated.Card
	assert.Nil(t, c.DueDate)
	assert.Nil(t, c.AssigneeID)
	assert.Nil(t, c.Priority)
	assert.Nil(t, util.PgNumericToFloat64Ptr(c.EstimatedHours))
	assert.Equal(t, "Test Card", c.Title, "clearing other fields must not touch the title")
}

// Regression for T2: the handler used to read the card, merge the request, and write every
// column back, so two edits of different fields raced and the later write undid the other.
func TestUpdateCard_ConcurrentDifferentFieldEdits_BothLand(t *testing.T) {
	ctx := context.Background()
	f := newCardFixture(t)

	for round := 0; round < 20; round++ {
		_, err := f.svc.UpdateCard(ctx, service.UpdateCardParams{
			ID: f.cardID, Title: util.StringToPtr("original"),
			Priority: service.FieldPatch[string]{Set: true},
		})
		require.NoError(t, err)

		newTitle := fmt.Sprintf("renamed %d", round)
		edits := []service.UpdateCardParams{
			{ID: f.cardID, Title: &newTitle},
			{ID: f.cardID, Priority: service.FieldPatch[string]{Set: true, Value: util.StringToPtr("high")}},
		}
		var next atomic.Int32
		for _, err := range raceN(2, func() error {
			_, err := f.svc.UpdateCard(ctx, edits[next.Add(1)-1])
			return err
		}) {
			require.NoError(t, err)
		}

		final, err := f.queries.GetCard(ctx, f.cardID)
		require.NoError(t, err)
		require.Equal(t, newTitle, final.Title, "round %d: title edit lost", round)
		require.NotNil(t, final.Priority, "round %d: priority edit lost", round)
		require.Equal(t, "high", *final.Priority)
	}
}

// An update that only touches the title must not wipe acceptance_criteria /
// implementation_note that PromoteItem copied in from a planning item.
func TestUpdateCard_AcceptanceCriteriaAndNote_PreservedWhenNotTouched(t *testing.T) {
	ctx := context.Background()
	f := newCardFixture(t)

	ac := "Given/When/Then..."
	note := "Watch for the race on X"
	_, err := f.svc.UpdateCard(ctx, service.UpdateCardParams{
		ID: f.cardID, Title: util.StringToPtr("Original"), AcceptanceCriteria: &ac, ImplementationNote: &note,
	})
	require.NoError(t, err)

	// A later edit that only changes the title, and passes nil for both —
	// PATCH semantics: nil means "don't touch".
	updated, err := f.svc.UpdateCard(ctx, service.UpdateCardParams{
		ID: f.cardID, Title: util.StringToPtr("Renamed"), AcceptanceCriteria: nil, ImplementationNote: nil,
	})
	require.NoError(t, err)

	assert.Equal(t, "Renamed", updated.Card.Title)
	require.NotNil(t, updated.Card.AcceptanceCriteria)
	assert.Equal(t, ac, *updated.Card.AcceptanceCriteria, "AC must survive an edit that never touched it")
	require.NotNil(t, updated.Card.ImplementationNote)
	assert.Equal(t, note, *updated.Card.ImplementationNote, "dev note must survive an edit that never touched it")
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

	res, err := f.svc.UpdateCard(ctx, service.UpdateCardParams{
		ID: f.cardID, Title: util.StringToPtr("Renamed only"), TagIDs: nil,
	})
	require.NoError(t, err)

	assert.Equal(t, []string{tag}, f.cardTags(ctx, t), "nil TagIDs must mean \"don't touch tags\", not \"clear them\"")
	// The handler broadcasts res.Tags; returning empty here would wipe tags on every client.
	require.Len(t, res.Tags, 1)
	assert.Equal(t, tag, res.Tags[0].ID)
}

func TestUpdateCard_EmptyTagIDs_ClearsAllTags(t *testing.T) {
	ctx := context.Background()
	f := newCardFixture(t)
	seed := testutil.NewSeed(t, f.pool)
	tag := seed.Tag(ctx, f.boardID, "bug")
	require.NoError(t, f.queries.InsertCardTag(ctx, db.InsertCardTagParams{CardID: f.cardID, TagID: tag}))

	empty := []string{}
	res, err := f.svc.UpdateCard(ctx, service.UpdateCardParams{
		ID: f.cardID, Title: util.StringToPtr("Clearing tags"), TagIDs: &empty,
	})
	require.NoError(t, err)

	assert.Empty(t, f.cardTags(ctx, t))
	assert.NotNil(t, res.Tags, "an untagged card returns [], not nil")
	assert.Empty(t, res.Tags)
}

func TestUpdateCard_ReplaceTagIDs_SwapsToExactlyTheNewSet(t *testing.T) {
	ctx := context.Background()
	f := newCardFixture(t)
	seed := testutil.NewSeed(t, f.pool)
	oldTag := seed.Tag(ctx, f.boardID, "old")
	newTag := seed.Tag(ctx, f.boardID, "new")
	require.NoError(t, f.queries.InsertCardTag(ctx, db.InsertCardTagParams{CardID: f.cardID, TagID: oldTag}))

	newSet := []string{newTag}
	res, err := f.svc.UpdateCard(ctx, service.UpdateCardParams{
		ID: f.cardID, Title: util.StringToPtr("Swapping tags"), TagIDs: &newSet,
	})
	require.NoError(t, err)

	assert.Equal(t, []string{newTag}, f.cardTags(ctx, t))
	require.Len(t, res.Tags, 1)
	assert.Equal(t, newTag, res.Tags[0].ID)
	assert.Equal(t, "new", res.Tags[0].Name)
}

// The >5 tag check runs AFTER qtx.UpdateCard applied the field change, in the same
// transaction — the whole reason UpdateCard needs one. Confirms both halves: the error
// surfaces and the earlier title change rolls back with it.
func TestUpdateCard_MoreThanFiveTags_ErrorsAndRollsBackEntireUpdate(t *testing.T) {
	ctx := context.Background()
	f := newCardFixture(t)
	seed := testutil.NewSeed(t, f.pool)

	tooMany := make([]string, 0, 6)
	for i := 0; i < 6; i++ {
		tooMany = append(tooMany, seed.Tag(ctx, f.boardID, fmt.Sprintf("tag-%d", i)))
	}

	_, err := f.svc.UpdateCard(ctx, service.UpdateCardParams{
		ID: f.cardID, Title: util.StringToPtr("Should not stick"), TagIDs: &tooMany,
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
