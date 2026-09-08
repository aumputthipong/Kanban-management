package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/dto"
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
func (h *BoardCommandHandler) emit(boardID, msgType string, payload map[string]any) {
	if h.broadcaster == nil {
		return
	}
	msg, err := json.Marshal(map[string]any{"type": msgType, "payload": payload})
	if err != nil {
		slog.Error("marshal broadcast failed", "type", msgType, "board_id", boardID, "err", err)
		return
	}
	h.broadcaster.Broadcast(boardID, msg)
}

func (h *BoardCommandHandler) record(p service.RecordParams) {
	if h.activity == nil {
		return
	}
	h.activity.RecordAsync(p)
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

// MoveCard moves a card to a column and position. is_done and completed_at are derived
// from the target column's category, never taken from the request.
//
// @Summary  Move card
// @Tags     cards
// @Accept   json
// @Produce  json
// @Security CookieAuth
// @Param    cardID  path     string              true "Card UUID"
// @Param    payload body     dto.MoveCardRequest true "Target column + position"
// @Success  204
// @Failure  400     {object} httputil.ErrorResponse
// @Failure  404     {object} httputil.ErrorResponse
// @Router   /api/cards/{cardID}/move [patch]
func (h *BoardCommandHandler) MoveCard(w http.ResponseWriter, r *http.Request) error {
	cardID, apiErr := parseUUIDParam(r, "cardID", "card ID")
	if apiErr != nil {
		return apiErr
	}
	var req dto.MoveCardRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		return err
	}
	boardID, userID, apiErr := h.cardContext(r, cardID)
	if apiErr != nil {
		return apiErr
	}
	// The target column must be on the same board, or a member of board A could move a
	// card into board B by supplying one of its column ids.
	if err := h.boardCmd.VerifyColumnInBoard(r.Context(), req.ColumnID, boardID); err != nil {
		return httputil.NewAPIError(http.StatusNotFound, "Not found", nil)
	}

	result, err := h.boardCmd.MoveCard(r.Context(), cardID, req.ColumnID, req.Position)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to move card", err)
	}

	h.record(service.RecordParams{
		BoardID: boardID, ActorID: userID,
		EventType: service.EventCardMoved, EntityType: service.EntityCard, EntityID: &cardID,
		Payload: service.CardMovedPayload{Title: result.CardTitle, ToColumnID: req.ColumnID},
	})
	h.emit(boardID, "CARD_MOVED", map[string]any{
		"card_id":       cardID,
		"new_column_id": req.ColumnID,
		"position":      req.Position,
		"is_done":       result.IsDone,
		"completed_at":  result.CompletedAt,
	})
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// DeleteCard removes a card from its board.
//
// @Summary  Delete card
// @Tags     cards
// @Produce  json
// @Security CookieAuth
// @Param    cardID path string true "Card UUID"
// @Success  204
// @Failure  404    {object} httputil.ErrorResponse
// @Router   /api/cards/{cardID} [delete]
func (h *BoardCommandHandler) DeleteCard(w http.ResponseWriter, r *http.Request) error {
	cardID, apiErr := parseUUIDParam(r, "cardID", "card ID")
	if apiErr != nil {
		return apiErr
	}
	boardID, userID, apiErr := h.cardContext(r, cardID)
	if apiErr != nil {
		return apiErr
	}

	title, err := h.boardCmd.DeleteCard(r.Context(), cardID)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to delete card", err)
	}

	h.record(service.RecordParams{
		BoardID: boardID, ActorID: userID,
		EventType: service.EventCardDeleted, EntityType: service.EntityCard, EntityID: &cardID,
		Payload: service.CardDeletedPayload{Title: title},
	})
	h.emit(boardID, "CARD_DELETED", map[string]any{"card_id": cardID})
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// ToggleCardDone marks a card done or not done, moving it into the board's DONE column
// or back out. Broadcasts CARD_MOVED so clients reuse a single handler.
//
// @Summary  Toggle card done
// @Tags     cards
// @Accept   json
// @Produce  json
// @Security CookieAuth
// @Param    cardID  path     string                    true "Card UUID"
// @Param    payload body     dto.ToggleCardDoneRequest true "Desired done state"
// @Success  204
// @Failure  400     {object} httputil.ErrorResponse
// @Failure  404     {object} httputil.ErrorResponse
// @Router   /api/cards/{cardID}/done [patch]
func (h *BoardCommandHandler) ToggleCardDone(w http.ResponseWriter, r *http.Request) error {
	cardID, apiErr := parseUUIDParam(r, "cardID", "card ID")
	if apiErr != nil {
		return apiErr
	}
	var req dto.ToggleCardDoneRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		return err
	}
	boardID, userID, apiErr := h.cardContext(r, cardID)
	if apiErr != nil {
		return apiErr
	}

	result, err := h.boardCmd.ToggleCardDone(r.Context(), cardID, boardID, *req.IsDone)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to toggle card", err)
	}

	h.record(service.RecordParams{
		BoardID: boardID, ActorID: userID,
		EventType: service.EventCardDoneToggled, EntityType: service.EntityCard, EntityID: &cardID,
		Payload: map[string]any{"title": result.CardTitle, "is_done": *req.IsDone},
	})
	h.emit(boardID, "CARD_MOVED", map[string]any{
		"card_id":       cardID,
		"new_column_id": result.TargetColumnID,
		"position":      0,
		"is_done":       *req.IsDone,
		"completed_at":  result.CompletedAt,
	})
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// CreateColumn adds a column to a board.
//
// @Summary  Create column
// @Tags     columns
// @Accept   json
// @Produce  json
// @Security CookieAuth
// @Param    boardID path     string                  true "Board UUID"
// @Param    payload body     dto.CreateColumnRequest true "Title + category"
// @Success  201     {object} dto.ColumnResponse
// @Failure  400     {object} httputil.ErrorResponse
// @Failure  404     {object} httputil.ErrorResponse
// @Router   /api/boards/{boardID}/columns [post]
func (h *BoardCommandHandler) CreateColumn(w http.ResponseWriter, r *http.Request) error {
	boardID, apiErr := parseUUIDParam(r, "boardID", "board ID")
	if apiErr != nil {
		return apiErr
	}
	var req dto.CreateColumnRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		return err
	}
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		return httputil.NewAPIError(http.StatusUnauthorized, "Unauthorized", nil)
	}

	col, err := h.boardCmd.CreateColumn(r.Context(), boardID, req.Title, req.Category, req.Color)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to create column", err)
	}

	h.record(service.RecordParams{
		BoardID: boardID, ActorID: userID,
		EventType: service.EventColumnCreated, EntityType: service.EntityColumn, EntityID: &col.ID,
		Payload: service.ColumnCreatedPayload{Title: col.Title},
	})
	h.emit(boardID, "COLUMN_CREATED", map[string]any{
		"id":       col.ID,
		"board_id": boardID,
		"title":    col.Title,
		"position": col.Position,
		"category": col.Category,
		"color":    col.Color,
	})
	httputil.RespondJSON(w, http.StatusCreated, dto.ColumnResponse{
		ID:       col.ID,
		Title:    col.Title,
		Position: col.Position,
		Category: col.Category,
		Color:    col.Color,
	})
	return nil
}

// UpdateColumn sets a column's title, category and colour. Title and category are both
// required: the underlying SQL assigns them outright rather than through COALESCE.
//
// @Summary  Update column
// @Tags     columns
// @Accept   json
// @Produce  json
// @Security CookieAuth
// @Param    columnID path     string                  true "Column UUID"
// @Param    payload  body     dto.UpdateColumnRequest true "Title + category (+ colour)"
// @Success  204
// @Failure  400      {object} httputil.ErrorResponse
// @Failure  404      {object} httputil.ErrorResponse
// @Router   /api/columns/{columnID} [patch]
func (h *BoardCommandHandler) UpdateColumn(w http.ResponseWriter, r *http.Request) error {
	columnID, apiErr := parseUUIDParam(r, "columnID", "column ID")
	if apiErr != nil {
		return apiErr
	}
	var req dto.UpdateColumnRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		return err
	}
	boardID, userID, apiErr := h.columnContext(r, columnID)
	if apiErr != nil {
		return apiErr
	}

	if err := h.boardCmd.UpdateColumn(r.Context(), service.UpdateColumnParams{
		ID: columnID, Title: req.Title, Category: req.Category, Color: req.Color,
	}); err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to update column", err)
	}

	h.record(service.RecordParams{
		BoardID: boardID, ActorID: userID,
		EventType: service.EventColumnRenamed, EntityType: service.EntityColumn, EntityID: &columnID,
		Payload: service.ColumnRenamedPayload{NewTitle: req.Title},
	})
	h.emit(boardID, "COLUMN_UPDATED", map[string]any{
		"column_id": columnID,
		"title":     req.Title,
		"category":  req.Category,
		"color":     req.Color,
	})
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// DeleteColumn removes a column and the cards in it.
//
// @Summary  Delete column
// @Tags     columns
// @Produce  json
// @Security CookieAuth
// @Param    columnID path string true "Column UUID"
// @Success  204
// @Failure  404      {object} httputil.ErrorResponse
// @Router   /api/columns/{columnID} [delete]
func (h *BoardCommandHandler) DeleteColumn(w http.ResponseWriter, r *http.Request) error {
	columnID, apiErr := parseUUIDParam(r, "columnID", "column ID")
	if apiErr != nil {
		return apiErr
	}
	boardID, userID, apiErr := h.columnContext(r, columnID)
	if apiErr != nil {
		return apiErr
	}

	if err := h.boardCmd.DeleteColumn(r.Context(), columnID); err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to delete column", err)
	}

	h.record(service.RecordParams{
		BoardID: boardID, ActorID: userID,
		EventType: service.EventColumnDeleted, EntityType: service.EntityColumn, EntityID: &columnID,
		Payload: service.ColumnDeletedPayload{},
	})
	h.emit(boardID, "COLUMN_DELETED", map[string]any{"column_id": columnID})
	w.WriteHeader(http.StatusNoContent)
	return nil
}
