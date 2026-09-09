package handler

import (
	"net/http"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/core"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/dto"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/middleware"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
)

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

	h.record(r.Context(), service.RecordParams{
		BoardID: boardID, ActorID: userID,
		EventType: service.EventCardMoved, EntityType: service.EntityCard, EntityID: &cardID,
		Payload: service.CardMovedPayload{Title: result.CardTitle, ToColumnID: req.ColumnID},
	})
	h.emit(boardID, core.WSCardMoved, map[string]any{
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

	h.record(r.Context(), service.RecordParams{
		BoardID: boardID, ActorID: userID,
		EventType: service.EventCardDeleted, EntityType: service.EntityCard, EntityID: &cardID,
		Payload: service.CardDeletedPayload{Title: title},
	})
	h.emit(boardID, core.WSCardDeleted, map[string]any{"card_id": cardID})
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

	h.record(r.Context(), service.RecordParams{
		BoardID: boardID, ActorID: userID,
		EventType: service.EventCardDoneToggled, EntityType: service.EntityCard, EntityID: &cardID,
		Payload: map[string]any{"title": result.CardTitle, "is_done": *req.IsDone},
	})
	h.emit(boardID, core.WSCardMoved, map[string]any{
		"card_id":       cardID,
		"new_column_id": result.TargetColumnID,
		"position":      0,
		"is_done":       *req.IsDone,
		"completed_at":  result.CompletedAt,
	})
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// CreateCard creates a card, optionally with a description and subtasks, in one
// transaction. Position 0 appends. It calls the same service the WS path does, so
// both produce identical rows.
//
// @Summary  Create card
// @Tags     cards
// @Accept   json
// @Produce  json
// @Security CookieAuth
// @Param    payload body     dto.CreateCardRequest true "Column + title (+ optional fields)"
// @Success  201     {object} dto.CardResponse
// @Failure  400     {object} httputil.ErrorResponse
// @Failure  404     {object} httputil.ErrorResponse
// @Router   /api/cards [post]
func (h *BoardCommandHandler) CreateCard(w http.ResponseWriter, r *http.Request) error {
	var req dto.CreateCardRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		return err
	}
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		return httputil.NewAPIError(http.StatusUnauthorized, "Unauthorized", nil)
	}

	boardID, err := h.boardService.GetBoardIDByColumn(r.Context(), req.ColumnID)
	if err != nil {
		return httputil.NewAPIError(http.StatusNotFound, "Not found", nil)
	}
	if _, apiErr := boardMembership(r.Context(), h.boardService, boardID, userID); apiErr != nil {
		return apiErr
	}

	priority := ""
	if req.Priority != nil {
		priority = *req.Priority
	}
	card, subtasks, err := h.boardCmd.CreateCardWS(
		r.Context(), req.ColumnID, userID, req.Title, priority, req.Position,
		req.AssigneeID, req.DueDate, req.Description, req.Subtasks,
	)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to create card", err)
	}

	subPayload := make([]map[string]any, 0, len(subtasks))
	for _, st := range subtasks {
		subPayload = append(subPayload, map[string]any{
			"id": st.ID, "title": st.Title, "is_done": st.IsDone, "position": st.Position,
		})
	}

	h.record(r.Context(), service.RecordParams{
		BoardID: boardID, ActorID: userID,
		EventType: service.EventCardCreated, EntityType: service.EntityCard, EntityID: &card.ID,
		Payload: service.CardCreatedPayload{Title: card.Title, ColumnID: card.ColumnID},
	})
	h.emit(boardID, core.WSCardCreated, map[string]any{
		"id":                 card.ID,
		"column_id":          card.ColumnID,
		"title":              card.Title,
		"position":           card.Position,
		"priority":           card.Priority,
		"created_by":         card.CreatedBy,
		"assignee_id":        req.AssigneeID,
		"due_date":           req.DueDate,
		"description":        req.Description,
		"subtasks":           subPayload,
		"total_subtasks":     len(subPayload),
		"completed_subtasks": 0,
	})
	// Respond with the snake_case DTO the rest of the API uses. The raw sqlc row
	// marshals as PascalCase, which no client reads.
	httputil.RespondJSON(w, http.StatusCreated, dto.CardResponse{
		ID:                 card.ID,
		ColumnID:           card.ColumnID,
		Title:              card.Title,
		Description:        card.Description,
		Position:           card.Position,
		DueDate:            req.DueDate,
		AssigneeID:         card.AssigneeID,
		Priority:           card.Priority,
		CreatedBy:          card.CreatedBy,
		TotalSubtasks:      int64(len(subPayload)),
		CompletedSubtasks:  0,
		Tags:               []dto.TagResponse{},
		AcceptanceCriteria: card.AcceptanceCriteria,
		ImplementationNote: card.ImplementationNote,
	})
	return nil
}
