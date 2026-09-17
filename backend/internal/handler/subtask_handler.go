package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/core"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/dto"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/mapper"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/middleware"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type SubtaskHandler struct {
	subtaskService service.SubtaskServicer
	boardService   service.BoardServicer
	broadcaster    Broadcaster
}

func NewSubtaskHandler(subtaskService service.SubtaskServicer, boardService service.BoardServicer, broadcaster Broadcaster) *SubtaskHandler {
	return &SubtaskHandler{subtaskService: subtaskService, boardService: boardService, broadcaster: broadcaster}
}

type subtaskCard struct {
	cardID  string
	boardID string
	canEdit bool
}

// cardAccess resolves the card's board and gates membership on it: an unknown card and
// a board the caller is not on both 404 (docs/adr/0004). canEdit follows canEditCard.
func (h *SubtaskHandler) cardAccess(r *http.Request) (subtaskCard, *httputil.APIError) {
	cardID := chi.URLParam(r, "cardID")
	if _, err := uuid.Parse(cardID); err != nil {
		return subtaskCard{}, httputil.NewAPIError(http.StatusBadRequest, "Invalid card ID", err)
	}
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		return subtaskCard{}, httputil.NewAPIError(http.StatusUnauthorized, "Unauthorized", nil)
	}
	card, err := h.boardService.GetCard(r.Context(), cardID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return subtaskCard{}, httputil.NewAPIError(http.StatusNotFound, "Not found", nil)
		}
		return subtaskCard{}, httputil.NewAPIError(http.StatusInternalServerError, "Failed to load card", err)
	}
	boardID, err := h.boardService.GetBoardIDByColumn(r.Context(), card.ColumnID)
	if err != nil {
		return subtaskCard{}, httputil.NewAPIError(http.StatusInternalServerError, "Failed to resolve board", err)
	}
	role, apiErr := boardMembership(r.Context(), h.boardService, boardID, userID)
	if apiErr != nil {
		return subtaskCard{}, apiErr
	}
	return subtaskCard{cardID: cardID, boardID: boardID, canEdit: canEditCard(card, userID, role)}, nil
}

// cardForEdit is cardAccess plus the edit rule. A member without edit rights gets 403,
// not 404: they can already see the card, so there is nothing left to hide.
func (h *SubtaskHandler) cardForEdit(r *http.Request) (subtaskCard, *httputil.APIError) {
	card, apiErr := h.cardAccess(r)
	if apiErr != nil {
		return subtaskCard{}, apiErr
	}
	if !card.canEdit {
		return subtaskCard{}, httputil.NewAPIError(http.StatusForbidden, "You do not have permission to edit this card", nil)
	}
	return card, nil
}

// subtaskOnCard 404s a subtask that belongs to a different card, so access to one card
// cannot be used to reach a subtask on a board the caller is not a member of.
func (h *SubtaskHandler) subtaskOnCard(r *http.Request, cardID string) (db.CardSubtask, *httputil.APIError) {
	subtaskID := chi.URLParam(r, "subtaskID")
	if _, err := uuid.Parse(subtaskID); err != nil {
		return db.CardSubtask{}, httputil.NewAPIError(http.StatusBadRequest, "Invalid subtask ID", err)
	}
	subtask, err := h.subtaskService.GetSubtaskByID(r.Context(), subtaskID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.CardSubtask{}, httputil.NewAPIError(http.StatusNotFound, "Subtask not found", nil)
		}
		return db.CardSubtask{}, httputil.NewAPIError(http.StatusInternalServerError, "Failed to load subtask", err)
	}
	if subtask.CardID != cardID {
		return db.CardSubtask{}, httputil.NewAPIError(http.StatusNotFound, "Subtask not found", nil)
	}
	return subtask, nil
}

// broadcastSubtasks sends the card's full subtask list, so a receiver that applies it
// twice, or out of order with its own optimistic edit, still ends up correct.
func (h *SubtaskHandler) broadcastSubtasks(r *http.Request, boardID, cardID string) {
	subtasks, err := h.subtaskService.GetSubtasksByCardID(r.Context(), cardID)
	if err != nil {
		slog.Error("load subtasks for broadcast failed", "card_id", cardID, "err", err)
		return
	}
	emitTo(h.broadcaster, boardID, core.WSCardSubtasksUpdated, map[string]any{
		"card_id":  cardID,
		"subtasks": mapper.ToSubtaskResponses(subtasks),
	})
}

func (h *SubtaskHandler) CreateSubtask(w http.ResponseWriter, r *http.Request) error {
	var payload dto.SubtaskRequest
	if err := httputil.DecodeAndValidate(r, &payload); err != nil {
		return err
	}
	card, apiErr := h.cardForEdit(r)
	if apiErr != nil {
		return apiErr
	}

	subtask, err := h.subtaskService.CreateSubtask(r.Context(), db.CreateSubtaskParams{
		CardID:   card.cardID,
		Title:    payload.Title,
		Position: payload.Position,
	})
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to create subtask", err)
	}

	h.broadcastSubtasks(r, card.boardID, card.cardID)
	httputil.RespondJSON(w, http.StatusCreated, mapper.ToSubtaskResponse(subtask))
	return nil
}

func (h *SubtaskHandler) GetSubtasks(w http.ResponseWriter, r *http.Request) error {
	card, apiErr := h.cardAccess(r)
	if apiErr != nil {
		return apiErr
	}

	subtasks, err := h.subtaskService.GetSubtasksByCardID(r.Context(), card.cardID)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to retrieve subtasks", err)
	}

	httputil.RespondJSON(w, http.StatusOK, mapper.ToSubtaskResponses(subtasks))
	return nil
}

func (h *SubtaskHandler) UpdateSubtask(w http.ResponseWriter, r *http.Request) error {
	var req dto.UpdateSubtaskRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		return err
	}
	card, apiErr := h.cardForEdit(r)
	if apiErr != nil {
		return apiErr
	}
	existing, apiErr := h.subtaskOnCard(r, card.cardID)
	if apiErr != nil {
		return apiErr
	}

	subtask, err := h.subtaskService.UpdateSubtask(r.Context(), existing.ID, req)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to update subtask", err)
	}

	h.broadcastSubtasks(r, card.boardID, card.cardID)
	httputil.RespondJSON(w, http.StatusOK, mapper.ToSubtaskResponse(subtask))
	return nil
}

func (h *SubtaskHandler) DeleteSubtask(w http.ResponseWriter, r *http.Request) error {
	card, apiErr := h.cardForEdit(r)
	if apiErr != nil {
		return apiErr
	}
	existing, apiErr := h.subtaskOnCard(r, card.cardID)
	if apiErr != nil {
		return apiErr
	}

	if err := h.subtaskService.DeleteSubtask(r.Context(), existing.ID); err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to delete subtask", err)
	}

	h.broadcastSubtasks(r, card.boardID, card.cardID)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (h *SubtaskHandler) GetSubtask(w http.ResponseWriter, r *http.Request) error {
	card, apiErr := h.cardAccess(r)
	if apiErr != nil {
		return apiErr
	}
	subtask, apiErr := h.subtaskOnCard(r, card.cardID)
	if apiErr != nil {
		return apiErr
	}

	httputil.RespondJSON(w, http.StatusOK, mapper.ToSubtaskResponse(subtask))
	return nil
}
