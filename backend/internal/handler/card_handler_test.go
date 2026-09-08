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
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service/mock"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// jsonBody marshals v to JSON for use as a request body.
func jsonBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

// otherUserID is a member of the same board as validUserID, used to verify
// the "edit own card" carve-out doesn't leak across users.
const otherUserID = "11111111-2222-3333-4444-555555555555"

// ────────────────────────────────────────────────
// CreateCard
// ────────────────────────────────────────────────

// ────────────────────────────────────────────────
// UpdateCard
// ────────────────────────────────────────────────

// cardOwnedBy returns a db.Card with the given creator/assignee, for setting
// up the "is this user allowed to edit?" branch in UpdateCard.
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
		UpdateCardFn: func(ctx context.Context, arg service.UpdateCardParams) (db.Card, error) {
			return db.Card{ID: arg.ID, Title: arg.Title}, nil
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
	// Member role + caller is the creator → carve-out allows edit.
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
		UpdateCardFn: func(ctx context.Context, arg service.UpdateCardParams) (db.Card, error) {
			return db.Card{ID: arg.ID, Title: arg.Title}, nil
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
	// Member role + caller is assignee (but not creator) → still allowed.
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
		UpdateCardFn: func(ctx context.Context, arg service.UpdateCardParams) (db.Card, error) {
			return db.Card{ID: arg.ID}, nil
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

// TestUpdateCard_MemberEditsOthersCard_Returns403 — the key permission test:
// a plain member must NOT be able to edit a card they neither created nor
// were assigned. Manager+ would pass, this caller is just "member".
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
		UpdateCardFn: func(ctx context.Context, arg service.UpdateCardParams) (db.Card, error) {
			t.Fatal("UpdateCard must not run when permission check fails")
			return db.Card{}, nil
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

// TestUpdateCard_NonMember_Returns404 — anti-enumeration applies here too:
// knowing a valid card ID must not reveal whether you're a member of its board.
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

// Pins the silent-clobber case AGENTS.md flags: omitting `title` from a PATCH must
// leave it intact. UpdateCard's SQL overwrites the column directly, so the handler merge
// is what preserves it — this asserts that merge at the handler boundary.
func TestUpdateCard_PATCHSemantics_OmittedTitle(t *testing.T) {
	creator := ptr(validUserID)
	var received service.UpdateCardParams
	svc := &mock.MockBoardService{
		GetCardFn: func(ctx context.Context, cardID string) (db.Card, error) {
			return cardOwnedBy(creator, nil), nil // Title: "Existing"
		},
		GetBoardIDByColumnFn: func(ctx context.Context, columnID string) (string, error) {
			return validBoardID, nil
		},
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			return "member", nil
		},
		UpdateCardFn: func(ctx context.Context, arg service.UpdateCardParams) (db.Card, error) {
			received = arg
			return db.Card{ID: arg.ID}, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	// Only description supplied; title omitted entirely.
	body := map[string]any{"description": "hello"}
	req := withUserID(httptest.NewRequest(http.MethodPatch, "/cards/"+validCardID, jsonBody(t, body)), validUserID)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateCard)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Existing", received.Title, "omitted title must be preserved from the existing row, not clobbered to empty")
	require.NotNil(t, received.Description)
	assert.Equal(t, "hello", *received.Description)
}

// The My Work snooze regression: a PATCH touching only due_date must not wipe assignee,
// priority, description or estimated_hours. Before the handler merge, the overwrite-style
// SQL nulled every omitted column and the card left its owner's inbox.
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
			return "member", nil // not manager — relies on the assignee carve-out
		},
		UpdateCardFn: func(ctx context.Context, arg service.UpdateCardParams) (db.Card, error) {
			received = arg
			return db.Card{ID: arg.ID}, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	// Snooze: only due_date supplied, exactly like lib/myWorkApi snoozeCardDueDate.
	body := map[string]any{"due_date": "2026-06-10"}
	req := withUserID(httptest.NewRequest(http.MethodPatch, "/cards/"+validCardID, jsonBody(t, body)), validUserID)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateCard)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Keep me", received.Title, "title must survive a due_date-only patch")
	require.NotNil(t, received.AssigneeID, "assignee must not be nulled — card would vanish from My Work")
	assert.Equal(t, validUserID, *received.AssigneeID)
	require.NotNil(t, received.Description)
	assert.Equal(t, "keep this desc", *received.Description)
	require.NotNil(t, received.Priority)
	assert.Equal(t, "high", *received.Priority)
	require.NotNil(t, received.DueDate, "due_date must be applied")
}

// TestUpdateCard_PATCHSemantics_EmptyTitleRejected verifies the validator
// guard: title has `omitempty,min=1,max=200`, so explicitly sending "" must
// be rejected (would otherwise blank out the title). AGENTS.md: "" on a
// required column → 400.
func TestUpdateCard_PATCHSemantics_EmptyTitleRejected(t *testing.T) {
	creator := ptr(validUserID)
	svc := &mock.MockBoardService{
		GetCardFn: func(ctx context.Context, cardID string) (db.Card, error) {
			return cardOwnedBy(creator, nil), nil
		},
		UpdateCardFn: func(ctx context.Context, arg service.UpdateCardParams) (db.Card, error) {
			t.Fatal("UpdateCard must not be called when validation fails")
			return db.Card{}, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	// Send raw JSON because the helper wraps map[string]any.
	body := strings.NewReader(`{"title": ""}`)
	req := withUserID(httptest.NewRequest(http.MethodPatch, "/cards/"+validCardID, body), validUserID)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateCard)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ────────────────────────────────────────────────
// GetCard
// ────────────────────────────────────────────────

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
	// Enriched response is now snake_case dto.CardDetailResponse.
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
	// Handler maps both pgx.ErrNoRows and sql.ErrNoRows to 404.
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

// TestUpdateCard_RespondsWithSnakeCase pins the wire shape. The handler used to
// respond with the raw sqlc row, which marshals as PascalCase — no client reads
// that, and the Swagger annotation promises dto.CardResponse.
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
		UpdateCardFn: func(ctx context.Context, arg service.UpdateCardParams) (db.Card, error) {
			return db.Card{ID: arg.ID, ColumnID: validColumnID, Title: arg.Title}, nil
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

// Guards a trap: every CARD_UPDATED field must come from the updated row, never the
// request. A request-sourced field is null whenever the caller omitted it, and the
// receiving store spreads the payload over its copy — the value vanishes everywhere.
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
		UpdateCardFn: func(ctx context.Context, arg service.UpdateCardParams) (db.Card, error) {
			return db.Card{ID: arg.ID, ColumnID: validColumnID, Title: arg.Title, DueDate: arg.DueDate}, nil
		},
	}
	bc := &mock.MockBroadcaster{}
	h := NewBoardHandler(svc, nil, nil, bc)

	// Body carries no due_date, so the handler must merge the stored one.
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
