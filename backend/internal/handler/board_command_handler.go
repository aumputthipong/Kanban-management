package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/middleware"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Broadcaster fans a message out to one board's WebSocket room. The hub implements
// it; handlers depend on the interface so a test can assert what was broadcast.
type Broadcaster interface {
	Broadcast(boardID string, message []byte)
}

// BoardCommandHandler is the REST write path for the kanban board: move, delete and
// done-toggle a card, plus column CRUD. It persists through the same service the WS
// handlers use and then broadcasts, so a dropped socket costs realtime, not the write.
type BoardCommandHandler struct {
	boardCmd     service.BoardCommandServicer
	boardService service.BoardServicer
	activity     service.ActivityRecorder
	broadcaster  Broadcaster
}

func NewBoardCommandHandler(
	boardCmd service.BoardCommandServicer,
	boardService service.BoardServicer,
	activity service.ActivityRecorder,
	broadcaster Broadcaster,
) *BoardCommandHandler {
	return &BoardCommandHandler{
		boardCmd:     boardCmd,
		boardService: boardService,
		activity:     activity,
		broadcaster:  broadcaster,
	}
}

// emit sends a message shaped exactly like the WebSocket handlers' own, so existing
// frontend listeners need no change. Best-effort like the audit row: the mutation has
// already committed and must not fail because fan-out did.
func emitTo(b Broadcaster, boardID, msgType string, payload map[string]any) {
	if b == nil {
		return
	}
	msg, err := json.Marshal(map[string]any{"type": msgType, "payload": payload})
	if err != nil {
		slog.Error("marshal broadcast failed", "type", msgType, "board_id", boardID, "err", err)
		return
	}
	b.Broadcast(boardID, msg)
}

func (h *BoardCommandHandler) emit(boardID, msgType string, payload map[string]any) {
	emitTo(h.broadcaster, boardID, msgType, payload)
}

// record writes the audit row and announces it, so the activity feed stays live.
// It uses the synchronous Record rather than RecordAsync because the broadcast
// needs the row's id and created_at, which only the sync call returns.
func (h *BoardCommandHandler) record(ctx context.Context, p service.RecordParams) {
	if h.activity == nil {
		return
	}
	act, err := h.activity.Record(ctx, p)
	if err != nil {
		slog.Error("record activity failed", "event_type", p.EventType, "board_id", p.BoardID, "err", err)
		return
	}
	emitTo(h.broadcaster, p.BoardID, "ACTIVITY_CREATED", activityPayload(act))
}

// activityPayload is the wire shape the activity feed consumes.
func activityPayload(act db.Activity) map[string]any {
	payload := json.RawMessage(act.Payload)
	if len(payload) == 0 {
		payload = json.RawMessage("{}")
	}
	return map[string]any{
		"id":          act.ID,
		"board_id":    act.BoardID,
		"actor_id":    act.ActorID,
		"event_type":  act.EventType,
		"entity_type": act.EntityType,
		"entity_id":   act.EntityID,
		"payload":     payload,
		"created_at":  act.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

// cardContext resolves the board owning cardID and gates membership on it.
func (h *BoardCommandHandler) cardContext(r *http.Request, cardID string) (string, string, *httputil.APIError) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		return "", "", httputil.NewAPIError(http.StatusUnauthorized, "Unauthorized", nil)
	}

	card, err := h.boardService.GetCard(r.Context(), cardID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", httputil.NewAPIError(http.StatusNotFound, "Not found", nil)
		}
		return "", "", httputil.NewAPIError(http.StatusInternalServerError, "Failed to load card", err)
	}

	boardID, err := h.boardService.GetBoardIDByColumn(r.Context(), card.ColumnID)
	if err != nil {
		return "", "", httputil.NewAPIError(http.StatusInternalServerError, "Failed to resolve board", err)
	}
	if _, apiErr := boardMembership(r.Context(), h.boardService, boardID, userID); apiErr != nil {
		return "", "", apiErr
	}
	return boardID, userID, nil
}

// columnContext resolves the board owning columnID and gates membership on it.
func (h *BoardCommandHandler) columnContext(r *http.Request, columnID string) (string, string, *httputil.APIError) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		return "", "", httputil.NewAPIError(http.StatusUnauthorized, "Unauthorized", nil)
	}

	boardID, err := h.boardService.GetBoardIDByColumn(r.Context(), columnID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", httputil.NewAPIError(http.StatusNotFound, "Not found", nil)
		}
		return "", "", httputil.NewAPIError(http.StatusInternalServerError, "Failed to resolve board", err)
	}
	if _, apiErr := boardMembership(r.Context(), h.boardService, boardID, userID); apiErr != nil {
		return "", "", apiErr
	}
	return boardID, userID, nil
}

func parseUUIDParam(r *http.Request, name, label string) (string, *httputil.APIError) {
	v := chi.URLParam(r, name)
	if _, err := uuid.Parse(v); err != nil {
		return "", httputil.NewAPIError(http.StatusBadRequest, "Invalid "+label+" format", err)
	}
	return v, nil
}
