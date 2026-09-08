package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service/mock"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const otherColumnID = "3a7c8e11-90d2-4b6f-8c25-71ff4d9e0a68"

// memberBoardService resolves validCardID onto validBoardID and answers the
// membership gate. Tests override individual fields to model the failure cases.
func memberBoardService() *mock.MockBoardService {
	return &mock.MockBoardService{
		GetCardFn: func(ctx context.Context, cardID string) (db.Card, error) {
			return db.Card{ID: cardID, ColumnID: validColumnID}, nil
		},
		GetBoardIDByColumnFn: func(ctx context.Context, columnID string) (string, error) {
			return validBoardID, nil
		},
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			return "member", nil
		},
	}
}

func newCmdHandler(cmd *mock.MockBoardCommandService, boards *mock.MockBoardService) (*BoardCommandHandler, *mock.MockBroadcaster) {
	bc := &mock.MockBroadcaster{}
	return NewBoardCommandHandler(cmd, boards, nil, bc), bc
}

func patchReq(t *testing.T, target, body string, params ...string) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodPatch, target, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return withUserID(chiCtx(r, params...), validUserID)
}

// decodeBroadcast returns the type tag and payload of the single message sent.
func decodeBroadcast(t *testing.T, bc *mock.MockBroadcaster) (string, map[string]any) {
	t.Helper()
	require.Len(t, bc.Sent, 1, "expected exactly one broadcast")
	var msg struct {
		Type    string         `json:"type"`
		Payload map[string]any `json:"payload"`
	}
	require.NoError(t, json.Unmarshal(bc.Sent[0].Message, &msg))
	return msg.Type, msg.Payload
}

// ────────────────────────────────────────────────
// MoveCard
// ────────────────────────────────────────────────

func TestMoveCard_Success_BroadcastsCardMoved(t *testing.T) {
	cmd := &mock.MockBoardCommandService{
		VerifyColumnInBoardFn: func(ctx context.Context, columnID, boardID string) error { return nil },
		MoveCardFn: func(ctx context.Context, cardID, newColumnID string, position float64) (service.MoveCardResult, error) {
			return service.MoveCardResult{CardTitle: "Ship it", IsDone: true}, nil
		},
	}
	h, bc := newCmdHandler(cmd, memberBoardService())

	req := patchReq(t, "/api/cards/"+validCardID+"/move",
		`{"column_id":"`+otherColumnID+`","position":128}`, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.MoveCard)(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	msgType, payload := decodeBroadcast(t, bc)
	assert.Equal(t, "CARD_MOVED", msgType)
	assert.Equal(t, validCardID, payload["card_id"])
	assert.Equal(t, otherColumnID, payload["new_column_id"])
	// is_done is derived from the target column, never taken from the request body.
	assert.Equal(t, true, payload["is_done"])
	assert.Equal(t, validBoardID, bc.Sent[0].BoardID)
}

// A member of board A must not be able to move a card into board B by naming one of
// its columns — the same anti-enumeration 404 as a missing column.
func TestMoveCard_ColumnOnAnotherBoard_Returns404(t *testing.T) {
	cmd := &mock.MockBoardCommandService{
		VerifyColumnInBoardFn: func(ctx context.Context, columnID, boardID string) error {
			return service.ErrEntityBoardMismatch
		},
		MoveCardFn: func(ctx context.Context, cardID, newColumnID string, position float64) (service.MoveCardResult, error) {
			t.Fatal("must not move a card into a column on another board")
			return service.MoveCardResult{}, nil
		},
	}
	h, bc := newCmdHandler(cmd, memberBoardService())

	req := patchReq(t, "/api/cards/"+validCardID+"/move",
		`{"column_id":"`+otherColumnID+`","position":128}`, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.MoveCard)(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Empty(t, bc.Sent)
}

func TestMoveCard_NonMember_Returns404(t *testing.T) {
	boards := memberBoardService()
	boards.GetBoardMemberRoleFn = func(ctx context.Context, boardID, userID string) (string, error) {
		return "", pgx.ErrNoRows
	}
	cmd := &mock.MockBoardCommandService{
		MoveCardFn: func(ctx context.Context, cardID, newColumnID string, position float64) (service.MoveCardResult, error) {
			t.Fatal("must not move a card for a non-member")
			return service.MoveCardResult{}, nil
		},
	}
	h, bc := newCmdHandler(cmd, boards)

	req := patchReq(t, "/api/cards/"+validCardID+"/move",
		`{"column_id":"`+otherColumnID+`","position":128}`, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.MoveCard)(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Empty(t, bc.Sent)
}

func TestMoveCard_MalformedCardID_Returns400(t *testing.T) {
	h, bc := newCmdHandler(&mock.MockBoardCommandService{}, memberBoardService())

	req := patchReq(t, "/api/cards/not-a-uuid/move",
		`{"column_id":"`+otherColumnID+`","position":128}`, "cardID", "not-a-uuid")
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.MoveCard)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, bc.Sent)
}

// ────────────────────────────────────────────────
// DeleteCard
// ────────────────────────────────────────────────

func TestDeleteCard_Success_BroadcastsCardDeleted(t *testing.T) {
	cmd := &mock.MockBoardCommandService{
		DeleteCardFn: func(ctx context.Context, cardID string) (string, error) { return "Gone", nil },
	}
	h, bc := newCmdHandler(cmd, memberBoardService())

	r := httptest.NewRequest(http.MethodDelete, "/api/cards/"+validCardID, nil)
	req := withUserID(chiCtx(r, "cardID", validCardID), validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.DeleteCard)(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	msgType, payload := decodeBroadcast(t, bc)
	assert.Equal(t, "CARD_DELETED", msgType)
	assert.Equal(t, validCardID, payload["card_id"])
}

// ────────────────────────────────────────────────
// ToggleCardDone
// ────────────────────────────────────────────────

func TestToggleCardDone_Success_BroadcastsCardMoved(t *testing.T) {
	cmd := &mock.MockBoardCommandService{
		ToggleCardDoneFn: func(ctx context.Context, cardID, boardID string, isDone bool) (service.ToggleCardDoneResult, error) {
			assert.True(t, isDone)
			return service.ToggleCardDoneResult{TargetColumnID: otherColumnID, CardTitle: "Done thing"}, nil
		},
	}
	h, bc := newCmdHandler(cmd, memberBoardService())

	req := patchReq(t, "/api/cards/"+validCardID+"/done", `{"is_done":true}`, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.ToggleCardDone)(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	// Broadcast as CARD_MOVED so the client reuses one handler for both paths.
	msgType, payload := decodeBroadcast(t, bc)
	assert.Equal(t, "CARD_MOVED", msgType)
	assert.Equal(t, otherColumnID, payload["new_column_id"])
	assert.Equal(t, true, payload["is_done"])
}

// is_done is a pointer so an omitted field is a 400, not a silent "mark it undone".
func TestToggleCardDone_MissingIsDone_Returns400(t *testing.T) {
	cmd := &mock.MockBoardCommandService{
		ToggleCardDoneFn: func(ctx context.Context, cardID, boardID string, isDone bool) (service.ToggleCardDoneResult, error) {
			t.Fatal("must not toggle without an explicit is_done")
			return service.ToggleCardDoneResult{}, nil
		},
	}
	h, bc := newCmdHandler(cmd, memberBoardService())

	req := patchReq(t, "/api/cards/"+validCardID+"/done", `{}`, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.ToggleCardDone)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, bc.Sent)
}

// ────────────────────────────────────────────────
// Columns
// ────────────────────────────────────────────────

func TestCreateColumn_Success_BroadcastsColumnCreated(t *testing.T) {
	cmd := &mock.MockBoardCommandService{
		CreateColumnFn: func(ctx context.Context, boardID, title, category string, color *string) (db.CreateColumnRow, error) {
			return db.CreateColumnRow{ID: validColumnID, Title: title, Category: category, Position: 3}, nil
		},
	}
	h, bc := newCmdHandler(cmd, memberBoardService())

	r := httptest.NewRequest(http.MethodPost, "/api/boards/"+validBoardID+"/columns",
		strings.NewReader(`{"title":"Review","category":"TODO"}`))
	r.Header.Set("Content-Type", "application/json")
	req := withUserID(chiCtx(r, "boardID", validBoardID), validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateColumn)(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	msgType, payload := decodeBroadcast(t, bc)
	assert.Equal(t, "COLUMN_CREATED", msgType)
	assert.Equal(t, "Review", payload["title"])
	assert.Contains(t, w.Body.String(), "Review")
}

// category is a closed set on the client, so an unknown value must not reach the DB.
func TestCreateColumn_UnknownCategory_Returns400(t *testing.T) {
	cmd := &mock.MockBoardCommandService{
		CreateColumnFn: func(ctx context.Context, boardID, title, category string, color *string) (db.CreateColumnRow, error) {
			t.Fatal("must not create a column with an unknown category")
			return db.CreateColumnRow{}, nil
		},
	}
	h, bc := newCmdHandler(cmd, memberBoardService())

	r := httptest.NewRequest(http.MethodPost, "/api/boards/"+validBoardID+"/columns",
		strings.NewReader(`{"title":"Review","category":"ARCHIVED"}`))
	r.Header.Set("Content-Type", "application/json")
	req := withUserID(chiCtx(r, "boardID", validBoardID), validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateColumn)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, bc.Sent)
}

func TestUpdateColumn_Success_BroadcastsColumnUpdated(t *testing.T) {
	cmd := &mock.MockBoardCommandService{
		UpdateColumnFn: func(ctx context.Context, p service.UpdateColumnParams) error {
			assert.Equal(t, validColumnID, p.ID)
			assert.Equal(t, "Renamed", p.Title)
			return nil
		},
	}
	h, bc := newCmdHandler(cmd, memberBoardService())

	req := patchReq(t, "/api/columns/"+validColumnID,
		`{"title":"Renamed","category":"TODO"}`, "columnID", validColumnID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateColumn)(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	msgType, payload := decodeBroadcast(t, bc)
	assert.Equal(t, "COLUMN_UPDATED", msgType)
	assert.Equal(t, "Renamed", payload["title"])
}

func TestDeleteColumn_NonMember_Returns404(t *testing.T) {
	boards := memberBoardService()
	boards.GetBoardMemberRoleFn = func(ctx context.Context, boardID, userID string) (string, error) {
		return "", pgx.ErrNoRows
	}
	cmd := &mock.MockBoardCommandService{
		DeleteColumnFn: func(ctx context.Context, columnID string) error {
			t.Fatal("must not delete a column for a non-member")
			return nil
		},
	}
	h, bc := newCmdHandler(cmd, boards)

	r := httptest.NewRequest(http.MethodDelete, "/api/columns/"+validColumnID, nil)
	req := withUserID(chiCtx(r, "columnID", validColumnID), validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.DeleteColumn)(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Empty(t, bc.Sent)
}
