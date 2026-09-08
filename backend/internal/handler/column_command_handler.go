package handler

import (
	"net/http"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/dto"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/middleware"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
)

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
