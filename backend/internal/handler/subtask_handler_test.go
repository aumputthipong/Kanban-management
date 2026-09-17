package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/dto"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service/mock"
)

// validCardID and chiCtx/withUserID helpers are defined in board_handler_test.go (same package)

const (
	validSubtaskID = "b2c3d4e5-f6a7-8901-bcde-f12345678901"
	otherCardID    = "c3d4e5f6-a7b8-9012-cdef-123456789012"
)

// memberBoards resolves validCardID to validBoardID and makes the caller a member.
func memberBoards() *mock.MockBoardService {
	return &mock.MockBoardService{
		GetBoardIDByCardFn: func(ctx context.Context, cardID string) (string, error) {
			if cardID != validCardID {
				return "", pgx.ErrNoRows
			}
			return validBoardID, nil
		},
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			return "member", nil
		},
	}
}

func nonMemberBoards() *mock.MockBoardService {
	boards := memberBoards()
	boards.GetBoardMemberRoleFn = func(ctx context.Context, boardID, userID string) (string, error) {
		return "", pgx.ErrNoRows
	}
	return boards
}

// subtaskService returns a service whose subtask lives on cardID and whose list read
// (used by the broadcast) succeeds.
func subtaskService(cardID string) *mock.MockSubtaskService {
	return &mock.MockSubtaskService{
		GetSubtaskByIDFn: func(ctx context.Context, subtaskID string) (db.CardSubtask, error) {
			return db.CardSubtask{ID: subtaskID, CardID: cardID, Title: "Step 1", Position: 1}, nil
		},
		GetSubtasksByCardIDFn: func(ctx context.Context, cardID string) ([]db.CardSubtask, error) {
			return []db.CardSubtask{{ID: validSubtaskID, CardID: cardID, Title: "Step 1", IsDone: true, Position: 1}}, nil
		},
	}
}

func subtaskRequest(method, body string, params ...string) *http.Request {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, "/cards/x/subtasks", reader)
	return chiCtx(withUserID(req, validUserID), params...)
}

func requireSubtasksBroadcast(t *testing.T, bc *mock.MockBroadcaster) {
	t.Helper()
	require.Len(t, bc.Sent, 1)
	assert.Equal(t, validBoardID, bc.Sent[0].BoardID)
	var msg struct {
		Type    string `json:"type"`
		Payload struct {
			CardID   string                `json:"card_id"`
			Subtasks []dto.SubtaskResponse `json:"subtasks"`
		} `json:"payload"`
	}
	require.NoError(t, json.Unmarshal(bc.Sent[0].Message, &msg))
	assert.Equal(t, "CARD_SUBTASKS_UPDATED", msg.Type)
	assert.Equal(t, validCardID, msg.Payload.CardID)
	require.Len(t, msg.Payload.Subtasks, 1, "the full list is sent, not just the changed row")
	assert.True(t, msg.Payload.Subtasks[0].IsDone)
}

// ────────────────────────────────────────────────
// CreateSubtask
// ────────────────────────────────────────────────

func TestCreateSubtask_Success_BroadcastsList(t *testing.T) {
	svc := subtaskService(validCardID)
	svc.CreateSubtaskFn = func(ctx context.Context, arg db.CreateSubtaskParams) (db.CardSubtask, error) {
		assert.Equal(t, validCardID, arg.CardID)
		assert.Equal(t, "Write tests", arg.Title)
		return db.CardSubtask{ID: validSubtaskID, CardID: validCardID, Title: "Write tests", Position: 1}, nil
	}
	bc := &mock.MockBroadcaster{}
	h := NewSubtaskHandler(svc, memberBoards(), bc)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateSubtask)(w, subtaskRequest(http.MethodPost, `{"title":"Write tests","position":1}`, "cardID", validCardID))

	assert.Equal(t, http.StatusCreated, w.Code)
	var res map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&res))
	assert.Equal(t, validSubtaskID, res["id"])
	requireSubtasksBroadcast(t, bc)
}

func TestCreateSubtask_NonMember_Returns404(t *testing.T) {
	svc := &mock.MockSubtaskService{
		CreateSubtaskFn: func(ctx context.Context, arg db.CreateSubtaskParams) (db.CardSubtask, error) {
			t.Fatal("a non-member must not reach the service")
			return db.CardSubtask{}, nil
		},
	}
	h := NewSubtaskHandler(svc, nonMemberBoards(), &mock.MockBroadcaster{})
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateSubtask)(w, subtaskRequest(http.MethodPost, `{"title":"x","position":1}`, "cardID", validCardID))

	assert.Equal(t, http.StatusNotFound, w.Code, "404 not 403: membership is not disclosed")
}

func TestCreateSubtask_UnknownCard_Returns404(t *testing.T) {
	h := NewSubtaskHandler(&mock.MockSubtaskService{}, memberBoards(), &mock.MockBroadcaster{})
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateSubtask)(w, subtaskRequest(http.MethodPost, `{"title":"x","position":1}`, "cardID", otherCardID))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreateSubtask_InvalidJSON_Returns400(t *testing.T) {
	h := NewSubtaskHandler(&mock.MockSubtaskService{}, memberBoards(), &mock.MockBroadcaster{})
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateSubtask)(w, subtaskRequest(http.MethodPost, `{bad json}`, "cardID", validCardID))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateSubtask_ServiceError_Returns500WithoutBroadcast(t *testing.T) {
	svc := &mock.MockSubtaskService{
		CreateSubtaskFn: func(ctx context.Context, arg db.CreateSubtaskParams) (db.CardSubtask, error) {
			return db.CardSubtask{}, errors.New("db error")
		},
	}
	bc := &mock.MockBroadcaster{}
	h := NewSubtaskHandler(svc, memberBoards(), bc)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateSubtask)(w, subtaskRequest(http.MethodPost, `{"title":"x","position":1}`, "cardID", validCardID))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Empty(t, bc.Sent)
}

// ────────────────────────────────────────────────
// GetSubtasks / GetSubtask
// ────────────────────────────────────────────────

func TestGetSubtasks_Success(t *testing.T) {
	h := NewSubtaskHandler(subtaskService(validCardID), memberBoards(), nil)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetSubtasks)(w, subtaskRequest(http.MethodGet, "", "cardID", validCardID))

	assert.Equal(t, http.StatusOK, w.Code)
	var res []map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&res))
	assert.Len(t, res, 1)
}

func TestGetSubtasks_InvalidCardID_Returns400(t *testing.T) {
	h := NewSubtaskHandler(&mock.MockSubtaskService{}, memberBoards(), nil)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetSubtasks)(w, subtaskRequest(http.MethodGet, "", "cardID", "not-a-uuid"))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetSubtasks_NonMember_Returns404(t *testing.T) {
	h := NewSubtaskHandler(subtaskService(validCardID), nonMemberBoards(), nil)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetSubtasks)(w, subtaskRequest(http.MethodGet, "", "cardID", validCardID))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetSubtasks_ServiceError_Returns500(t *testing.T) {
	svc := &mock.MockSubtaskService{
		GetSubtasksByCardIDFn: func(ctx context.Context, cardID string) ([]db.CardSubtask, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewSubtaskHandler(svc, memberBoards(), nil)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetSubtasks)(w, subtaskRequest(http.MethodGet, "", "cardID", validCardID))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetSubtask_Success(t *testing.T) {
	h := NewSubtaskHandler(subtaskService(validCardID), memberBoards(), nil)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetSubtask)(w, subtaskRequest(http.MethodGet, "", "cardID", validCardID, "subtaskID", validSubtaskID))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetSubtask_NotFound_Returns404(t *testing.T) {
	svc := &mock.MockSubtaskService{
		GetSubtaskByIDFn: func(ctx context.Context, subtaskID string) (db.CardSubtask, error) {
			return db.CardSubtask{}, pgx.ErrNoRows
		},
	}
	h := NewSubtaskHandler(svc, memberBoards(), nil)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetSubtask)(w, subtaskRequest(http.MethodGet, "", "cardID", validCardID, "subtaskID", validSubtaskID))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// A member of one board must not reach a subtask on another board by pairing a card
// they can see with a foreign subtask ID.
func TestGetSubtask_SubtaskOnAnotherCard_Returns404(t *testing.T) {
	h := NewSubtaskHandler(subtaskService(otherCardID), memberBoards(), nil)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetSubtask)(w, subtaskRequest(http.MethodGet, "", "cardID", validCardID, "subtaskID", validSubtaskID))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ────────────────────────────────────────────────
// UpdateSubtask
// ────────────────────────────────────────────────

func TestUpdateSubtask_Success_BroadcastsList(t *testing.T) {
	svc := subtaskService(validCardID)
	svc.UpdateSubtaskFn = func(ctx context.Context, subtaskID string, req dto.UpdateSubtaskRequest) (db.CardSubtask, error) {
		assert.Equal(t, validSubtaskID, subtaskID)
		return db.CardSubtask{ID: validSubtaskID, CardID: validCardID, Title: "Done", IsDone: true, Position: 1}, nil
	}
	bc := &mock.MockBroadcaster{}
	h := NewSubtaskHandler(svc, memberBoards(), bc)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateSubtask)(w, subtaskRequest(http.MethodPatch, `{"is_done":true}`, "cardID", validCardID, "subtaskID", validSubtaskID))

	assert.Equal(t, http.StatusOK, w.Code)
	requireSubtasksBroadcast(t, bc)
}

func TestUpdateSubtask_SubtaskOnAnotherCard_Returns404(t *testing.T) {
	svc := subtaskService(otherCardID)
	svc.UpdateSubtaskFn = func(ctx context.Context, subtaskID string, req dto.UpdateSubtaskRequest) (db.CardSubtask, error) {
		t.Fatal("a subtask on another card must not be updated")
		return db.CardSubtask{}, nil
	}
	bc := &mock.MockBroadcaster{}
	h := NewSubtaskHandler(svc, memberBoards(), bc)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateSubtask)(w, subtaskRequest(http.MethodPatch, `{"is_done":true}`, "cardID", validCardID, "subtaskID", validSubtaskID))

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Empty(t, bc.Sent)
}

func TestUpdateSubtask_NonMember_Returns404(t *testing.T) {
	h := NewSubtaskHandler(subtaskService(validCardID), nonMemberBoards(), &mock.MockBroadcaster{})
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateSubtask)(w, subtaskRequest(http.MethodPatch, `{"is_done":true}`, "cardID", validCardID, "subtaskID", validSubtaskID))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateSubtask_InvalidSubtaskID_Returns400(t *testing.T) {
	h := NewSubtaskHandler(&mock.MockSubtaskService{}, memberBoards(), nil)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateSubtask)(w, subtaskRequest(http.MethodPatch, `{"is_done":true}`, "cardID", validCardID, "subtaskID", ""))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSubtask_InvalidJSON_Returns400(t *testing.T) {
	h := NewSubtaskHandler(&mock.MockSubtaskService{}, memberBoards(), nil)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateSubtask)(w, subtaskRequest(http.MethodPatch, `{bad json}`, "cardID", validCardID, "subtaskID", validSubtaskID))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSubtask_ServiceError_Returns500(t *testing.T) {
	svc := subtaskService(validCardID)
	svc.UpdateSubtaskFn = func(ctx context.Context, subtaskID string, req dto.UpdateSubtaskRequest) (db.CardSubtask, error) {
		return db.CardSubtask{}, errors.New("db error")
	}
	h := NewSubtaskHandler(svc, memberBoards(), &mock.MockBroadcaster{})
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateSubtask)(w, subtaskRequest(http.MethodPatch, `{"is_done":true}`, "cardID", validCardID, "subtaskID", validSubtaskID))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// The write already committed, so a failed list read for the broadcast must not turn
// the response into an error; the other clients just miss this one update.
func TestUpdateSubtask_BroadcastReadFails_StillReturns200(t *testing.T) {
	svc := subtaskService(validCardID)
	svc.UpdateSubtaskFn = func(ctx context.Context, subtaskID string, req dto.UpdateSubtaskRequest) (db.CardSubtask, error) {
		return db.CardSubtask{ID: validSubtaskID, CardID: validCardID}, nil
	}
	svc.GetSubtasksByCardIDFn = func(ctx context.Context, cardID string) ([]db.CardSubtask, error) {
		return nil, errors.New("db error")
	}
	bc := &mock.MockBroadcaster{}
	h := NewSubtaskHandler(svc, memberBoards(), bc)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateSubtask)(w, subtaskRequest(http.MethodPatch, `{"is_done":true}`, "cardID", validCardID, "subtaskID", validSubtaskID))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, bc.Sent)
}

// ────────────────────────────────────────────────
// DeleteSubtask
// ────────────────────────────────────────────────

func TestDeleteSubtask_Success_BroadcastsList(t *testing.T) {
	called := false
	svc := subtaskService(validCardID)
	svc.DeleteSubtaskFn = func(ctx context.Context, subtaskID string) error {
		assert.Equal(t, validSubtaskID, subtaskID)
		called = true
		return nil
	}
	bc := &mock.MockBroadcaster{}
	h := NewSubtaskHandler(svc, memberBoards(), bc)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.DeleteSubtask)(w, subtaskRequest(http.MethodDelete, "", "cardID", validCardID, "subtaskID", validSubtaskID))

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.True(t, called)
	requireSubtasksBroadcast(t, bc)
}

func TestDeleteSubtask_SubtaskOnAnotherCard_Returns404(t *testing.T) {
	svc := subtaskService(otherCardID)
	svc.DeleteSubtaskFn = func(ctx context.Context, subtaskID string) error {
		t.Fatal("a subtask on another card must not be deleted")
		return nil
	}
	h := NewSubtaskHandler(svc, memberBoards(), &mock.MockBroadcaster{})
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.DeleteSubtask)(w, subtaskRequest(http.MethodDelete, "", "cardID", validCardID, "subtaskID", validSubtaskID))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteSubtask_ServiceError_Returns500(t *testing.T) {
	svc := subtaskService(validCardID)
	svc.DeleteSubtaskFn = func(ctx context.Context, subtaskID string) error {
		return errors.New("db error")
	}
	h := NewSubtaskHandler(svc, memberBoards(), &mock.MockBroadcaster{})
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.DeleteSubtask)(w, subtaskRequest(http.MethodDelete, "", "cardID", validCardID, "subtaskID", validSubtaskID))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
