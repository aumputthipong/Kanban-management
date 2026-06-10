package handler

import (
	"net/http"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/dto"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/middleware"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/google/uuid"
)

// GetMyTasks returns the caller's cross-board work inbox: cards assigned to
// them plus (optionally) unassigned cards on boards they're a member of.
// Cards are pre-grouped by due-date bucket and counts cover the unfiltered
// inbox so the UI can render filter chips with totals.
//
// Query params:
//   filter            — all (default) | overdue | today | this_week | no_date
//   include_unassigned — bool. Solo users without an assignee toggle this on.
//
// @Summary  My work (cross-board)
// @Tags     my-tasks
// @Produce  json
// @Param    filter             query string false "all|overdue|today|this_week|no_date"
// @Param    include_unassigned query bool   false "include unassigned cards on my boards"
// @Security CookieAuth
// @Success  200 {object} dto.MyWorkResponse
// @Failure  401 {object} httputil.ErrorResponse
// @Router   /api/my-tasks [get]
func (h *BoardHandler) GetMyTasks(w http.ResponseWriter, r *http.Request) error {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		return httputil.NewAPIError(http.StatusUnauthorized, "Unauthorized", nil)
	}

	filter := service.MyWorkFilter(r.URL.Query().Get("filter"))

	// Settings is the source of truth for include_unassigned + timezone.
	// Query params for these are intentionally ignored (S.2 decision).
	var include bool
	var tz string
	if h.settingsService != nil {
		settings, err := h.settingsService.Get(r.Context(), userID)
		if err != nil {
			return httputil.NewAPIError(http.StatusInternalServerError, "Failed to load settings", err)
		}
		include = settings.ShowAllCards
		tz = settings.Timezone
	}

	result, err := h.boardService.GetMyWork(r.Context(), service.MyWorkOptions{
		UserID:            userID,
		IncludeUnassigned: include,
		Filter:            filter,
		Today:             service.MyWorkToday(time.Now(), tz),
	})
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to load my tasks", err)
	}

	cards := make([]dto.MyTaskResponse, 0, len(result.Cards))
	for _, t := range result.Cards {
		var due *string
		if t.DueDate != nil {
			s := t.DueDate.Format("2006-01-02")
			due = &s
		}
		cards = append(cards, dto.MyTaskResponse{
			ID:                t.ID,
			Title:             t.Title,
			BoardID:           t.BoardID,
			BoardName:         t.BoardName,
			ColumnName:        t.ColumnName,
			Priority:          t.Priority,
			DueDate:           due,
			EstimatedHours:    t.EstimatedHours,
			Status:            t.Status,
			Group:             t.Group,
			TotalSubtasks:     t.TotalSubtasks,
			CompletedSubtasks: t.CompletedSubtasks,
		})
	}

	httputil.RespondJSON(w, http.StatusOK, dto.MyWorkResponse{
		Cards: cards,
		Counts: dto.MyWorkCounts{
			Overdue:  result.Counts.Overdue,
			Today:    result.Counts.Today,
			ThisWeek: result.Counts.ThisWeek,
			Later:    result.Counts.Later,
			NoDate:   result.Counts.NoDate,
			Total:    result.Counts.Total,
		},
	})
	return nil
}

// CompleteMyTask marks an assigned card as done. Caller must be the assignee.
//
// @Summary  Complete one of my tasks
// @Tags     my-tasks
// @Security CookieAuth
// @Param    cardID path string true "Card UUID"
// @Success  204
// @Failure  400    {object} httputil.ErrorResponse
// @Failure  404    {object} httputil.ErrorResponse "not the assignee or card missing"
// @Router   /api/my-tasks/{cardID}/complete [post]
func (h *BoardHandler) CompleteMyTask(w http.ResponseWriter, r *http.Request) error {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		return httputil.NewAPIError(http.StatusUnauthorized, "Unauthorized", nil)
	}

	cardID, err := httputil.GetUUIDParam(r, "cardID")
	if err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid card ID", err)
	}
	if _, err := uuid.Parse(cardID); err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid card ID", err)
	}

	result, err := h.boardService.CompleteMyTask(r.Context(), cardID, userID)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to complete task", err)
	}
	if !result.OK {
		// Not the assignee, or card doesn't exist.
		return httputil.NewAPIError(http.StatusNotFound, "Task not found", nil)
	}

	// Best-effort audit row so My Work completions show up on the board's
	// activity feed alongside drag-to-done. Audit failures don't roll back
	// the mutation — RecordAsync handles its own logging.
	if h.activity != nil {
		h.activity.RecordAsync(service.RecordParams{
			BoardID:    result.BoardID,
			ActorID:    userID,
			EventType:  service.EventCardDoneToggled,
			EntityType: service.EntityCard,
			EntityID:   &cardID,
			Payload: map[string]any{
				"title":   result.CardTitle,
				"is_done": true,
				"via":     "my_work",
			},
		})
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}
