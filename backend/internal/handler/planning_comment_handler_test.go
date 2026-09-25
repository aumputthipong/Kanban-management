package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service/mock"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validCommentID = "33333333-4444-5555-6666-777777777777"

func commentRig() (*mock.MockPlanningService, *mock.MockBoardService, *mock.MockActivityRecorder, *PlanningHandler) {
	plan, boards, act, h := newPromoteTestRig()
	plan.GetItemBoardIDFn = func(ctx context.Context, itemID string) (string, error) {
		return validBoardID, nil
	}
	return plan, boards, act, h
}

func TestCreateComment_HappyPath_RespondsWithRowAndLogsActivity(t *testing.T) {
	plan, boards, act, h := commentRig()
	plan.CreateCommentFn = func(ctx context.Context, itemID, authorID, body string) (db.PlanningItemComment, error) {
		assert.Equal(t, validPlanningItemID, itemID)
		assert.Equal(t, validUserID, authorID)
		now := time.Now()
		return db.PlanningItemComment{
			ID:        validCommentID,
			ItemID:    itemID,
			AuthorID:  authorID,
			Body:      body,
			CreatedAt: now,
			UpdatedAt: now,
		}, nil
	}
	boards.GetAllUsersFn = func(ctx context.Context) ([]db.GetAllUsersRow, error) {
		return []db.GetAllUsersRow{{ID: validUserID, FullName: "Test User"}}, nil
	}

	body := strings.NewReader(`{"body": "first thought"}`)
	req := httptest.NewRequest(http.MethodPost, "/planning/items/"+validPlanningItemID+"/comments", body)
	req = chiCtx(req, "itemID", validPlanningItemID)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()
	httputil.MakeHandler(h.CreateComment)(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), `"Test User"`)
	require.Len(t, act.Calls, 1)
	assert.Equal(t, service.EventPlanningCommentCreated, act.Calls[0].EventType)
}

func TestEditComment_NotOwn_Returns404(t *testing.T) {
	// Anti-enumeration: someone else's comment looks like a missing one.
	plan, _, act, h := commentRig()
	plan.GetCommentFn = func(ctx context.Context, commentID string) (db.PlanningItemComment, error) {
		return db.PlanningItemComment{
			ID:       commentID,
			AuthorID: otherUserID,
			ItemID:   validPlanningItemID,
		}, nil
	}
	plan.GetCommentBoardIDFn = func(ctx context.Context, commentID string) (string, error) {
		return validBoardID, nil
	}

	body := strings.NewReader(`{"body": "trying to edit"}`)
	req := httptest.NewRequest(http.MethodPatch, "/planning/comments/"+validCommentID, body)
	req = chiCtx(req, "commentID", validCommentID)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()
	httputil.MakeHandler(h.EditComment)(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Empty(t, act.Calls)
}

func TestDeleteComment_OwnComment_SoftDeletesAndLogs(t *testing.T) {
	plan, _, act, h := commentRig()
	plan.GetCommentFn = func(ctx context.Context, commentID string) (db.PlanningItemComment, error) {
		return db.PlanningItemComment{
			ID:       commentID,
			AuthorID: validUserID,
			ItemID:   validPlanningItemID,
		}, nil
	}
	plan.GetCommentBoardIDFn = func(ctx context.Context, commentID string) (string, error) {
		return validBoardID, nil
	}
	var deleted bool
	plan.DeleteCommentFn = func(ctx context.Context, commentID string) error {
		deleted = true
		assert.Equal(t, validCommentID, commentID)
		return nil
	}

	req := httptest.NewRequest(http.MethodDelete, "/planning/comments/"+validCommentID, nil)
	req = chiCtx(req, "commentID", validCommentID)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()
	httputil.MakeHandler(h.DeleteComment)(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.True(t, deleted)
	require.Len(t, act.Calls, 1)
	assert.Equal(t, service.EventPlanningCommentDeleted, act.Calls[0].EventType)
}

func TestDeleteComment_NotOwnAsRegularMember_Returns404(t *testing.T) {
	plan, boards, act, h := commentRig()
	boards.GetBoardMemberRoleFn = func(ctx context.Context, boardID, userID string) (string, error) {
		return "member", nil
	}
	plan.GetCommentFn = func(ctx context.Context, commentID string) (db.PlanningItemComment, error) {
		return db.PlanningItemComment{
			ID:       commentID,
			AuthorID: otherUserID,
			ItemID:   validPlanningItemID,
		}, nil
	}
	plan.GetCommentBoardIDFn = func(ctx context.Context, commentID string) (string, error) {
		return validBoardID, nil
	}

	req := httptest.NewRequest(http.MethodDelete, "/planning/comments/"+validCommentID, nil)
	req = chiCtx(req, "commentID", validCommentID)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()
	httputil.MakeHandler(h.DeleteComment)(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Empty(t, act.Calls)
}

func TestDeleteComment_NotOwnAsManager_Succeeds(t *testing.T) {
	plan, boards, _, h := commentRig()
	boards.GetBoardMemberRoleFn = func(ctx context.Context, boardID, userID string) (string, error) {
		return "manager", nil
	}
	plan.GetCommentFn = func(ctx context.Context, commentID string) (db.PlanningItemComment, error) {
		return db.PlanningItemComment{
			ID:       commentID,
			AuthorID: otherUserID,
			ItemID:   validPlanningItemID,
		}, nil
	}
	plan.GetCommentBoardIDFn = func(ctx context.Context, commentID string) (string, error) {
		return validBoardID, nil
	}
	var deleted bool
	plan.DeleteCommentFn = func(ctx context.Context, commentID string) error {
		deleted = true
		return nil
	}

	req := httptest.NewRequest(http.MethodDelete, "/planning/comments/"+validCommentID, nil)
	req = chiCtx(req, "commentID", validCommentID)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()
	httputil.MakeHandler(h.DeleteComment)(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.True(t, deleted, "manager should be allowed to force-delete")
}

func TestListComments_NonMember_Returns404(t *testing.T) {
	plan, boards, _, h := commentRig()
	plan.GetItemBoardIDFn = func(ctx context.Context, itemID string) (string, error) {
		return validBoardID, nil
	}
	boards.GetBoardMemberRoleFn = func(ctx context.Context, boardID, userID string) (string, error) {
		return "", pgx.ErrNoRows
	}

	req := httptest.NewRequest(http.MethodGet, "/planning/items/"+validPlanningItemID+"/comments", nil)
	req = chiCtx(req, "itemID", validPlanningItemID)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()
	httputil.MakeHandler(h.ListComments)(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
