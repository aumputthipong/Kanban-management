//go:build integration

// Integration tests for TagService. Validation is pure Go and already
// covered at the handler layer (mocked) — these focus on what only a real
// Postgres can show: the DB-scoped delete and the UNIQUE(board_id, name)
// constraint's interaction with the length check.
package service_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/testutil"
)

type tagFixture struct {
	queries *db.Queries
	svc     *service.TagService
	boardID string
}

func newTagFixture(t *testing.T) *tagFixture {
	t.Helper()
	ctx := context.Background()
	pool := testutil.NewTestDB(t)
	queries := db.New(pool)
	seed := testutil.NewSeed(t, pool)
	return &tagFixture{
		queries: queries,
		svc:     service.NewTagService(pool, queries),
		boardID: seed.Board(ctx, seed.User(ctx)),
	}
}

func TestCreateTag_Success(t *testing.T) {
	ctx := context.Background()
	f := newTagFixture(t)

	tag, err := f.svc.CreateTag(ctx, f.boardID, "bug", "red")
	require.NoError(t, err)
	assert.Equal(t, "bug", tag.Name)
}

func TestCreateTag_Whitespace_ErrTagNameEmpty(t *testing.T) {
	ctx := context.Background()
	f := newTagFixture(t)

	_, err := f.svc.CreateTag(ctx, f.boardID, "   ", "red")
	assert.ErrorIs(t, err, service.ErrTagNameEmpty)
}

func TestCreateTag_51AsciiChars_ErrTagNameTooLong(t *testing.T) {
	ctx := context.Background()
	f := newTagFixture(t)

	_, err := f.svc.CreateTag(ctx, f.boardID, strings.Repeat("a", 51), "red")
	assert.ErrorIs(t, err, service.ErrTagNameTooLong)
}

// Documents a real bug (not fixed here): CreateTag checks len(name), which
// counts BYTES, against a 50-char limit meant for the user-facing character
// count. 20 Thai characters is well under any reasonable "50 characters"
// limit, but each is 3 bytes — 60 bytes — so it is wrongly rejected as too
// long on a Thai-first UI. A correct check would use
// utf8.RuneCountInString(name) instead of len(name).
func TestCreateTag_20ThaiChars_WronglyRejectedAsTooLong(t *testing.T) {
	ctx := context.Background()
	f := newTagFixture(t)
	name := strings.Repeat("ก", 20) // 20 runes, 60 bytes

	_, err := f.svc.CreateTag(ctx, f.boardID, name, "red")
	assert.ErrorIs(t, err, service.ErrTagNameTooLong,
		"current (buggy) behaviour: 20 Thai characters trips the 50-char limit because len() counts 60 bytes, not 20 runes")
}

// DeleteTag's query scopes the DELETE by board_id, so naming a tag id that
// belongs to a different board must be a no-op rather than letting a member
// of board B delete a tag that only board A owns.
func TestDeleteTag_WrongBoardID_DoesNotDelete(t *testing.T) {
	ctx := context.Background()
	f := newTagFixture(t)
	tag, err := f.svc.CreateTag(ctx, f.boardID, "bug", "red")
	require.NoError(t, err)
	otherBoard, err := f.queries.CreateBoard(ctx, db.CreateBoardParams{Title: "Other Board"})
	require.NoError(t, err)

	require.NoError(t, f.svc.DeleteTag(ctx, otherBoard.ID, tag.ID))

	tags, err := f.svc.GetTagsByBoard(ctx, f.boardID)
	require.NoError(t, err)
	assert.Len(t, tags, 1, "the tag must survive a delete scoped to a different board")
}

func TestDeleteTag_CorrectBoard_Deletes(t *testing.T) {
	ctx := context.Background()
	f := newTagFixture(t)
	tag, err := f.svc.CreateTag(ctx, f.boardID, "bug", "red")
	require.NoError(t, err)

	require.NoError(t, f.svc.DeleteTag(ctx, f.boardID, tag.ID))

	tags, err := f.svc.GetTagsByBoard(ctx, f.boardID)
	require.NoError(t, err)
	assert.Empty(t, tags)
}
