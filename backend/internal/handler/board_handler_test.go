package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/middleware"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service/mock"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// chiCtx injects URL params into the request context for the chi router.
func chiCtx(r *http.Request, pairs ...string) *http.Request {
	rctx := chi.NewRouteContext()
	for i := 0; i+1 < len(pairs); i += 2 {
		rctx.URLParams.Add(pairs[i], pairs[i+1])
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// withUserID injects userID into the context the way the middleware does.
func withUserID(r *http.Request, userID string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), middleware.UserIDKey, userID))
}

const validBoardID = "452ae618-9e69-49f5-88a9-47728a5f17ac"
const validUserID = "550e8400-e29b-41d4-a716-446655440000"
const validColumnID = "7f3b9a2e-1c4d-4e8f-a6b0-2d5e8f1a3c7b"
const validCardID = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"

// ────────────────────────────────────────────────
// GetAllBoards
// ────────────────────────────────────────────────

func TestGetAllBoards_Success(t *testing.T) {
	svc := &mock.MockBoardService{
		GetAllBoardsFn: func(ctx context.Context, userID string) ([]service.BoardSummaryData, error) {
			return []service.BoardSummaryData{
				{ID: "id-1", Title: "Board A"},
				{ID: "id-2", Title: "Board B"},
			}, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil)
	req := withUserID(httptest.NewRequest(http.MethodGet, "/boards", nil), validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetAllBoards)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	assert.Len(t, result, 2)
	assert.Equal(t, "Board A", result[0]["title"])
}

func TestGetAllBoards_DBError(t *testing.T) {
	svc := &mock.MockBoardService{
		GetAllBoardsFn: func(ctx context.Context, userID string) ([]service.BoardSummaryData, error) {
			return nil, errors.New("connection refused")
		},
	}
	h := NewBoardHandler(svc, nil, nil)
	req := withUserID(httptest.NewRequest(http.MethodGet, "/boards", nil), validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetAllBoards)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ────────────────────────────────────────────────
// GetBoardData
// ────────────────────────────────────────────────

func TestGetBoardData_Success(t *testing.T) {
	svc := &mock.MockBoardService{
		GetBoardWithCardsFn: func(ctx context.Context, boardID string) ([]service.ColumnData, error) {
			return []service.ColumnData{
				{ID: "col-1", Title: "To Do", Category: "TODO", Cards: []service.CardData{}},
			}, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/boards/"+validBoardID, nil)
	req = chiCtx(req, "boardID", validBoardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetBoardData)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	assert.Len(t, result, 1)
	assert.Equal(t, "To Do", result[0]["title"])
}

func TestGetBoardData_InvalidBoardID(t *testing.T) {
	svc := &mock.MockBoardService{}
	h := NewBoardHandler(svc, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/boards/not-a-uuid", nil)
	req = chiCtx(req, "boardID", "not-a-uuid")
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetBoardData)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetBoardData_TouchesMembershipAccess(t *testing.T) {
	touched := make(chan struct{ boardID, userID string }, 1)
	svc := &mock.MockBoardService{
		GetBoardWithCardsFn: func(ctx context.Context, boardID string) ([]service.ColumnData, error) {
			return []service.ColumnData{}, nil
		},
		TouchBoardMemberAccessFn: func(ctx context.Context, boardID, userID string) error {
			touched <- struct{ boardID, userID string }{boardID, userID}
			return nil
		},
	}
	h := NewBoardHandler(svc, nil, nil)
	req := withUserID(httptest.NewRequest(http.MethodGet, "/boards/"+validBoardID, nil), validUserID)
	req = chiCtx(req, "boardID", validBoardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetBoardData)(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	select {
	case got := <-touched:
		assert.Equal(t, validBoardID, got.boardID)
		assert.Equal(t, validUserID, got.userID)
	case <-time.After(2 * time.Second):
		t.Fatal("expected TouchBoardMemberAccess to be invoked")
	}
}

func TestGetBoardData_TouchFailureDoesNotAffectResponse(t *testing.T) {
	called := make(chan struct{}, 1)
	svc := &mock.MockBoardService{
		GetBoardWithCardsFn: func(ctx context.Context, boardID string) ([]service.ColumnData, error) {
			return []service.ColumnData{}, nil
		},
		TouchBoardMemberAccessFn: func(ctx context.Context, boardID, userID string) error {
			called <- struct{}{}
			return errors.New("transient db error")
		},
	}
	h := NewBoardHandler(svc, nil, nil)
	req := withUserID(httptest.NewRequest(http.MethodGet, "/boards/"+validBoardID, nil), validUserID)
	req = chiCtx(req, "boardID", validBoardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetBoardData)(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	select {
	case <-called:
	case <-time.After(2 * time.Second):
		t.Fatal("touch should have been attempted")
	}
}

func TestGetBoardData_NoUserContext_SkipsTouch(t *testing.T) {
	svc := &mock.MockBoardService{
		GetBoardWithCardsFn: func(ctx context.Context, boardID string) ([]service.ColumnData, error) {
			return []service.ColumnData{}, nil
		},
		TouchBoardMemberAccessFn: func(ctx context.Context, boardID, userID string) error {
			t.Fatal("touch must not be called without userID in context")
			return nil
		},
	}
	h := NewBoardHandler(svc, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/boards/"+validBoardID, nil)
	req = chiCtx(req, "boardID", validBoardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetBoardData)(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	// Give any erroneous goroutine a chance to fire before the test ends.
	time.Sleep(50 * time.Millisecond)
}

func TestGetBoardData_ServiceError(t *testing.T) {
	svc := &mock.MockBoardService{
		GetBoardWithCardsFn: func(ctx context.Context, boardID string) ([]service.ColumnData, error) {
			return nil, errors.New("db timeout")
		},
	}
	h := NewBoardHandler(svc, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/boards/"+validBoardID, nil)
	req = chiCtx(req, "boardID", validBoardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetBoardData)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ────────────────────────────────────────────────
// CreateBoard
// ────────────────────────────────────────────────

func TestCreateBoard_Success(t *testing.T) {
	svc := &mock.MockBoardService{
		CreateBoardFn: func(ctx context.Context, title string, description, color, icon *string, ownerID string) (string, error) {
			assert.Equal(t, "My Board", title)
			assert.Equal(t, validUserID, ownerID)
			return validBoardID, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil)
	body := strings.NewReader(`{"title":"My Board"}`)
	req := httptest.NewRequest(http.MethodPost, "/boards", body)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateBoard)(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var result map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	assert.Equal(t, validBoardID, result["id"])
}

func TestCreateBoard_Unauthorized(t *testing.T) {
	svc := &mock.MockBoardService{}
	h := NewBoardHandler(svc, nil, nil)
	body := strings.NewReader(`{"title":"My Board"}`)
	req := httptest.NewRequest(http.MethodPost, "/boards", body)
	// No userID in context → unauthorized.
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateBoard)(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateBoard_InvalidJSON(t *testing.T) {
	svc := &mock.MockBoardService{}
	h := NewBoardHandler(svc, nil, nil)
	body := strings.NewReader(`{invalid json}`)
	req := httptest.NewRequest(http.MethodPost, "/boards", body)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateBoard)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ────────────────────────────────────────────────
// StashBoard
// ────────────────────────────────────────────────

func TestStashBoard_Success(t *testing.T) {
	called := false
	svc := &mock.MockBoardService{
		StashBoardFn: func(ctx context.Context, boardID string) error {
			assert.Equal(t, validBoardID, boardID)
			called = true
			return nil
		},
	}
	h := NewBoardHandler(svc, nil, nil)
	req := httptest.NewRequest(http.MethodDelete, "/boards/"+validBoardID, nil)
	req = chiCtx(req, "boardID", validBoardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.StashBoard)(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.True(t, called)
}

func TestStashBoard_ServiceError(t *testing.T) {
	svc := &mock.MockBoardService{
		StashBoardFn: func(ctx context.Context, boardID string) error {
			return errors.New("board not found")
		},
	}
	h := NewBoardHandler(svc, nil, nil)
	req := httptest.NewRequest(http.MethodDelete, "/boards/"+validBoardID, nil)
	req = chiCtx(req, "boardID", validBoardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.StashBoard)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ────────────────────────────────────────────────
// HardDelete
// ────────────────────────────────────────────────

func TestHardDelete_Success(t *testing.T) {
	svc := &mock.MockBoardService{
		HardDeleteBoardFn: func(ctx context.Context, id string) error {
			return nil
		},
	}
	h := NewBoardHandler(svc, nil, nil)
	req := httptest.NewRequest(http.MethodDelete, "/boards/"+validBoardID+"/hard", nil)
	req = chiCtx(req, "boardID", validBoardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.HardDelete)(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// ────────────────────────────────────────────────
// GetBoardMembers
// ────────────────────────────────────────────────

func TestGetBoardMembers_Success(t *testing.T) {
	svc := &mock.MockBoardService{
		GetBoardMembersFn: func(ctx context.Context, boardID string) ([]db.GetBoardMembersRow, error) {
			return []db.GetBoardMembersRow{
				{ID: "m-1", Role: "owner", UserID: validUserID, Email: "a@example.com", FullName: "Alice"},
			}, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/boards/"+validBoardID+"/members", nil)
	req = chiCtx(req, "boardID", validBoardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetBoardMembers)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	assert.Len(t, result, 1)
	assert.Equal(t, "Alice", result[0]["full_name"])
}

func TestGetBoardMembers_DBError(t *testing.T) {
	svc := &mock.MockBoardService{
		GetBoardMembersFn: func(ctx context.Context, boardID string) ([]db.GetBoardMembersRow, error) {
			return nil, errors.New("query failed")
		},
	}
	h := NewBoardHandler(svc, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/boards/"+validBoardID+"/members", nil)
	req = chiCtx(req, "boardID", validBoardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetBoardMembers)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ────────────────────────────────────────────────
// UpdateBoard
// ────────────────────────────────────────────────

func TestUpdateBoard_Success(t *testing.T) {
	svc := &mock.MockBoardService{
		UpdateBoardFn: func(ctx context.Context, id string, title *string, budget *float64, description, color, icon *string) (db.Board, error) {
			assert.Equal(t, validBoardID, id)
			return db.Board{ID: validBoardID, Title: "New Title"}, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil)
	body := strings.NewReader(`{"title":"New Title"}`)
	req := httptest.NewRequest(http.MethodPatch, "/boards/"+validBoardID, body)
	req = chiCtx(req, "boardID", validBoardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateBoard)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var res map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&res))
	assert.Equal(t, validBoardID, res["id"])
}

func TestUpdateBoard_InvalidBoardID(t *testing.T) {
	svc := &mock.MockBoardService{}
	h := NewBoardHandler(svc, nil, nil)
	body := strings.NewReader(`{"title":"New Title"}`)
	req := httptest.NewRequest(http.MethodPatch, "/boards/not-a-uuid", body)
	req = chiCtx(req, "boardID", "not-a-uuid")
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateBoard)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateBoard_InvalidJSON(t *testing.T) {
	svc := &mock.MockBoardService{}
	h := NewBoardHandler(svc, nil, nil)
	body := strings.NewReader(`{bad json}`)
	req := httptest.NewRequest(http.MethodPatch, "/boards/"+validBoardID, body)
	req = chiCtx(req, "boardID", validBoardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateBoard)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateBoard_ServiceError(t *testing.T) {
	svc := &mock.MockBoardService{
		UpdateBoardFn: func(ctx context.Context, id string, title *string, budget *float64, description, color, icon *string) (db.Board, error) {
			return db.Board{}, errors.New("db error")
		},
	}
	h := NewBoardHandler(svc, nil, nil)
	body := strings.NewReader(`{"title":"New Title"}`)
	req := httptest.NewRequest(http.MethodPatch, "/boards/"+validBoardID, body)
	req = chiCtx(req, "boardID", validBoardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateBoard)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ────────────────────────────────────────────────
// GetStashedBoards
// ────────────────────────────────────────────────

func TestGetStashedBoards_Success(t *testing.T) {
	svc := &mock.MockBoardService{
		GetStashedBoardsFn: func(ctx context.Context, userID string) ([]db.GetStashedBoardsForOwnerRow, error) {
			return []db.GetStashedBoardsForOwnerRow{
				{ID: "b-1", Title: "Old Board"},
			}, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil)
	req := withUserID(httptest.NewRequest(http.MethodGet, "/stash", nil), validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetStashedBoards)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var res []map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&res))
	assert.Len(t, res, 1)
	assert.Equal(t, "b-1", res[0]["id"])
}

func TestGetStashedBoards_ServiceError(t *testing.T) {
	svc := &mock.MockBoardService{
		GetStashedBoardsFn: func(ctx context.Context, userID string) ([]db.GetStashedBoardsForOwnerRow, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewBoardHandler(svc, nil, nil)
	req := withUserID(httptest.NewRequest(http.MethodGet, "/stash", nil), validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetStashedBoards)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ────────────────────────────────────────────────
// CreateCard
// ────────────────────────────────────────────────

func TestCreateCard_Success(t *testing.T) {
	svc := &mock.MockBoardService{
		GetBoardIDByColumnFn: func(ctx context.Context, columnID string) (string, error) {
			return validBoardID, nil
		},
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			return "member", nil
		},
		CreateCardFn: func(ctx context.Context, arg db.CreateCardParams) (db.CreateCardRow, error) {
			assert.Equal(t, validColumnID, arg.ColumnID)
			assert.Equal(t, "Test Card", arg.Title)
			return db.CreateCardRow{ID: validCardID, ColumnID: validColumnID, Title: "Test Card"}, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil)
	body := strings.NewReader(`{"column_id":"` + validColumnID + `","title":"Test Card"}`)
	req := httptest.NewRequest(http.MethodPost, "/cards", body)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateCard)(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var res map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&res))
	assert.Equal(t, validCardID, res["ID"])
}

func TestCreateCard_Unauthorized(t *testing.T) {
	svc := &mock.MockBoardService{}
	h := NewBoardHandler(svc, nil, nil)
	body := strings.NewReader(`{"column_id":"` + validColumnID + `","title":"Test Card"}`)
	req := httptest.NewRequest(http.MethodPost, "/cards", body)
	// No userID in context.
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateCard)(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateCard_InvalidColumnID(t *testing.T) {
	svc := &mock.MockBoardService{}
	h := NewBoardHandler(svc, nil, nil)
	body := strings.NewReader(`{"column_id":"not-a-uuid","title":"Test Card"}`)
	req := httptest.NewRequest(http.MethodPost, "/cards", body)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateCard)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateCard_InvalidJSON(t *testing.T) {
	svc := &mock.MockBoardService{}
	h := NewBoardHandler(svc, nil, nil)
	body := strings.NewReader(`{bad json}`)
	req := httptest.NewRequest(http.MethodPost, "/cards", body)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateCard)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateCard_ServiceError(t *testing.T) {
	svc := &mock.MockBoardService{
		GetBoardIDByColumnFn: func(ctx context.Context, columnID string) (string, error) {
			return validBoardID, nil
		},
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			return "member", nil
		},
		CreateCardFn: func(ctx context.Context, arg db.CreateCardParams) (db.CreateCardRow, error) {
			return db.CreateCardRow{}, errors.New("db error")
		},
	}
	h := NewBoardHandler(svc, nil, nil)
	body := strings.NewReader(`{"column_id":"` + validColumnID + `","title":"Test Card"}`)
	req := httptest.NewRequest(http.MethodPost, "/cards", body)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateCard)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ────────────────────────────────────────────────
// UpdateCard
// ────────────────────────────────────────────────

func TestUpdateCard_Success(t *testing.T) {
	userID := validUserID
	svc := &mock.MockBoardService{
		GetCardFn: func(ctx context.Context, cardID string) (db.Card, error) {
			return db.Card{ID: validCardID, ColumnID: validColumnID, CreatedBy: &userID}, nil
		},
		GetBoardIDByColumnFn: func(ctx context.Context, columnID string) (string, error) {
			return validBoardID, nil
		},
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, uid string) (string, error) {
			return "member", nil
		},
		UpdateCardFn: func(ctx context.Context, arg service.UpdateCardParams) (db.Card, error) {
			assert.Equal(t, validCardID, arg.ID)
			return db.Card{ID: validCardID, ColumnID: validColumnID, Title: "Updated"}, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil)
	body := strings.NewReader(`{"title":"Updated"}`)
	req := httptest.NewRequest(http.MethodPatch, "/cards/"+validCardID, body)
	req = chiCtx(req, "cardID", validCardID)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateCard)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var res map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&res))
	assert.Equal(t, validCardID, res["ID"])
}

func TestUpdateCard_InvalidCardID(t *testing.T) {
	svc := &mock.MockBoardService{}
	h := NewBoardHandler(svc, nil, nil)
	body := strings.NewReader(`{"title":"Updated"}`)
	req := httptest.NewRequest(http.MethodPatch, "/cards/not-a-uuid", body)
	req = chiCtx(req, "cardID", "not-a-uuid")
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateCard)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateCard_InvalidJSON(t *testing.T) {
	svc := &mock.MockBoardService{}
	h := NewBoardHandler(svc, nil, nil)
	body := strings.NewReader(`{bad json}`)
	req := httptest.NewRequest(http.MethodPatch, "/cards/"+validCardID, body)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateCard)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateCard_ServiceError(t *testing.T) {
	userID := validUserID
	svc := &mock.MockBoardService{
		GetCardFn: func(ctx context.Context, cardID string) (db.Card, error) {
			return db.Card{ID: validCardID, ColumnID: validColumnID, CreatedBy: &userID}, nil
		},
		GetBoardIDByColumnFn: func(ctx context.Context, columnID string) (string, error) {
			return validBoardID, nil
		},
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, uid string) (string, error) {
			return "member", nil
		},
		UpdateCardFn: func(ctx context.Context, arg service.UpdateCardParams) (db.Card, error) {
			return db.Card{}, errors.New("db error")
		},
	}
	h := NewBoardHandler(svc, nil, nil)
	body := strings.NewReader(`{"title":"Updated"}`)
	req := httptest.NewRequest(http.MethodPatch, "/cards/"+validCardID, body)
	req = chiCtx(req, "cardID", validCardID)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateCard)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ────────────────────────────────────────────────
// GetCard
// ────────────────────────────────────────────────

func TestGetCard_Success(t *testing.T) {
	svc := &mock.MockBoardService{
		GetCardDetailFn: func(ctx context.Context, cardID string) (service.CardDetailData, error) {
			assert.Equal(t, validCardID, cardID)
			return service.CardDetailData{
				Card: db.Card{ID: validCardID, ColumnID: validColumnID, Title: "My Card"},
			}, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/cards/"+validCardID, nil)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetCard)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var res map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&res))
	assert.Equal(t, validCardID, res["id"])
}

func TestGetCard_NotFound(t *testing.T) {
	svc := &mock.MockBoardService{
		GetCardDetailFn: func(ctx context.Context, cardID string) (service.CardDetailData, error) {
			return service.CardDetailData{}, sql.ErrNoRows
		},
	}
	h := NewBoardHandler(svc, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/cards/"+validCardID, nil)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetCard)(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetCard_InvalidCardID(t *testing.T) {
	svc := &mock.MockBoardService{}
	h := NewBoardHandler(svc, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/cards/not-a-uuid", nil)
	req = chiCtx(req, "cardID", "not-a-uuid")
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetCard)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetCard_ServiceError(t *testing.T) {
	svc := &mock.MockBoardService{
		GetCardDetailFn: func(ctx context.Context, cardID string) (service.CardDetailData, error) {
			return service.CardDetailData{}, errors.New("db error")
		},
	}
	h := NewBoardHandler(svc, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/cards/"+validCardID, nil)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetCard)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
