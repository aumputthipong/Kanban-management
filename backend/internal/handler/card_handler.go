package handler

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/core"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/dto"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/mapper"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/middleware"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/util"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// requireBoardMembership checks that userID is a member of boardID and returns
// the role, mirroring the RequireBoardMember middleware: non-members get 404.
func (h *BoardHandler) requireBoardMembership(r *http.Request, boardID, userID string) (core.BoardRole, *httputil.APIError) {
	role, err := h.boardService.GetBoardMemberRole(r.Context(), boardID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", httputil.NewAPIError(http.StatusNotFound, "Not found", nil)
		}
		return "", httputil.NewAPIError(http.StatusInternalServerError, "Failed to check board access", err)
	}
	return core.BoardRole(role), nil
}

// CreateCard creates a card in a column. Caller must be a member of the
// owning board.
//
// @Summary  Create card
// @Tags     cards
// @Accept   json
// @Produce  json
// @Security CookieAuth
// @Param    payload body     dto.CreateCardRequest true  "Column + title (+ optional fields)"
// @Success  201     {object} dto.CardResponse
// @Failure  400     {object} httputil.ErrorResponse
// @Failure  404     {object} httputil.ErrorResponse "column not found"
// @Router   /api/cards [post]
func (h *BoardHandler) CreateCard(w http.ResponseWriter, r *http.Request) error {
	var req dto.CreateCardRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		return err
	}

	if _, err := uuid.Parse(req.ColumnID); err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid column ID", err)
	}

	userIDStr, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userIDStr == "" {
		return httputil.NewAPIError(http.StatusUnauthorized, "Unauthorized", nil)
	}

	boardID, err := h.boardService.GetBoardIDByColumn(r.Context(), req.ColumnID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.NewAPIError(http.StatusNotFound, "Column not found", nil)
		}
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to resolve column", err)
	}
	if _, apiErr := h.requireBoardMembership(r, boardID, userIDStr); apiErr != nil {
		return apiErr
	}

	card, err := h.boardService.CreateCard(r.Context(), db.CreateCardParams{
		ColumnID:   req.ColumnID,
		Title:      req.Title,
		Position:   0,
		DueDate:    util.PtrStringToTimePtr(req.DueDate),
		AssigneeID: req.AssigneeID,
		Priority:   req.Priority,
		CreatedBy:  util.StringToPtr(userIDStr),
	})
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to create card", err)
	}

	httputil.RespondJSON(w, http.StatusCreated, card)
	return nil
}

// UpdateCard partially updates a card. Used for inline edits + drag-and-drop
// (column_id and position changes broadcast CARD_MOVED).
//
// @Summary  Update card
// @Tags     cards
// @Accept   json
// @Produce  json
// @Security CookieAuth
// @Param    cardID  path     string                true  "Card UUID"
// @Param    payload body     dto.UpdateCardRequest true  "Fields to update"
// @Success  200     {object} dto.CardResponse
// @Failure  400     {object} httputil.ErrorResponse
// @Failure  404     {object} httputil.ErrorResponse
// @Router   /api/cards/{cardID} [patch]
func (h *BoardHandler) UpdateCard(w http.ResponseWriter, r *http.Request) error {
	cardIDStr := chi.URLParam(r, "cardID")
	if _, err := uuid.Parse(cardIDStr); err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid card ID format", err)
	}

	var req dto.UpdateCardRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		return err
	}

	userIDStr, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userIDStr == "" {
		return httputil.NewAPIError(http.StatusUnauthorized, "Unauthorized", nil)
	}

	existing, err := h.boardService.GetCard(r.Context(), cardIDStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
			return httputil.NewAPIError(http.StatusNotFound, "Card not found", nil)
		}
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to load card", err)
	}

	boardID, err := h.boardService.GetBoardIDByColumn(r.Context(), existing.ColumnID)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to resolve board", err)
	}
	role, apiErr := h.requireBoardMembership(r, boardID, userIDStr)
	if apiErr != nil {
		return apiErr
	}

	// Creator / assignee can always edit their own card; everyone else needs
	// manager or above.
	isOwnCard := (existing.CreatedBy != nil && *existing.CreatedBy == userIDStr) ||
		(existing.AssigneeID != nil && *existing.AssigneeID == userIDStr)
	if !isOwnCard && role != core.RoleOwner && role != core.RoleManager {
		return httputil.NewAPIError(http.StatusForbidden, "You do not have permission to edit this card", nil)
	}

	// PATCH semantics: nil means "leave unchanged", but UpdateCard's SQL overwrites these
	// columns directly (no COALESCE), so the handler must merge each omitted field from
	// the existing row. This bit My Work snooze, which sends only { due_date }: every
	// other column was wiped and the card left the inbox when assignee_id went NULL.
	title := existing.Title
	if req.Title != nil {
		title = *req.Title
	}
	description := existing.Description
	if req.Description != nil {
		description = req.Description
	}
	dueDate := existing.DueDate
	if req.DueDate != nil {
		dueDate = util.PtrStringToTimePtr(req.DueDate)
	}
	assigneeID := existing.AssigneeID
	if req.AssigneeID != nil {
		assigneeID = req.AssigneeID
	}
	priority := existing.Priority
	if req.Priority != nil {
		priority = req.Priority
	}
	estimatedHours := util.PgNumericToFloat64Ptr(existing.EstimatedHours)
	if req.EstimatedHours != nil {
		estimatedHours = req.EstimatedHours
	}

	card, err := h.boardService.UpdateCard(r.Context(), service.UpdateCardParams{
		ID:                 cardIDStr,
		Title:              title,
		Description:        description,
		DueDate:            dueDate,
		AssigneeID:         assigneeID,
		Priority:           priority,
		EstimatedHours:     estimatedHours,
		TagIDs:             req.TagIDs,
		AcceptanceCriteria: req.AcceptanceCriteria,
		ImplementationNote: req.ImplementationNote,
	})
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to update card", err)
	}

	httputil.RespondJSON(w, http.StatusOK, card)
	return nil
}

// GetCard returns a single card with its subtasks + tags hydrated.
//
// @Summary  Get card detail
// @Tags     cards
// @Produce  json
// @Security CookieAuth
// @Param    cardID path     string true "Card UUID"
// @Success  200    {object} dto.CardResponse
// @Failure  404    {object} httputil.ErrorResponse
// @Router   /api/cards/{cardID} [get]
func (h *BoardHandler) GetCard(w http.ResponseWriter, r *http.Request) error {
	cardIDStr := chi.URLParam(r, "cardID")
	if _, err := uuid.Parse(cardIDStr); err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid card ID format", err)
	}

	detail, err := h.boardService.GetCardDetail(r.Context(), cardIDStr)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			return httputil.NewAPIError(http.StatusNotFound, "Card not found", nil)
		}
		return httputil.NewAPIError(http.StatusInternalServerError, "Internal server error", err)
	}

	httputil.RespondJSON(w, http.StatusOK, mapper.ToCardDetailResponse(detail))
	return nil
}
