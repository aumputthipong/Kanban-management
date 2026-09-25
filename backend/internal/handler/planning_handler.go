package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/core"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/dto"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/middleware"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PlanningHandler struct {
	planning service.PlanningServicer
	boards   service.BoardServicer
	activity service.ActivityRecorder
}

func NewPlanningHandler(p service.PlanningServicer, b service.BoardServicer, a service.ActivityRecorder) *PlanningHandler {
	return &PlanningHandler{planning: p, boards: b, activity: a}
}

// Fire-and-forget: the REST planning path has no broadcast that needs the row.
func (h *PlanningHandler) recordActivity(
	r *http.Request,
	boardID, actorID, eventType, entityType string,
	entityID *string,
	payload any,
) {
	_ = r // async write uses a background ctx, not the request ctx
	if h.activity == nil {
		return
	}
	if _, err := uuid.Parse(actorID); err != nil {
		return
	}
	h.activity.RecordAsync(service.RecordParams{
		BoardID:    boardID,
		ActorID:    actorID,
		EventType:  eventType,
		EntityType: entityType,
		EntityID:   entityID,
		Payload:    payload,
	})
}

func strPtr(s string) *string { return &s }

// Re-checks membership when the URL only carries a session/item id (404, not 403).
func (h *PlanningHandler) requireMembership(r *http.Request, boardID, userID string) (core.BoardRole, *httputil.APIError) {
	role, err := h.boards.GetBoardMemberRole(r.Context(), boardID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", httputil.NewAPIError(http.StatusNotFound, "Not found", nil)
		}
		return "", httputil.NewAPIError(http.StatusInternalServerError, "Failed to check board access", err)
	}
	return core.BoardRole(role), nil
}

func userIDFrom(r *http.Request) (string, *httputil.APIError) {
	id, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || id == "" {
		return "", httputil.NewAPIError(http.StatusUnauthorized, "Unauthorized", nil)
	}
	return id, nil
}

func ptrTimeToString(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

func sessionRowToSummary(r db.ListPlanningSessionsByBoardRow) dto.PlanningSessionSummary {
	return dto.PlanningSessionSummary{
		ID:            r.ID,
		BoardID:       r.BoardID,
		Title:         r.Title,
		Label:         r.Label,
		MeetingAt:     ptrTimeToString(r.MeetingAt),
		CreatedAt:     r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     r.UpdatedAt.Format(time.RFC3339),
		ReqCount:      r.ReqCount,
		DecCount:      r.DecCount,
		QCount:        r.QCount,
		PromotedCount: r.PromotedCount,
		DroppedCount:  r.DroppedCount,
	}
}

func itemToResponse(it db.PlanningItem) dto.PlanningItemResponse {
	return dto.PlanningItemResponse{
		ID:                 it.ID,
		SessionID:          it.SessionID,
		Type:               it.Type,
		Title:              it.Title,
		Description:        it.Description,
		Status:             it.Status,
		PromotedToCardID:   it.PromotedToCardID,
		Position:           it.Position,
		CreatedAt:          it.CreatedAt.Format(time.RFC3339),
		AcceptanceCriteria: it.AcceptanceCriteria,
		ImplementationNote: it.ImplementationNote,
	}
}

// Sessions

func (h *PlanningHandler) ListSessions(w http.ResponseWriter, r *http.Request) error {
	boardID, err := httputil.GetUUIDParam(r, "boardID")
	if err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid board ID", err)
	}
	rows, err := h.planning.ListSessionsByBoard(r.Context(), boardID)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to list sessions", err)
	}
	out := make([]dto.PlanningSessionSummary, len(rows))
	for i, row := range rows {
		out[i] = sessionRowToSummary(row)
	}
	httputil.RespondJSON(w, http.StatusOK, out)
	return nil
}

func (h *PlanningHandler) CreateSession(w http.ResponseWriter, r *http.Request) error {
	boardID, err := httputil.GetUUIDParam(r, "boardID")
	if err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid board ID", err)
	}
	userID, apiErr := userIDFrom(r)
	if apiErr != nil {
		return apiErr
	}
	var req dto.CreatePlanningSessionRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		return err
	}
	sess, err := h.planning.CreateSession(r.Context(), boardID, req.Title, req.Label, req.MeetingAt, userID)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to create session", err)
	}
	h.recordActivity(r, boardID, userID,
		service.EventPlanningSessionCreated, service.EntityPlanningSession,
		strPtr(sess.ID),
		service.PlanningSessionCreatedPayload{Title: sess.Title},
	)
	httputil.RespondJSON(w, http.StatusCreated, dto.PlanningSessionSummary{
		ID:        sess.ID,
		BoardID:   sess.BoardID,
		Title:     sess.Title,
		Label:     sess.Label,
		MeetingAt: ptrTimeToString(sess.MeetingAt),
		CreatedAt: sess.CreatedAt.Format(time.RFC3339),
		UpdatedAt: sess.UpdatedAt.Format(time.RFC3339),
	})
	return nil
}

func (h *PlanningHandler) GetSession(w http.ResponseWriter, r *http.Request) error {
	sessionID := chi.URLParam(r, "sessionID")
	if _, err := uuid.Parse(sessionID); err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid session ID", err)
	}
	userID, apiErr := userIDFrom(r)
	if apiErr != nil {
		return apiErr
	}
	sess, err := h.planning.GetSession(r.Context(), sessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.NewAPIError(http.StatusNotFound, "Not found", nil)
		}
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to load session", err)
	}
	if _, apiErr := h.requireMembership(r, sess.BoardID, userID); apiErr != nil {
		return apiErr
	}
	items, err := h.planning.ListItems(r.Context(), sessionID)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to list items", err)
	}
	itemDTOs := make([]dto.PlanningItemResponse, len(items))
	for i, it := range items {
		itemDTOs[i] = itemToResponse(it)
	}
	httputil.RespondJSON(w, http.StatusOK, dto.PlanningSessionDetail{
		ID:        sess.ID,
		BoardID:   sess.BoardID,
		Title:     sess.Title,
		Label:     sess.Label,
		MeetingAt: ptrTimeToString(sess.MeetingAt),
		CreatedAt: sess.CreatedAt.Format(time.RFC3339),
		UpdatedAt: sess.UpdatedAt.Format(time.RFC3339),
		Items:     itemDTOs,
	})
	return nil
}

func (h *PlanningHandler) UpdateSession(w http.ResponseWriter, r *http.Request) error {
	sessionID := chi.URLParam(r, "sessionID")
	if _, err := uuid.Parse(sessionID); err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid session ID", err)
	}
	userID, apiErr := userIDFrom(r)
	if apiErr != nil {
		return apiErr
	}
	boardID, err := h.planning.GetSessionBoardID(r.Context(), sessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.NewAPIError(http.StatusNotFound, "Not found", nil)
		}
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to resolve board", err)
	}
	if _, apiErr := h.requireMembership(r, boardID, userID); apiErr != nil {
		return apiErr
	}
	var req dto.UpdatePlanningSessionRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		return err
	}
	// Defence in depth — `min=1` already rejects "".
	if req.Title != nil && *req.Title == "" {
		return httputil.NewAPIError(http.StatusBadRequest, "title cannot be empty", nil)
	}
	sess, err := h.planning.UpdateSession(r.Context(), sessionID, req.Title, req.Label, req.MeetingAt)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to update session", err)
	}
	fields := make([]string, 0, 3)
	if req.Title != nil {
		fields = append(fields, "title")
	}
	if req.Label != nil {
		fields = append(fields, "label")
	}
	if req.MeetingAt != nil {
		fields = append(fields, "meeting_at")
	}
	h.recordActivity(r, boardID, userID,
		service.EventPlanningSessionUpdated, service.EntityPlanningSession,
		strPtr(sess.ID),
		service.PlanningSessionUpdatedPayload{Title: sess.Title, Fields: fields},
	)
	httputil.RespondJSON(w, http.StatusOK, dto.PlanningSessionSummary{
		ID:        sess.ID,
		BoardID:   sess.BoardID,
		Title:     sess.Title,
		Label:     sess.Label,
		MeetingAt: ptrTimeToString(sess.MeetingAt),
		CreatedAt: sess.CreatedAt.Format(time.RFC3339),
		UpdatedAt: sess.UpdatedAt.Format(time.RFC3339),
	})
	return nil
}

func (h *PlanningHandler) DeleteSession(w http.ResponseWriter, r *http.Request) error {
	sessionID := chi.URLParam(r, "sessionID")
	if _, err := uuid.Parse(sessionID); err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid session ID", err)
	}
	userID, apiErr := userIDFrom(r)
	if apiErr != nil {
		return apiErr
	}
	// Read first so the activity row can carry the title.
	sess, err := h.planning.GetSession(r.Context(), sessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.NewAPIError(http.StatusNotFound, "Not found", nil)
		}
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to load session", err)
	}
	if _, apiErr := h.requireMembership(r, sess.BoardID, userID); apiErr != nil {
		return apiErr
	}
	if err := h.planning.DeleteSession(r.Context(), sessionID); err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to delete session", err)
	}
	h.recordActivity(r, sess.BoardID, userID,
		service.EventPlanningSessionDeleted, service.EntityPlanningSession,
		strPtr(sessionID),
		service.PlanningSessionDeletedPayload{Title: sess.Title},
	)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// Items

func (h *PlanningHandler) CreateItem(w http.ResponseWriter, r *http.Request) error {
	sessionID := chi.URLParam(r, "sessionID")
	if _, err := uuid.Parse(sessionID); err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid session ID", err)
	}
	userID, apiErr := userIDFrom(r)
	if apiErr != nil {
		return apiErr
	}
	boardID, err := h.planning.GetSessionBoardID(r.Context(), sessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.NewAPIError(http.StatusNotFound, "Not found", nil)
		}
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to resolve board", err)
	}
	if _, apiErr := h.requireMembership(r, boardID, userID); apiErr != nil {
		return apiErr
	}
	var req dto.CreatePlanningItemRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		return err
	}
	item, err := h.planning.CreateItem(r.Context(), sessionID, req.Type, req.Title, req.Description)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to create item", err)
	}
	h.recordActivity(r, boardID, userID,
		service.EventPlanningItemCreated, service.EntityPlanningItem,
		strPtr(item.ID),
		service.PlanningItemCreatedPayload{Type: item.Type, Title: item.Title},
	)
	httputil.RespondJSON(w, http.StatusCreated, itemToResponse(item))
	return nil
}

func (h *PlanningHandler) UpdateItem(w http.ResponseWriter, r *http.Request) error {
	itemID := chi.URLParam(r, "itemID")
	if _, err := uuid.Parse(itemID); err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid item ID", err)
	}
	userID, apiErr := userIDFrom(r)
	if apiErr != nil {
		return apiErr
	}
	// Also captures the pre-update type (previous_type, promoted freeze).
	current, err := h.planning.GetItem(r.Context(), itemID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.NewAPIError(http.StatusNotFound, "Not found", nil)
		}
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to load item", err)
	}
	boardID, err := h.planning.GetSessionBoardID(r.Context(), current.SessionID)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to resolve board", err)
	}
	if _, apiErr := h.requireMembership(r, boardID, userID); apiErr != nil {
		return apiErr
	}
	var req dto.UpdatePlanningItemRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		return err
	}
	if req.Title != nil && *req.Title == "" {
		return httputil.NewAPIError(http.StatusBadRequest, "title cannot be empty", nil)
	}
	// Promoted items can't be retyped — the card already carries the original meaning.
	if req.Type != nil && *req.Type != current.Type && current.Status == "promoted" {
		return httputil.NewAPIError(http.StatusBadRequest, "ส่งเข้า Board แล้ว เปลี่ยนประเภทไม่ได้", nil)
	}
	item, err := h.planning.UpdateItem(r.Context(), itemID, req.Type, req.Title, req.Description, req.Status, req.Position, req.AcceptanceCriteria, req.ImplementationNote)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to update item", err)
	}
	fields := make([]string, 0, 7)
	if req.Type != nil {
		fields = append(fields, "type")
	}
	if req.Title != nil {
		fields = append(fields, "title")
	}
	if req.Description != nil {
		fields = append(fields, "description")
	}
	if req.Status != nil {
		fields = append(fields, "status")
	}
	if req.Position != nil {
		fields = append(fields, "position")
	}
	if req.AcceptanceCriteria != nil {
		fields = append(fields, "acceptance_criteria")
	}
	if req.ImplementationNote != nil {
		fields = append(fields, "implementation_note")
	}
	payload := service.PlanningItemUpdatedPayload{
		Type:   item.Type,
		Title:  item.Title,
		Fields: fields,
	}
	if req.Type != nil && *req.Type != current.Type {
		payload.PreviousType = current.Type
	}
	h.recordActivity(r, boardID, userID,
		service.EventPlanningItemUpdated, service.EntityPlanningItem,
		strPtr(item.ID),
		payload,
	)
	httputil.RespondJSON(w, http.StatusOK, itemToResponse(item))
	return nil
}

func (h *PlanningHandler) DeleteItem(w http.ResponseWriter, r *http.Request) error {
	itemID := chi.URLParam(r, "itemID")
	if _, err := uuid.Parse(itemID); err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid item ID", err)
	}
	userID, apiErr := userIDFrom(r)
	if apiErr != nil {
		return apiErr
	}
	// Read first so the activity row can carry type + title.
	item, err := h.planning.GetItem(r.Context(), itemID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.NewAPIError(http.StatusNotFound, "Not found", nil)
		}
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to load item", err)
	}
	boardID, err := h.planning.GetItemBoardID(r.Context(), itemID)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to resolve board", err)
	}
	if _, apiErr := h.requireMembership(r, boardID, userID); apiErr != nil {
		return apiErr
	}
	if err := h.planning.DeleteItem(r.Context(), itemID); err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to delete item", err)
	}
	h.recordActivity(r, boardID, userID,
		service.EventPlanningItemDeleted, service.EntityPlanningItem,
		strPtr(itemID),
		service.PlanningItemDeletedPayload{Type: item.Type, Title: item.Title},
	)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (h *PlanningHandler) PromoteItem(w http.ResponseWriter, r *http.Request) error {
	itemID := chi.URLParam(r, "itemID")
	if _, err := uuid.Parse(itemID); err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid item ID", err)
	}
	userID, apiErr := userIDFrom(r)
	if apiErr != nil {
		return apiErr
	}
	before, err := h.planning.GetItem(r.Context(), itemID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.NewAPIError(http.StatusNotFound, "Not found", nil)
		}
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to load item", err)
	}
	boardID, err := h.planning.GetSessionBoardID(r.Context(), before.SessionID)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to resolve board", err)
	}
	if _, apiErr := h.requireMembership(r, boardID, userID); apiErr != nil {
		return apiErr
	}
	item, card, err := h.planning.PromoteItem(r.Context(), itemID, userID)
	if err != nil {
		if errors.Is(err, service.ErrPlanningItemAlreadyPromoted) {
			return httputil.NewAPIError(http.StatusConflict, "Item already promoted", err)
		}
		if errors.Is(err, service.ErrPlanningItemDropped) {
			return httputil.NewAPIError(http.StatusUnprocessableEntity, "Cannot promote a dropped item — un-drop it first", err)
		}
		if errors.Is(err, service.ErrPlanningNoTodoColumn) {
			return httputil.NewAPIError(http.StatusUnprocessableEntity, "Board has no TODO column — add one before promoting", err)
		}
		if errors.Is(err, service.ErrPlanningNotFound) {
			return httputil.NewAPIError(http.StatusNotFound, "Not found", err)
		}
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to promote item", err)
	}
	// No card.created here — it would double-count on the feed.
	h.recordActivity(r, boardID, userID,
		service.EventPlanningItemPromoted, service.EntityPlanningItem,
		strPtr(item.ID),
		service.PlanningItemPromotedPayload{
			Type:     item.Type,
			Title:    item.Title,
			ToCardID: card.ID,
		},
	)
	httputil.RespondJSON(w, http.StatusOK, map[string]any{
		"item":    itemToResponse(item),
		"card_id": card.ID,
	})
	return nil
}

// Card source

// null (not 404) when never promoted. Membership is re-checked — /api/cards has no gate.
func (h *PlanningHandler) GetCardSource(w http.ResponseWriter, r *http.Request) error {
	cardID := chi.URLParam(r, "cardID")
	if _, err := uuid.Parse(cardID); err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid card ID", err)
	}
	userID, apiErr := userIDFrom(r)
	if apiErr != nil {
		return apiErr
	}
	boardID, err := h.boards.GetBoardIDByCard(r.Context(), cardID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.NewAPIError(http.StatusNotFound, "Not found", nil)
		}
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to resolve board", err)
	}
	if _, apiErr := h.requireMembership(r, boardID, userID); apiErr != nil {
		return apiErr
	}

	source, err := h.planning.GetCardSource(r.Context(), cardID, 3)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to fetch card source", err)
	}
	if source == nil {
		// Explicit JSON null: RespondJSON writes an empty body for nil.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("null"))
		return nil
	}

	resp := dto.CardSourceResponse{
		Session: dto.CardSourceSession{
			ID:        source.SessionID,
			Title:     source.SessionTitle,
			Label:     source.SessionLabel,
			MeetingAt: ptrTimeToString(source.SessionMeetingAt),
		},
		Item: dto.CardSourceItem{
			ID:     source.ItemID,
			Type:   source.ItemType,
			Title:  source.ItemTitle,
			Status: source.ItemStatus,
		},
		PendingQuestions: make([]dto.CardSourceQuestion, len(source.PendingQuestions)),
	}
	for i, q := range source.PendingQuestions {
		resp.PendingQuestions[i] = dto.CardSourceQuestion{ID: q.ID, Title: q.Title}
	}
	httputil.RespondJSON(w, http.StatusOK, resp)
	return nil
}
