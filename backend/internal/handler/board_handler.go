package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/dto"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/mapper"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/middleware"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/util"
	"github.com/google/uuid"
)

type BoardHandler struct {
	boardService    service.BoardServicer
	settingsService service.UserSettingsServicer
	activity        service.ActivityRecorder
}

// NewBoardHandler wires the board handler. settingsService is required for
// the /my-tasks (My Work) endpoint, which reads the caller's timezone and
// "show all cards" preference. activity is used by CompleteMyTask to log a
// card.done_toggled event; passing nil keeps the mutation working but skips
// the audit row, which is fine for unit tests that don't assert on the feed.
func NewBoardHandler(
	boardService service.BoardServicer,
	settingsService service.UserSettingsServicer,
	activity service.ActivityRecorder,
) *BoardHandler {
	return &BoardHandler{
		boardService:    boardService,
		settingsService: settingsService,
		activity:        activity,
	}
}

// GetAllBoards lists every board the caller is a member of.
//
// @Summary  List my boards
// @Tags     boards
// @Produce  json
// @Security CookieAuth
// @Success  200 {array}  dto.BoardSummaryResponse
// @Failure  401 {object} httputil.ErrorResponse
// @Router   /api/boards [get]
func (h *BoardHandler) GetAllBoards(w http.ResponseWriter, r *http.Request) error {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		return httputil.NewAPIError(http.StatusUnauthorized, "Unauthorized", nil)
	}
	boards, err := h.boardService.GetAllBoards(r.Context(), userID)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to fetch boards", err)
	}

	result := make([]dto.BoardSummaryResponse, 0, len(boards))
	for _, b := range boards {
		result = append(result, dto.BoardSummaryResponse{
			ID:             b.ID,
			Title:          b.Title,
			Description:    b.Description,
			Color:          b.Color,
			Icon:           b.Icon,
			CreatedAt:      b.CreatedAt,
			UpdatedAt:      b.UpdatedAt,
			LastAccessedAt: b.LastAccessedAt,
			TotalCards:     b.TotalCards,
			DoneCards:      b.DoneCards,
			Members:        b.Members,
		})
	}
	httputil.RespondJSON(w, http.StatusOK, result)
	return nil
}

// GetBoardData returns the columns + cards for one board (full board view).
//
// @Summary  Get board contents
// @Tags     boards
// @Produce  json
// @Security CookieAuth
// @Param    boardID path     string  true  "Board UUID"
// @Success  200     {array}  dto.ColumnResponse
// @Failure  400     {object} httputil.ErrorResponse
// @Failure  404     {object} httputil.ErrorResponse "not a member or board not found"
// @Router   /api/boards/{boardID} [get]
func (h *BoardHandler) GetBoardData(w http.ResponseWriter, r *http.Request) error {
	boardID, err := httputil.GetUUIDParam(r, "boardID")
	if err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid board ID format", err)
	}

	columns, err := h.boardService.GetBoardWithCards(r.Context(), boardID)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to fetch board data", err)
	}

	// Bump "recently opened" sort for this user. Detached from r.Context()
	// because the request finishes before the write — using r.Context() would
	// race the cancel. SQL-level throttle (5 min) keeps the write rate sane.
	if userID, ok := r.Context().Value(middleware.UserIDKey).(string); ok && userID != "" {
		go func(boardID, userID string) {
			defer func() {
				if rec := recover(); rec != nil {
					slog.Warn("touch board access panic", "err", rec)
				}
			}()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := h.boardService.TouchBoardMemberAccess(ctx, boardID, userID); err != nil {
				slog.Warn("touch board access failed", "board_id", boardID, "user_id", userID, "err", err)
			}
		}(boardID, userID)
	}

	httputil.RespondJSON(w, http.StatusOK, mapper.ToColumnResponses(columns))
	return nil
}

// CreateBoard creates a new board owned by the caller.
//
// @Summary  Create board
// @Tags     boards
// @Accept   json
// @Produce  json
// @Security CookieAuth
// @Param    payload body     dto.CreateBoardRequest true  "Board title"
// @Success  201     {object} map[string]string      "id"
// @Failure  400     {object} httputil.ErrorResponse
// @Router   /api/boards [post]
func (h *BoardHandler) CreateBoard(w http.ResponseWriter, r *http.Request) error {
	var req dto.CreateBoardRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		return err
	}

	userIDStr, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userIDStr == "" {
		return httputil.NewAPIError(http.StatusUnauthorized, "Unauthorized", nil)
	}
	// Validate the UUID format of userID from the token.
	if _, err := uuid.Parse(userIDStr); err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Invalid user ID in token", err)
	}

	boardID, err := h.boardService.CreateBoard(r.Context(), req.Title, req.Description, req.Color, req.Icon, userIDStr)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to create board", err)
	}

	httputil.RespondJSON(w, http.StatusCreated, map[string]string{"id": boardID})
	return nil
}

// UpdateBoard updates a board's title and/or budget. Manager+ only.
//
// @Summary  Update board
// @Tags     boards
// @Accept   json
// @Produce  json
// @Security CookieAuth
// @Param    boardID path     string                 true  "Board UUID"
// @Param    payload body     dto.UpdateBoardRequest true  "Fields to update"
// @Success  200     {object} dto.BoardSummaryResponse
// @Failure  400     {object} httputil.ErrorResponse
// @Failure  403     {object} httputil.ErrorResponse "not a manager"
// @Router   /api/boards/{boardID} [patch]
func (h *BoardHandler) UpdateBoard(w http.ResponseWriter, r *http.Request) error {
	boardID, err := httputil.GetUUIDParam(r, "boardID")
	if err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid or missing board ID", err)
	}

	var req dto.UpdateBoardRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		return err
	}

	updatedBoard, err := h.boardService.UpdateBoard(r.Context(), boardID, req.Title, req.Budget, req.Description, req.Color, req.Icon)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to update board", err)
	}

	httputil.RespondJSON(w, http.StatusOK, dto.BoardResponse{
		ID:          updatedBoard.ID,
		Title:       updatedBoard.Title,
		Description: updatedBoard.Description,
		Color:       updatedBoard.Color,
		Icon:        updatedBoard.Icon,
		Budget:      util.PgNumericToFloat64Ptr(updatedBoard.Budget),
	})
	return nil
}

// StashBoard stashes a board (recoverable soft-delete; sets deleted_at).
// Owner only.
//
// @Summary  Stash board
// @Tags     boards
// @Security CookieAuth
// @Param    boardID path string true "Board UUID"
// @Success  204
// @Failure  403 {object} httputil.ErrorResponse "not the owner"
// @Router   /api/boards/{boardID} [delete]
func (h *BoardHandler) StashBoard(w http.ResponseWriter, r *http.Request) error {
	boardID, err := httputil.GetUUIDParam(r, "boardID")
	if err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid board ID format", err)
	}

	if err := h.boardService.StashBoard(r.Context(), boardID); err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to stash board", err)
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

// GetStashedBoards lists stashed boards owned by the caller.
//
// @Summary  List stashed boards
// @Tags     stash
// @Produce  json
// @Security CookieAuth
// @Success  200 {array} dto.StashedBoardDTO
// @Router   /api/stash [get]
func (h *BoardHandler) GetStashedBoards(w http.ResponseWriter, r *http.Request) error {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		return httputil.NewAPIError(http.StatusUnauthorized, "Unauthorized", nil)
	}
	boards, err := h.boardService.GetStashedBoards(r.Context(), userID)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to get stashed boards", err)
	}

	httputil.RespondJSON(w, http.StatusOK, mapper.ToStashedBoardDTOs(boards))
	return nil
}

// HardDelete permanently removes an stashed board. Owner only.
//
// @Summary  Permanently delete stashed board
// @Tags     stash
// @Security CookieAuth
// @Param    boardID path string true "Board UUID"
// @Success  204
// @Failure  403 {object} httputil.ErrorResponse
// @Router   /api/stash/{boardID} [delete]
func (h *BoardHandler) HardDelete(w http.ResponseWriter, r *http.Request) error {
	boardID, err := httputil.GetUUIDParam(r, "boardID")
	if err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid board ID format", err)
	}

	if err := h.boardService.HardDeleteBoard(r.Context(), boardID); err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Delete failed", err)
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

// RestoreBoard restores an stashed board. Owner only.
//
// @Summary  Restore stashed board
// @Tags     stash
// @Security CookieAuth
// @Param    boardID path string true "Board UUID"
// @Success  204
// @Router   /api/stash/{boardID}/restore [patch]
func (h *BoardHandler) RestoreBoard(w http.ResponseWriter, r *http.Request) error {
	boardID, err := httputil.GetUUIDParam(r, "boardID")
	if err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid board ID format", err)
	}

	if err := h.boardService.RestoreBoard(r.Context(), boardID); err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to restore board", err)
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}
