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

// validBoardID / validUserID defined in board_handler_test.go (same package).
const (
	validPlanningItemID    = "11111111-2222-3333-4444-555555555555"
	validPlanningSessionID = "22222222-3333-4444-5555-666666666666"
	validPromotedCardID    = "66666666-7777-8888-9999-aaaaaaaaaaaa"
)

// newPromoteTestRig wires the three collaborators a PromoteItem test needs.
// Each test then overrides the specific Fn fields it cares about; methods
// nothing touches stay nil and would panic if invoked, which is the point —
// missing stubs surface loudly rather than silently returning zero values.
func newPromoteTestRig() (*mock.MockPlanningService, *mock.MockBoardService, *mock.MockActivityRecorder, *PlanningHandler) {
	plan := &mock.MockPlanningService{}
	boards := &mock.MockBoardService{
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			return "owner", nil
		},
	}
	act := &mock.MockActivityRecorder{}
	h := NewPlanningHandler(plan, boards, act)
	return plan, boards, act, h
}

func newPromoteRequest(t *testing.T, itemID, userID string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/planning/items/"+itemID+"/promote", strings.NewReader("{}"))
	req = chiCtx(req, "itemID", itemID)
	return withUserID(req, userID)
}

// ────────────────────────────────────────────────
// PromoteItem — happy path + edge cases
// ────────────────────────────────────────────────

func TestPromoteItem_HappyPath_CreatesCardAndRecordsActivity(t *testing.T) {
	plan, _, act, h := newPromoteTestRig()
	stubItemAndBoard(plan)
	plan.PromoteItemFn = func(ctx context.Context, itemID, userID string) (db.PlanningItem, db.CreateCardRow, error) {
		return db.PlanningItem{
				ID:     validPlanningItemID,
				Type:   "REQ",
				Title:  "Add Google login",
				Status: "promoted",
			},
			db.CreateCardRow{ID: validPromotedCardID},
			nil
	}

	w := httptest.NewRecorder()
	httputil.MakeHandler(h.PromoteItem)(w, newPromoteRequest(t, validPlanningItemID, validUserID))

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, validPromotedCardID, body["card_id"])

	// Activity assertion — single planning.item_promoted row, payload
	// carries the new card_id so the feed can deep-link.
	require.Len(t, act.Calls, 1)
	call := act.Calls[0]
	assert.Equal(t, service.EventPlanningItemPromoted, call.EventType)
	assert.Equal(t, service.EntityPlanningItem, call.EntityType)
	require.NotNil(t, call.EntityID)
	assert.Equal(t, validPlanningItemID, *call.EntityID)
	payload, ok := call.Payload.(service.PlanningItemPromotedPayload)
	require.True(t, ok, "payload should be PlanningItemPromotedPayload")
	assert.Equal(t, "REQ", payload.Type)
	assert.Equal(t, "Add Google login", payload.Title)
	assert.Equal(t, validPromotedCardID, payload.ToCardID)
}

func TestPromoteItem_AlreadyPromoted_Returns409(t *testing.T) {
	plan, _, act, h := newPromoteTestRig()
	stubItemAndBoard(plan)
	plan.PromoteItemFn = func(ctx context.Context, itemID, userID string) (db.PlanningItem, db.CreateCardRow, error) {
		return db.PlanningItem{}, db.CreateCardRow{}, service.ErrPlanningItemAlreadyPromoted
	}

	w := httptest.NewRecorder()
	httputil.MakeHandler(h.PromoteItem)(w, newPromoteRequest(t, validPlanningItemID, validUserID))

	assert.Equal(t, http.StatusConflict, w.Code)
	// No activity row should be written on failure — we don't want the
	// audit feed showing a "promoted" event when the second attempt was
	// rejected.
	assert.Empty(t, act.Calls)
}

func TestPromoteItem_DroppedItem_Returns422(t *testing.T) {
	plan, _, act, h := newPromoteTestRig()
	stubItemAndBoard(plan)
	plan.PromoteItemFn = func(ctx context.Context, itemID, userID string) (db.PlanningItem, db.CreateCardRow, error) {
		return db.PlanningItem{}, db.CreateCardRow{}, service.ErrPlanningItemDropped
	}

	w := httptest.NewRecorder()
	httputil.MakeHandler(h.PromoteItem)(w, newPromoteRequest(t, validPlanningItemID, validUserID))

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "dropped")
	assert.Empty(t, act.Calls)
}

func TestPromoteItem_NoTodoColumn_Returns422(t *testing.T) {
	plan, _, act, h := newPromoteTestRig()
	stubItemAndBoard(plan)
	plan.PromoteItemFn = func(ctx context.Context, itemID, userID string) (db.PlanningItem, db.CreateCardRow, error) {
		return db.PlanningItem{}, db.CreateCardRow{}, service.ErrPlanningNoTodoColumn
	}

	w := httptest.NewRecorder()
	httputil.MakeHandler(h.PromoteItem)(w, newPromoteRequest(t, validPlanningItemID, validUserID))

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "TODO")
	assert.Empty(t, act.Calls)
}

func TestPromoteItem_NotMember_Returns404(t *testing.T) {
	plan, boards, act, h := newPromoteTestRig()
	stubItemAndBoard(plan)
	// Membership lookup returns pgx.ErrNoRows when the user isn't a member;
	// the handler must surface this as 404 (anti-enumeration), not 403.
	boards.GetBoardMemberRoleFn = func(ctx context.Context, boardID, userID string) (string, error) {
		return "", pgx.ErrNoRows
	}

	w := httptest.NewRecorder()
	httputil.MakeHandler(h.PromoteItem)(w, newPromoteRequest(t, validPlanningItemID, validUserID))

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Empty(t, act.Calls)
}

// PATCH empty-string convention for UpdateSession and UpdateItem, per AGENTS.md:
// omit or null means no change, "" on a required field is a 400, "" on a nullable one
// reaches the service as &"".

func TestUpdateSession_TitleEmpty_Returns400(t *testing.T) {
	plan, _, act, h := newPromoteTestRig()
	plan.GetSessionBoardIDFn = func(ctx context.Context, sessionID string) (string, error) {
		return validBoardID, nil
	}

	body := strings.NewReader(`{"title": ""}`)
	req := httptest.NewRequest(http.MethodPatch, "/planning/sessions/"+validPlanningSessionID, body)
	req = chiCtx(req, "sessionID", validPlanningSessionID)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()
	httputil.MakeHandler(h.UpdateSession)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "title")
	assert.Empty(t, act.Calls, "no activity row should be recorded on 400")
}

func TestUpdateSession_PartialPatch_OmittedFieldsArriveAsNil(t *testing.T) {
	plan, _, _, h := newPromoteTestRig()
	plan.GetSessionBoardIDFn = func(ctx context.Context, sessionID string) (string, error) {
		return validBoardID, nil
	}
	var capturedTitle, capturedLabel, capturedMeetingAt *string
	plan.UpdateSessionFn = func(ctx context.Context, sessionID string, title, label, meetingAt *string) (db.PlanningSession, error) {
		capturedTitle = title
		capturedLabel = label
		capturedMeetingAt = meetingAt
		return db.PlanningSession{ID: sessionID, Title: "Existing"}, nil
	}

	// Only label is patched; title and meeting_at are omitted from JSON.
	body := strings.NewReader(`{"label": "with @client"}`)
	req := httptest.NewRequest(http.MethodPatch, "/planning/sessions/"+validPlanningSessionID, body)
	req = chiCtx(req, "sessionID", validPlanningSessionID)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()
	httputil.MakeHandler(h.UpdateSession)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, capturedTitle, "omitted title should arrive as nil so SQL's COALESCE preserves it")
	require.NotNil(t, capturedLabel)
	assert.Equal(t, "with @client", *capturedLabel)
	assert.Nil(t, capturedMeetingAt, "omitted meeting_at should arrive as nil")
}

func TestUpdateSession_EmptyLabel_PassedThroughToService(t *testing.T) {
	// Nullable column convention: "" is a real value (stored as ""), NOT
	// "no change". Handler must pass &"" through, not collapse to nil.
	plan, _, _, h := newPromoteTestRig()
	plan.GetSessionBoardIDFn = func(ctx context.Context, sessionID string) (string, error) {
		return validBoardID, nil
	}
	var capturedLabel *string
	plan.UpdateSessionFn = func(ctx context.Context, sessionID string, title, label, meetingAt *string) (db.PlanningSession, error) {
		capturedLabel = label
		return db.PlanningSession{ID: sessionID}, nil
	}

	body := strings.NewReader(`{"label": ""}`)
	req := httptest.NewRequest(http.MethodPatch, "/planning/sessions/"+validPlanningSessionID, body)
	req = chiCtx(req, "sessionID", validPlanningSessionID)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()
	httputil.MakeHandler(h.UpdateSession)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capturedLabel, "empty-string label should arrive as &\"\", not nil")
	assert.Equal(t, "", *capturedLabel)
}

// stubItemAndBoard wires the GetItem + GetSessionBoardID pair UpdateItem calls.
// Defaults to a live REQ item under validBoardID; the mutator models promoted, dropped
// or different-type items for retype tests.
func stubItemAndBoard(plan *mock.MockPlanningService, mutate ...func(*db.PlanningItem)) {
	plan.GetItemFn = func(ctx context.Context, itemID string) (db.PlanningItem, error) {
		it := db.PlanningItem{
			ID:        itemID,
			SessionID: validPlanningSessionID,
			Type:      "REQ",
			Title:     "Existing item",
			Status:    "live",
		}
		for _, m := range mutate {
			m(&it)
		}
		return it, nil
	}
	plan.GetSessionBoardIDFn = func(ctx context.Context, sessionID string) (string, error) {
		return validBoardID, nil
	}
}

func TestUpdateItem_TitleEmpty_Returns400(t *testing.T) {
	plan, _, act, h := newPromoteTestRig()
	stubItemAndBoard(plan)

	body := strings.NewReader(`{"title": ""}`)
	req := httptest.NewRequest(http.MethodPatch, "/planning/items/"+validPlanningItemID, body)
	req = chiCtx(req, "itemID", validPlanningItemID)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()
	httputil.MakeHandler(h.UpdateItem)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "title")
	assert.Empty(t, act.Calls)
}

func TestUpdateItem_EmptyDescription_PassedThroughToService(t *testing.T) {
	plan, _, _, h := newPromoteTestRig()
	stubItemAndBoard(plan)
	var capturedDescription *string
	plan.UpdateItemFn = func(ctx context.Context, itemID string, itemType, title *string, description *string, status *string, position *float64, acceptanceCriteria, implementationNote *string) (db.PlanningItem, error) {
		capturedDescription = description
		return db.PlanningItem{ID: itemID, Type: "REQ", Title: "Test"}, nil
	}

	body := strings.NewReader(`{"description": ""}`)
	req := httptest.NewRequest(http.MethodPatch, "/planning/items/"+validPlanningItemID, body)
	req = chiCtx(req, "itemID", validPlanningItemID)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()
	httputil.MakeHandler(h.UpdateItem)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capturedDescription, "empty-string description should arrive as &\"\", not nil")
	assert.Equal(t, "", *capturedDescription)
}

func TestUpdateItem_OnlyStatusSent_OtherFieldsNilAtService(t *testing.T) {
	// Drop / undrop flow sends `{"status":"dropped"}` only. Service must
	// see type/title/description/position as nil so SQL's COALESCE keeps
	// them. Regressions here would silently wipe titles on drop.
	plan, _, _, h := newPromoteTestRig()
	stubItemAndBoard(plan)
	var capturedType, capturedTitle, capturedDescription, capturedStatus *string
	var capturedPosition *float64
	plan.UpdateItemFn = func(ctx context.Context, itemID string, itemType, title *string, description *string, status *string, position *float64, acceptanceCriteria, implementationNote *string) (db.PlanningItem, error) {
		capturedType = itemType
		capturedTitle = title
		capturedDescription = description
		capturedStatus = status
		capturedPosition = position
		return db.PlanningItem{ID: itemID, Type: "REQ", Title: "Test"}, nil
	}

	body := strings.NewReader(`{"status": "dropped"}`)
	req := httptest.NewRequest(http.MethodPatch, "/planning/items/"+validPlanningItemID, body)
	req = chiCtx(req, "itemID", validPlanningItemID)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()
	httputil.MakeHandler(h.UpdateItem)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, capturedType)
	assert.Nil(t, capturedTitle)
	assert.Nil(t, capturedDescription)
	require.NotNil(t, capturedStatus)
	assert.Equal(t, "dropped", *capturedStatus)
	assert.Nil(t, capturedPosition)
}

// ────────────────────────────────────────────────
// GetCardSource — backlink card → planning item
// ────────────────────────────────────────────────

func newCardSourceRequest(t *testing.T, cardID, userID string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/cards/"+cardID+"/source", nil)
	req = chiCtx(req, "cardID", cardID)
	return withUserID(req, userID)
}

func TestGetCardSource_PromotedCard_ReturnsSessionAndItem(t *testing.T) {
	plan, boards, _, h := newPromoteTestRig()
	boards.GetBoardIDByCardFn = func(ctx context.Context, cardID string) (string, error) {
		return validBoardID, nil
	}
	plan.GetCardSourceFn = func(ctx context.Context, cardID string, pendingLimit int32) (*service.CardSource, error) {
		assert.Equal(t, int32(3), pendingLimit)
		return &service.CardSource{
			SessionID:    validPlanningSessionID,
			SessionTitle: "Sprint Kickoff",
			ItemID:       validPlanningItemID,
			ItemType:     "REQ",
			ItemTitle:    "Add Google login",
			ItemStatus:   "promoted",
			PendingQuestions: []service.CardSourcePendingQuestion{
				{ID: "q1", Title: "Which Google scope?"},
			},
		}, nil
	}

	w := httptest.NewRecorder()
	httputil.MakeHandler(h.GetCardSource)(w, newCardSourceRequest(t, validPromotedCardID, validUserID))

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	session, _ := body["session"].(map[string]any)
	require.NotNil(t, session, "session field must be present")
	assert.Equal(t, "Sprint Kickoff", session["title"])
	item, _ := body["item"].(map[string]any)
	require.NotNil(t, item)
	assert.Equal(t, "REQ", item["type"])
	questions, _ := body["pending_questions"].([]any)
	require.Len(t, questions, 1)
}

func TestGetCardSource_CardNotPromoted_Returns200Null(t *testing.T) {
	// A non-planning-promoted card must NOT return 404 — the endpoint says
	// "what's the source of this card?" and "no source" is a valid answer.
	// 404 would force the frontend into an error fork for the common case.
	plan, boards, _, h := newPromoteTestRig()
	boards.GetBoardIDByCardFn = func(ctx context.Context, cardID string) (string, error) {
		return validBoardID, nil
	}
	plan.GetCardSourceFn = func(ctx context.Context, cardID string, pendingLimit int32) (*service.CardSource, error) {
		return nil, nil
	}

	w := httptest.NewRecorder()
	httputil.MakeHandler(h.GetCardSource)(w, newCardSourceRequest(t, validPromotedCardID, validUserID))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "null", w.Body.String())
}

func TestGetCardSource_CardNotFound_Returns404(t *testing.T) {
	_, boards, _, h := newPromoteTestRig()
	boards.GetBoardIDByCardFn = func(ctx context.Context, cardID string) (string, error) {
		return "", pgx.ErrNoRows
	}

	w := httptest.NewRecorder()
	httputil.MakeHandler(h.GetCardSource)(w, newCardSourceRequest(t, validPromotedCardID, validUserID))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetCardSource_NotMember_Returns404(t *testing.T) {
	// Anti-enumeration: a non-member probing /api/cards/X/source must not
	// be able to tell apart "card doesn't exist" from "you don't have
	// access to its board". Both arrive as 404.
	_, boards, _, h := newPromoteTestRig()
	boards.GetBoardIDByCardFn = func(ctx context.Context, cardID string) (string, error) {
		return validBoardID, nil
	}
	boards.GetBoardMemberRoleFn = func(ctx context.Context, boardID, userID string) (string, error) {
		return "", pgx.ErrNoRows
	}

	w := httptest.NewRecorder()
	httputil.MakeHandler(h.GetCardSource)(w, newCardSourceRequest(t, validPromotedCardID, validUserID))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetCardSource_BadCardID_Returns400(t *testing.T) {
	_, _, _, h := newPromoteTestRig()
	w := httptest.NewRecorder()
	httputil.MakeHandler(h.GetCardSource)(w, newCardSourceRequest(t, "not-a-uuid", validUserID))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ────────────────────────────────────────────────
// UpdateItem — type conversion (B-F2)
// ────────────────────────────────────────────────

func TestUpdateItem_RetypeOnLiveItem_RecordsPreviousType(t *testing.T) {
	// Q to DEC on a live item is the common "we finally decided" flow. The payload must
	// carry both the new type and previous_type so the chip tooltip renders without a
	// second query.
	plan, _, act, h := newPromoteTestRig()
	stubItemAndBoard(plan, func(it *db.PlanningItem) { it.Type = "Q" })
	plan.UpdateItemFn = func(ctx context.Context, itemID string, itemType, title *string, description *string, status *string, position *float64, acceptanceCriteria, implementationNote *string) (db.PlanningItem, error) {
		require.NotNil(t, itemType)
		assert.Equal(t, "DEC", *itemType)
		return db.PlanningItem{ID: itemID, Type: "DEC", Title: "Existing item", Status: "live"}, nil
	}

	body := strings.NewReader(`{"type": "DEC"}`)
	req := httptest.NewRequest(http.MethodPatch, "/planning/items/"+validPlanningItemID, body)
	req = chiCtx(req, "itemID", validPlanningItemID)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()
	httputil.MakeHandler(h.UpdateItem)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, act.Calls, 1)
	payload, ok := act.Calls[0].Payload.(service.PlanningItemUpdatedPayload)
	require.True(t, ok)
	assert.Equal(t, []string{"type"}, payload.Fields)
	assert.Equal(t, "DEC", payload.Type)
	assert.Equal(t, "Q", payload.PreviousType, "previous_type must record the pre-update type for the history tooltip")
}

func TestUpdateItem_RetypeOnPromoted_Returns400(t *testing.T) {
	// Promoted items are frozen for retype — the card already carries the original
	// semantics. The handler must reject with a Thai-friendly 400 so the UI reverts.
	plan, _, act, h := newPromoteTestRig()
	stubItemAndBoard(plan, func(it *db.PlanningItem) {
		it.Type = "REQ"
		it.Status = "promoted"
	})

	body := strings.NewReader(`{"type": "Q"}`)
	req := httptest.NewRequest(http.MethodPatch, "/planning/items/"+validPlanningItemID, body)
	req = chiCtx(req, "itemID", validPlanningItemID)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()
	httputil.MakeHandler(h.UpdateItem)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Board")
	assert.Empty(t, act.Calls, "rejected retype must not produce an audit row")
}

func TestUpdateItem_RetypeSameType_NoPreviousTypeInPayload(t *testing.T) {
	// Idempotent retype (REQ → REQ) — the type field is "set" in the
	// request but didn't actually change. previous_type should be empty so
	// the feed doesn't render a misleading "changed from REQ to REQ" line.
	plan, _, act, h := newPromoteTestRig()
	stubItemAndBoard(plan, func(it *db.PlanningItem) { it.Type = "REQ" })
	plan.UpdateItemFn = func(ctx context.Context, itemID string, itemType, title *string, description *string, status *string, position *float64, acceptanceCriteria, implementationNote *string) (db.PlanningItem, error) {
		return db.PlanningItem{ID: itemID, Type: "REQ", Title: "Existing item"}, nil
	}

	body := strings.NewReader(`{"type": "REQ"}`)
	req := httptest.NewRequest(http.MethodPatch, "/planning/items/"+validPlanningItemID, body)
	req = chiCtx(req, "itemID", validPlanningItemID)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()
	httputil.MakeHandler(h.UpdateItem)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, act.Calls, 1)
	payload, ok := act.Calls[0].Payload.(service.PlanningItemUpdatedPayload)
	require.True(t, ok)
	assert.Equal(t, "", payload.PreviousType)
}

// ────────────────────────────────────────────────
// UpdateItem — acceptance_criteria + implementation_note (B-F3)
// ────────────────────────────────────────────────

func TestUpdateItem_SetAcceptanceCriteria_PassesThroughAndLogsField(t *testing.T) {
	// Confirms the new PATCH fields reach the service and appear in the activity Fields
	// list. Without it a UI regression dropping a field would no-op while the audit feed
	// still claimed the change landed.
	plan, _, act, h := newPromoteTestRig()
	stubItemAndBoard(plan)
	var capturedAC, capturedNote *string
	plan.UpdateItemFn = func(ctx context.Context, itemID string, itemType, title *string, description *string, status *string, position *float64, acceptanceCriteria, implementationNote *string) (db.PlanningItem, error) {
		capturedAC = acceptanceCriteria
		capturedNote = implementationNote
		return db.PlanningItem{ID: itemID, Type: "REQ", Title: "Existing item"}, nil
	}

	body := strings.NewReader(`{"acceptance_criteria": "login ด้วย email ได้\nlogin ด้วย Google ได้"}`)
	req := httptest.NewRequest(http.MethodPatch, "/planning/items/"+validPlanningItemID, body)
	req = chiCtx(req, "itemID", validPlanningItemID)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()
	httputil.MakeHandler(h.UpdateItem)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capturedAC)
	assert.Contains(t, *capturedAC, "login")
	assert.Nil(t, capturedNote, "omitted note should arrive nil so COALESCE preserves it")

	require.Len(t, act.Calls, 1)
	payload, ok := act.Calls[0].Payload.(service.PlanningItemUpdatedPayload)
	require.True(t, ok)
	assert.Contains(t, payload.Fields, "acceptance_criteria")
	assert.NotContains(t, payload.Fields, "implementation_note")
}
