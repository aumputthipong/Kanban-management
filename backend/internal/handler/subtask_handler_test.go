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
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service/mock"
)

// validCardID and chiCtx/withUserID helpers are defined in board_handler_test.go (same package)

const (
	validSubtaskID = "b2c3d4e5-f6a7-8901-bcde-f12345678901"
	otherCardID    = "c3d4e5f6-a7b8-9012-cdef-123456789012"
)

// boardsFor resolves validCardID (in validColumnID on validBoardID) and gives the caller
// role on that board; role "" makes the caller a non-member.
func boardsFor(role string, card db.Card) *mock.MockBoardService {
	return &mock.MockBoardService{
		GetCardFn: func(ctx context.Context, cardID string) (db.Card, error) {
			if cardID != validCardID {
				return db.Card{}, pgx.ErrNoRows
			}
			return card, nil
		},
		GetBoardIDByColumnFn: func(ctx context.Context, columnID string) (string, error) {
			return validBoardID, nil
		},
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			if role == "" {
				return "", pgx.ErrNoRows
			}
			return role, nil
		},
	}
}

// memberBoards: the caller is a plain member assigned to the card, so they may edit it.
func memberBoards() *mock.MockBoardService {
	return boardsFor("member", cardOwnedBy(ptr(otherUserID), ptr(validUserID)))
}

// bystanderBoards: the caller is a plain member on someone else's card.
func bystanderBoards() *mock.MockBoardService {
	return boardsFor("member", cardOwnedBy(ptr(otherUserID), ptr(otherUserID)))
}

func nonMemberBoards() *mock.MockBoardService {
	return boardsFor("", cardOwnedBy(ptr(otherUserID), ptr(validUserID)))
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
	svc.CreateSubtaskFn = func(ctx context.Context, cardID, title string) (db.CardSubtask, error) {
		assert.Equal(t, validCardID, cardID)
		assert.Equal(t, "Write tests", title)
		return db.CardSubtask{ID: validSubtaskID, CardID: validCardID, Title: "Write tests", Position: 1}, nil
	}
	bc := &mock.MockBroadcaster{}
	h := NewSubtaskHandler(svc, memberBoards(), nil, bc)
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
		CreateSubtaskFn: func(ctx context.Context, cardID, title string) (db.CardSubtask, error) {
			t.Fatal("a non-member must not reach the service")
			return db.CardSubtask{}, nil
		},
	}
	h := NewSubtaskHandler(svc, nonMemberBoards(), nil, &mock.MockBroadcaster{})
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateSubtask)(w, subtaskRequest(http.MethodPost, `{"title":"x","position":1}`, "cardID", validCardID))

	assert.Equal(t, http.StatusNotFound, w.Code, "404 not 403: membership is not disclosed")
}

func TestCreateSubtask_UnknownCard_Returns404(t *testing.T) {
	h := NewSubtaskHandler(&mock.MockSubtaskService{}, memberBoards(), nil, &mock.MockBroadcaster{})
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateSubtask)(w, subtaskRequest(http.MethodPost, `{"title":"x","position":1}`, "cardID", otherCardID))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreateSubtask_InvalidJSON_Returns400(t *testing.T) {
	h := NewSubtaskHandler(&mock.MockSubtaskService{}, memberBoards(), nil, &mock.MockBroadcaster{})
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateSubtask)(w, subtaskRequest(http.MethodPost, `{bad json}`, "cardID", validCardID))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateSubtask_ServiceError_Returns500WithoutBroadcast(t *testing.T) {
	svc := &mock.MockSubtaskService{
		CreateSubtaskFn: func(ctx context.Context, cardID, title string) (db.CardSubtask, error) {
			return db.CardSubtask{}, errors.New("db error")
		},
	}
	bc := &mock.MockBroadcaster{}
	h := NewSubtaskHandler(svc, memberBoards(), nil, bc)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateSubtask)(w, subtaskRequest(http.MethodPost, `{"title":"x","position":1}`, "cardID", validCardID))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Empty(t, bc.Sent)
}

// ────────────────────────────────────────────────
// GetSubtasks / GetSubtask
// ────────────────────────────────────────────────

func TestGetSubtasks_Success(t *testing.T) {
	h := NewSubtaskHandler(subtaskService(validCardID), memberBoards(), nil, nil)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetSubtasks)(w, subtaskRequest(http.MethodGet, "", "cardID", validCardID))

	assert.Equal(t, http.StatusOK, w.Code)
	var res []map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&res))
	assert.Len(t, res, 1)
}

func TestGetSubtasks_InvalidCardID_Returns400(t *testing.T) {
	h := NewSubtaskHandler(&mock.MockSubtaskService{}, memberBoards(), nil, nil)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetSubtasks)(w, subtaskRequest(http.MethodGet, "", "cardID", "not-a-uuid"))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetSubtasks_NonMember_Returns404(t *testing.T) {
	h := NewSubtaskHandler(subtaskService(validCardID), nonMemberBoards(), nil, nil)
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
	h := NewSubtaskHandler(svc, memberBoards(), nil, nil)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetSubtasks)(w, subtaskRequest(http.MethodGet, "", "cardID", validCardID))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetSubtask_Success(t *testing.T) {
	h := NewSubtaskHandler(subtaskService(validCardID), memberBoards(), nil, nil)
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
	h := NewSubtaskHandler(svc, memberBoards(), nil, nil)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetSubtask)(w, subtaskRequest(http.MethodGet, "", "cardID", validCardID, "subtaskID", validSubtaskID))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// A member of one board must not reach a subtask on another board by pairing a card
// they can see with a foreign subtask ID.
func TestGetSubtask_SubtaskOnAnotherCard_Returns404(t *testing.T) {
	h := NewSubtaskHandler(subtaskService(otherCardID), memberBoards(), nil, nil)
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
	h := NewSubtaskHandler(svc, memberBoards(), nil, bc)
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
	h := NewSubtaskHandler(svc, memberBoards(), nil, bc)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateSubtask)(w, subtaskRequest(http.MethodPatch, `{"is_done":true}`, "cardID", validCardID, "subtaskID", validSubtaskID))

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Empty(t, bc.Sent)
}

func TestUpdateSubtask_NonMember_Returns404(t *testing.T) {
	h := NewSubtaskHandler(subtaskService(validCardID), nonMemberBoards(), nil, &mock.MockBroadcaster{})
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateSubtask)(w, subtaskRequest(http.MethodPatch, `{"is_done":true}`, "cardID", validCardID, "subtaskID", validSubtaskID))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateSubtask_InvalidSubtaskID_Returns400(t *testing.T) {
	h := NewSubtaskHandler(&mock.MockSubtaskService{}, memberBoards(), nil, nil)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateSubtask)(w, subtaskRequest(http.MethodPatch, `{"is_done":true}`, "cardID", validCardID, "subtaskID", ""))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSubtask_InvalidJSON_Returns400(t *testing.T) {
	h := NewSubtaskHandler(&mock.MockSubtaskService{}, memberBoards(), nil, nil)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateSubtask)(w, subtaskRequest(http.MethodPatch, `{bad json}`, "cardID", validCardID, "subtaskID", validSubtaskID))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSubtask_ServiceError_Returns500(t *testing.T) {
	svc := subtaskService(validCardID)
	svc.UpdateSubtaskFn = func(ctx context.Context, subtaskID string, req dto.UpdateSubtaskRequest) (db.CardSubtask, error) {
		return db.CardSubtask{}, errors.New("db error")
	}
	h := NewSubtaskHandler(svc, memberBoards(), nil, &mock.MockBroadcaster{})
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
	h := NewSubtaskHandler(svc, memberBoards(), nil, bc)
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
	h := NewSubtaskHandler(svc, memberBoards(), nil, bc)
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
	h := NewSubtaskHandler(svc, memberBoards(), nil, &mock.MockBroadcaster{})
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.DeleteSubtask)(w, subtaskRequest(http.MethodDelete, "", "cardID", validCardID, "subtaskID", validSubtaskID))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteSubtask_ServiceError_Returns500(t *testing.T) {
	svc := subtaskService(validCardID)
	svc.DeleteSubtaskFn = func(ctx context.Context, subtaskID string) error {
		return errors.New("db error")
	}
	h := NewSubtaskHandler(svc, memberBoards(), nil, &mock.MockBroadcaster{})
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.DeleteSubtask)(w, subtaskRequest(http.MethodDelete, "", "cardID", validCardID, "subtaskID", validSubtaskID))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ────────────────────────────────────────────────
// Edit rule (same as the card: creator, assignee, or manager+)
// ────────────────────────────────────────────────

func TestSubtaskWrites_MemberOnSomeoneElsesCard_Returns403(t *testing.T) {
	cases := map[string]struct {
		call   func(h *SubtaskHandler) httputil.APIFunc
		method string
		body   string
	}{
		"create": {func(h *SubtaskHandler) httputil.APIFunc { return h.CreateSubtask }, http.MethodPost, `{"title":"x","position":1}`},
		"toggle": {func(h *SubtaskHandler) httputil.APIFunc { return h.UpdateSubtask }, http.MethodPatch, `{"is_done":true}`},
		"delete": {func(h *SubtaskHandler) httputil.APIFunc { return h.DeleteSubtask }, http.MethodDelete, ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// No service Fn is set beyond reads: a write reaching the service would panic.
			svc := &mock.MockSubtaskService{}
			bc := &mock.MockBroadcaster{}
			h := NewSubtaskHandler(svc, bystanderBoards(), nil, bc)
			w := httptest.NewRecorder()

			httputil.MakeHandler(tc.call(h))(w, subtaskRequest(tc.method, tc.body, "cardID", validCardID, "subtaskID", validSubtaskID))

			assert.Equal(t, http.StatusForbidden, w.Code)
			assert.Empty(t, bc.Sent)
		})
	}
}

func TestSubtaskReads_MemberOnSomeoneElsesCard_Allowed(t *testing.T) {
	h := NewSubtaskHandler(subtaskService(validCardID), bystanderBoards(), nil, nil)

	w := httptest.NewRecorder()
	httputil.MakeHandler(h.GetSubtasks)(w, subtaskRequest(http.MethodGet, "", "cardID", validCardID))
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	httputil.MakeHandler(h.GetSubtask)(w, subtaskRequest(http.MethodGet, "", "cardID", validCardID, "subtaskID", validSubtaskID))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateSubtask_EditRights_AllowedRoles(t *testing.T) {
	for name, boards := range map[string]*mock.MockBoardService{
		"creator":  boardsFor("member", cardOwnedBy(ptr(validUserID), nil)),
		"assignee": boardsFor("member", cardOwnedBy(ptr(otherUserID), ptr(validUserID))),
		"manager":  boardsFor("manager", cardOwnedBy(ptr(otherUserID), ptr(otherUserID))),
		"owner":    boardsFor("owner", cardOwnedBy(ptr(otherUserID), nil)),
	} {
		t.Run(name, func(t *testing.T) {
			svc := subtaskService(validCardID)
			svc.UpdateSubtaskFn = func(ctx context.Context, subtaskID string, req dto.UpdateSubtaskRequest) (db.CardSubtask, error) {
				return db.CardSubtask{ID: subtaskID, CardID: validCardID, IsDone: true}, nil
			}
			h := NewSubtaskHandler(svc, boards, nil, &mock.MockBroadcaster{})
			w := httptest.NewRecorder()

			httputil.MakeHandler(h.UpdateSubtask)(w, subtaskRequest(http.MethodPatch, `{"is_done":true}`, "cardID", validCardID, "subtaskID", validSubtaskID))

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// ────────────────────────────────────────────────
// card.subtasks_completed activity
// ────────────────────────────────────────────────

func TestUpdateSubtask_SubtasksCompletedActivity(t *testing.T) {
	cases := map[string]struct {
		wasDone   bool
		nowDone   bool
		afterList []db.CardSubtask
		wantEvent bool
	}{
		"last open subtask ticked": {false, true, []db.CardSubtask{{IsDone: true}, {IsDone: true}}, true},
		"others still open":        {false, true, []db.CardSubtask{{IsDone: true}, {IsDone: false}}, false},
		"unticked":                 {true, false, []db.CardSubtask{{IsDone: false}}, false},
		"rename of a done subtask": {true, true, []db.CardSubtask{{IsDone: true}}, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var got []service.RecordParams
			svc := &mock.MockSubtaskService{
				GetSubtaskByIDFn: func(ctx context.Context, id string) (db.CardSubtask, error) {
					return db.CardSubtask{ID: id, CardID: validCardID, IsDone: tc.wasDone}, nil
				},
				UpdateSubtaskFn: func(ctx context.Context, id string, req dto.UpdateSubtaskRequest) (db.CardSubtask, error) {
					return db.CardSubtask{ID: id, CardID: validCardID, IsDone: tc.nowDone}, nil
				},
				GetSubtasksByCardIDFn: func(ctx context.Context, cardID string) ([]db.CardSubtask, error) {
					return tc.afterList, nil
				},
			}
			rec := &spyRecorder{record: func(ctx context.Context, p service.RecordParams) error {
				got = append(got, p)
				return nil
			}}
			h := NewSubtaskHandler(svc, memberBoards(), rec, &mock.MockBroadcaster{})
			w := httptest.NewRecorder()

			httputil.MakeHandler(h.UpdateSubtask)(w, subtaskRequest(http.MethodPatch, `{"is_done":true}`, "cardID", validCardID, "subtaskID", validSubtaskID))

			require.Equal(t, http.StatusOK, w.Code)
			if !tc.wantEvent {
				assert.Empty(t, got)
				return
			}
			require.Len(t, got, 1)
			assert.Equal(t, service.EventCardSubtasksCompleted, got[0].EventType)
			assert.Equal(t, validUserID, got[0].ActorID)
			assert.Equal(t, service.CardSubtasksCompletedPayload{Title: "Existing", Total: 2}, got[0].Payload)
		})
	}
}
