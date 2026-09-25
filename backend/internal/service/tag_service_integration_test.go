//go:build integration

package service_test

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

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

// 20 Thai characters are 60 bytes — the limit must count characters.
func TestCreateTag_ThaiName_LimitCountsCharactersNotBytes(t *testing.T) {
	ctx := context.Background()
	f := newTagFixture(t)

	tag, err := f.svc.CreateTag(ctx, f.boardID, strings.Repeat("ก", 50), "red")
	require.NoError(t, err, "50 Thai characters is exactly the limit (150 bytes)")
	assert.Equal(t, 50, utf8.RuneCountInString(tag.Name))

	_, err = f.svc.CreateTag(ctx, f.boardID, strings.Repeat("ข", 51), "red")
	assert.ErrorIs(t, err, service.ErrTagNameTooLong)
}

// Scoped by board_id, so a foreign tag id is a no-op.
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
