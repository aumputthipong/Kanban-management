package handler

import (
	"bytes"
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
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/dto"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service/mock"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func jsonBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

const otherUserID = "11111111-2222-3333-4444-555555555555"

// CreateCard

// UpdateCard

func cardOwnedBy(creatorID, assigneeID *string) db.Card {
	return db.Card{
		ID:         validCardID,
		ColumnID:   validColumnID,
		Title:      "Existing",
		CreatedBy:  creatorID,
		AssigneeID: assigneeID,
	}
}

func ptr(s string) *string { return &s }

func titleOr(title *string, stored string) string {
	if title == nil {
		return stored
	}
	return *title
}

func TestUpdateCard_ManagerEditsAnyCard_Success(t *testing.T) {
	other := ptr(otherUserID)
	svc := &mock.MockBoardService{
		GetCardFn: func(ctx context.Context, cardID string) (db.Card, error) {
			return cardOwnedBy(other, other), nil
		},
		GetBoardIDByColumnFn: func(ctx context.Context, columnID string) (string, error) {
			return validBoardID, nil
		},
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			return "manager", nil
		},
		UpdateCardFn: func(ctx context.Context, arg service.UpdateCardParams) (service.UpdateCardResult, error) {
			return service.UpdateCardResult{Card: db.Card{ID: arg.ID, Title: titleOr(arg.Title, "Existing")}}, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	body := map[string]any{"title": "edited by manager"}
	req := withUserID(httptest.NewRequest(http.MethodPatch, "/cards/"+validCardID, jsonBody(t, body)), validUserID)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateCard)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateCard_MemberEditsOwnCard_Success(t *testing.T) {
	creator := ptr(validUserID)
	svc := &mock.MockBoardService{
		GetCardFn: func(ctx context.Context, cardID string) (db.Card, error) {
			return cardOwnedBy(creator, nil), nil
		},
		GetBoardIDByColumnFn: func(ctx context.Context, columnID string) (string, error) {
			return validBoardID, nil
		},
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			return "member", nil
		},
		UpdateCardFn: func(ctx context.Context, arg service.UpdateCardParams) (service.UpdateCardResult, error) {
			return service.UpdateCardResult{Card: db.Card{ID: arg.ID, Title: titleOr(arg.Title, "Existing")}}, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	body := map[string]any{"title": "my own edit"}
	req := withUserID(httptest.NewRequest(http.MethodPatch, "/cards/"+validCardID, jsonBody(t, body)), validUserID)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateCard)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateCard_MemberEditsAssignedCard_Success(t *testing.T) {
	assignee := ptr(validUserID)
	other := ptr(otherUserID)
	svc := &mock.MockBoardService{
		GetCardFn: func(ctx context.Context, cardID string) (db.Card, error) {
			return cardOwnedBy(other, assignee), nil
		},
		GetBoardIDByColumnFn: func(ctx context.Context, columnID string) (string, error) {
			return validBoardID, nil
		},
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			return "member", nil
		},
		UpdateCardFn: func(ctx context.Context, arg service.UpdateCardParams) (service.UpdateCardResult, error) {
			return service.UpdateCardResult{Card: db.Card{ID: arg.ID}}, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	body := map[string]any{"title": "assignee edit"}
	req := withUserID(httptest.NewRequest(http.MethodPatch, "/cards/"+validCardID, jsonBody(t, body)), validUserID)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateCard)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// A plain member can't edit a card they neither created nor were assigned.
func TestUpdateCard_MemberEditsOthersCard_Returns403(t *testing.T) {
	other := ptr(otherUserID)
	svc := &mock.MockBoardService{
		GetCardFn: func(ctx context.Context, cardID string) (db.Card, error) {
			return cardOwnedBy(other, other), nil
		},
		GetBoardIDByColumnFn: func(ctx context.Context, columnID string) (string, error) {
			return validBoardID, nil
		},
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			return "member", nil
		},
		UpdateCardFn: func(ctx context.Context, arg service.UpdateCardParams) (service.UpdateCardResult, error) {
			t.Fatal("UpdateCard must not run when permission check fails")
			return service.UpdateCardResult{}, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	body := map[string]any{"title": "should be rejected"}
	req := withUserID(httptest.NewRequest(http.MethodPatch, "/cards/"+validCardID, jsonBody(t, body)), validUserID)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateCard)(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestUpdateCard_InvalidCardIDFormat_Returns400(t *testing.T) {
	svc := &mock.MockBoardService{}
	h := NewBoardHandler(svc, nil, nil, nil)

	body := map[string]any{"title": "x"}
	req := withUserID(httptest.NewRequest(http.MethodPatch, "/cards/not-a-uuid", jsonBody(t, body)), validUserID)
	req = chiCtx(req, "cardID", "not-a-uuid")
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateCard)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateCard_CardNotFound_Returns404(t *testing.T) {
	svc := &mock.MockBoardService{
		GetCardFn: func(ctx context.Context, cardID string) (db.Card, error) {
			return db.Card{}, pgx.ErrNoRows
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	body := map[string]any{"title": "x"}
	req := withUserID(httptest.NewRequest(http.MethodPatch, "/cards/"+validCardID, jsonBody(t, body)), validUserID)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateCard)(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// Anti-enumeration: a valid card id must not reveal membership.
func TestUpdateCard_NonMember_Returns404(t *testing.T) {
	other := ptr(otherUserID)
	svc := &mock.MockBoardService{
		GetCardFn: func(ctx context.Context, cardID string) (db.Card, error) {
			return cardOwnedBy(other, other), nil
		},
		GetBoardIDByColumnFn: func(ctx context.Context, columnID string) (string, error) {
			return validBoardID, nil
		},
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			return "", pgx.ErrNoRows
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	body := map[string]any{"title": "x"}
	req := withUserID(httptest.NewRequest(http.MethodPatch, "/cards/"+validCardID, jsonBody(t, body)), validUserID)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateCard)(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// Omitted title must reach the service as nil.
func TestUpdateCard_PATCHSemantics_OmittedTitle(t *testing.T) {
	creator := ptr(validUserID)
	var received service.UpdateCardParams
	svc := &mock.MockBoardService{
		GetCardFn: func(ctx context.Context, cardID string) (db.Card, error) {
			return cardOwnedBy(creator, nil), nil
		},
		GetBoardIDByColumnFn: func(ctx context.Context, columnID string) (string, error) {
			return validBoardID, nil
		},
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			return "member", nil
		},
		UpdateCardFn: func(ctx context.Context, arg service.UpdateCardParams) (service.UpdateCardResult, error) {
			received = arg
			return service.UpdateCardResult{Card: db.Card{ID: arg.ID}}, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	body := map[string]any{"description": "hello"}
	req := withUserID(httptest.NewRequest(http.MethodPatch, "/cards/"+validCardID, jsonBody(t, body)), validUserID)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateCard)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, received.Title, "omitted title must be passed as no-change, not clobbered")
	require.NotNil(t, received.Description)
	assert.Equal(t, "hello", *received.Description)
}

// Snooze regression: a due_date-only PATCH must not wipe other fields.
func TestUpdateCard_PartialPatch_PreservesUntouchedFields(t *testing.T) {
	assignee := ptr(validUserID)
	var received service.UpdateCardParams
	existing := db.Card{
		ID:          validCardID,
		ColumnID:    validColumnID,
		Title:       "Keep me",
		Description: ptr("keep this desc"),
		AssigneeID:  assignee,
		Priority:    ptr("high"),
		CreatedBy:   ptr(otherUserID),
	}
	svc := &mock.MockBoardService{
		GetCardFn: func(ctx context.Context, cardID string) (db.Card, error) {
			return existing, nil
		},
		GetBoardIDByColumnFn: func(ctx context.Context, columnID string) (string, error) {
			return validBoardID, nil
		},
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			return "member", nil
		},
		UpdateCardFn: func(ctx context.Context, arg service.UpdateCardParams) (service.UpdateCardResult, error) {
			received = arg
			return service.UpdateCardResult{Card: db.Card{ID: arg.ID}}, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	body := map[string]any{"due_date": "2026-06-10"}
	req := withUserID(httptest.NewRequest(http.MethodPatch, "/cards/"+validCardID, jsonBody(t, body)), validUserID)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateCard)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, received.Title, "title must survive a due_date-only patch")
	assert.False(t, received.AssigneeID.Set, "assignee must not be touched — card would vanish from My Work")
	assert.Nil(t, received.Description)
	assert.False(t, received.Priority.Set)
	assert.False(t, received.EstimatedHours.Set)
	require.True(t, received.DueDate.Set, "due_date must be applied")
	require.NotNil(t, received.DueDate.Value)
}

// "" on a required column → 400.
func TestUpdateCard_PATCHSemantics_EmptyTitleRejected(t *testing.T) {
	creator := ptr(validUserID)
	svc := &mock.MockBoardService{
		GetCardFn: func(ctx context.Context, cardID string) (db.Card, error) {
			return cardOwnedBy(creator, nil), nil
		},
		UpdateCardFn: func(ctx context.Context, arg service.UpdateCardParams) (service.UpdateCardResult, error) {
			t.Fatal("UpdateCard must not be called when validation fails")
			return service.UpdateCardResult{}, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	body := strings.NewReader(`{"title": ""}`)
	req := withUserID(httptest.NewRequest(http.MethodPatch, "/cards/"+validCardID, body), validUserID)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateCard)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// GetCard

func TestGetCard_Success_RoundtripsTitle(t *testing.T) {
	svc := &mock.MockBoardService{
		GetCardDetailFn: func(ctx context.Context, cardID string) (service.CardDetailData, error) {
			return service.CardDetailData{Card: db.Card{ID: cardID, Title: "Hello"}}, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	req := withUserID(httptest.NewRequest(http.MethodGet, "/cards/"+validCardID, nil), validUserID)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetCard)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "Hello", body["title"])
}

func TestGetCard_InvalidID_Returns400(t *testing.T) {
	svc := &mock.MockBoardService{
		GetCardDetailFn: func(ctx context.Context, cardID string) (service.CardDetailData, error) {
			t.Fatal("must not query DB for malformed ID")
			return service.CardDetailData{}, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	req := withUserID(httptest.NewRequest(http.MethodGet, "/cards/bad", nil), validUserID)
	req = chiCtx(req, "cardID", "bad")
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetCard)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetCard_NotFound_Returns404(t *testing.T) {
	svc := &mock.MockBoardService{
		GetCardDetailFn: func(ctx context.Context, cardID string) (service.CardDetailData, error) {
			return service.CardDetailData{}, sql.ErrNoRows
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	req := withUserID(httptest.NewRequest(http.MethodGet, "/cards/"+validCardID, nil), validUserID)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetCard)(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetCard_DBError_Returns500(t *testing.T) {
	svc := &mock.MockBoardService{
		GetCardDetailFn: func(ctx context.Context, cardID string) (service.CardDetailData, error) {
			return service.CardDetailData{}, errors.New("connection refused")
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	req := withUserID(httptest.NewRequest(http.MethodGet, "/cards/"+validCardID, nil), validUserID)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetCard)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// Pins the snake_case wire shape.
func TestUpdateCard_RespondsWithSnakeCase(t *testing.T) {
	creator := ptr(validUserID)
	svc := &mock.MockBoardService{
		GetCardFn: func(ctx context.Context, cardID string) (db.Card, error) {
			return cardOwnedBy(creator, nil), nil
		},
		GetBoardIDByColumnFn: func(ctx context.Context, columnID string) (string, error) {
			return validBoardID, nil
		},
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			return "member", nil
		},
		UpdateCardFn: func(ctx context.Context, arg service.UpdateCardParams) (service.UpdateCardResult, error) {
			return service.UpdateCardResult{Card: db.Card{ID: arg.ID, ColumnID: validColumnID, Title: titleOr(arg.Title, "Existing")}}, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	req := withUserID(httptest.NewRequest(http.MethodPatch, "/cards/"+validCardID,
		jsonBody(t, map[string]any{"title": "renamed"})), validUserID)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateCard)(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, validCardID, got["id"])
	assert.Equal(t, "renamed", got["title"])
	assert.NotContains(t, got, "ID")
	assert.NotContains(t, got, "Title")
}

// CARD_UPDATED fields come from the updated row, never the request.
func TestUpdateCard_OmittedDueDate_BroadcastsStoredValue(t *testing.T) {
	creator := ptr(validUserID)
	due := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)
	svc := &mock.MockBoardService{
		GetCardFn: func(ctx context.Context, cardID string) (db.Card, error) {
			card := cardOwnedBy(creator, nil)
			card.DueDate = &due
			return card, nil
		},
		GetBoardIDByColumnFn: func(ctx context.Context, columnID string) (string, error) {
			return validBoardID, nil
		},
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			return "member", nil
		},
		UpdateCardFn: func(ctx context.Context, arg service.UpdateCardParams) (service.UpdateCardResult, error) {
			return service.UpdateCardResult{Card: db.Card{ID: arg.ID, ColumnID: validColumnID, Title: titleOr(arg.Title, "Existing"), DueDate: &due}}, nil
		},
	}
	bc := &mock.MockBroadcaster{}
	h := NewBoardHandler(svc, nil, nil, bc)

	req := withUserID(httptest.NewRequest(http.MethodPatch, "/cards/"+validCardID,
		jsonBody(t, map[string]any{"title": "renamed"})), validUserID)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateCard)(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, bc.Sent, 1)
	var msg struct {
		Type    string `json:"type"`
		Payload struct {
			DueDate *string `json:"due_date"`
		} `json:"payload"`
	}
	require.NoError(t, json.Unmarshal(bc.Sent[0].Message, &msg))
	assert.Equal(t, "CARD_UPDATED", msg.Type)
	require.NotNil(t, msg.Payload.DueDate, "due_date must not be null when the caller omitted it")
	assert.Equal(t, "2026-03-14", *msg.Payload.DueDate)
}

func TestUpdateCard_BroadcastsStoredTagsAndNotes(t *testing.T) {
	creator := ptr(validUserID)
	ac := "Filters by priority"
	svc := &mock.MockBoardService{
		GetCardFn: func(ctx context.Context, cardID string) (db.Card, error) {
			return cardOwnedBy(creator, nil), nil
		},
		GetBoardIDByColumnFn: func(ctx context.Context, columnID string) (string, error) {
			return validBoardID, nil
		},
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			return "member", nil
		},
		UpdateCardFn: func(ctx context.Context, arg service.UpdateCardParams) (service.UpdateCardResult, error) {
			return service.UpdateCardResult{
				Card: db.Card{ID: arg.ID, ColumnID: validColumnID, Title: titleOr(arg.Title, "Existing"), AcceptanceCriteria: &ac},
				Tags: []service.TagData{{ID: "tag-1", BoardID: validBoardID, Name: "feat", Color: "blue"}},
			}, nil
		},
	}
	bc := &mock.MockBroadcaster{}
	h := NewBoardHandler(svc, nil, nil, bc)

	req := withUserID(httptest.NewRequest(http.MethodPatch, "/cards/"+validCardID,
		jsonBody(t, map[string]any{"title": "renamed"})), validUserID)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateCard)(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, bc.Sent, 1)
	var msg struct {
		Payload struct {
			Tags               []dto.TagResponse `json:"tags"`
			AcceptanceCriteria *string           `json:"acceptance_criteria"`
		} `json:"payload"`
	}
	require.NoError(t, json.Unmarshal(bc.Sent[0].Message, &msg))
	require.Len(t, msg.Payload.Tags, 1)
	assert.Equal(t, "feat", msg.Payload.Tags[0].Name)
	require.NotNil(t, msg.Payload.AcceptanceCriteria)
	assert.Equal(t, ac, *msg.Payload.AcceptanceCriteria)

	var resp dto.CardResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Tags, 1, "the response carries the stored tags too")
}

// "" and 0 are the clear sentinels (docs/adr/0009).
func TestUpdateCard_ClearSentinels_StoreNull(t *testing.T) {
	var received service.UpdateCardParams
	svc := &mock.MockBoardService{
		GetCardFn: func(ctx context.Context, cardID string) (db.Card, error) {
			card := cardOwnedBy(ptr(validUserID), ptr(validUserID))
			card.Priority = ptr("high")
			card.DueDate = &time.Time{}
			return card, nil
		},
		GetBoardIDByColumnFn: func(ctx context.Context, columnID string) (string, error) {
			return validBoardID, nil
		},
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			return "member", nil
		},
		UpdateCardFn: func(ctx context.Context, arg service.UpdateCardParams) (service.UpdateCardResult, error) {
			received = arg
			return service.UpdateCardResult{Card: db.Card{ID: arg.ID}}, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	body := strings.NewReader(`{"assignee_id": "", "priority": "", "due_date": "", "estimated_hours": 0}`)
	req := withUserID(httptest.NewRequest(http.MethodPatch, "/cards/"+validCardID, body), validUserID)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateCard)(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.True(t, received.AssigneeID.Set)
	assert.Nil(t, received.AssigneeID.Value, `assignee_id "" must store NULL, not an invalid uuid`)
	assert.True(t, received.Priority.Set)
	assert.Nil(t, received.Priority.Value)
	assert.True(t, received.DueDate.Set)
	assert.Nil(t, received.DueDate.Value)
	assert.True(t, received.EstimatedHours.Set)
	assert.Nil(t, received.EstimatedHours.Value, "estimated_hours 0 means no estimate")
}

func TestUpdateCard_NullValue_LeavesFieldUnchanged(t *testing.T) {
	var received service.UpdateCardParams
	svc := &mock.MockBoardService{
		GetCardFn: func(ctx context.Context, cardID string) (db.Card, error) {
			return cardOwnedBy(ptr(validUserID), ptr(otherUserID)), nil
		},
		GetBoardIDByColumnFn: func(ctx context.Context, columnID string) (string, error) {
			return validBoardID, nil
		},
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			return "member", nil
		},
		UpdateCardFn: func(ctx context.Context, arg service.UpdateCardParams) (service.UpdateCardResult, error) {
			received = arg
			return service.UpdateCardResult{Card: db.Card{ID: arg.ID}}, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	body := strings.NewReader(`{"assignee_id": null}`)
	req := withUserID(httptest.NewRequest(http.MethodPatch, "/cards/"+validCardID, body), validUserID)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateCard)(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.False(t, received.AssigneeID.Set, "null must leave the assignee alone, not clear it")
}

func TestUpdateCard_InvalidClearableValues_Return400(t *testing.T) {
	for name, raw := range map[string]string{
		"assignee not uuid": `{"assignee_id": "nope"}`,
		"priority unknown":  `{"priority": "urgent"}`,
		"due_date bad":      `{"due_date": "17/09/2026"}`,
	} {
		t.Run(name, func(t *testing.T) {
			svc := &mock.MockBoardService{
				UpdateCardFn: func(ctx context.Context, arg service.UpdateCardParams) (service.UpdateCardResult, error) {
					t.Fatal("UpdateCard must not run when validation fails")
					return service.UpdateCardResult{}, nil
				},
			}
			h := NewBoardHandler(svc, nil, nil, nil)
			req := withUserID(httptest.NewRequest(http.MethodPatch, "/cards/"+validCardID, strings.NewReader(raw)), validUserID)
			req = chiCtx(req, "cardID", validCardID)
			w := httptest.NewRecorder()

			httputil.MakeHandler(h.UpdateCard)(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}
