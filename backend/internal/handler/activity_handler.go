package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/dto"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
)

type ActivityHandler struct {
	activityService service.ActivityLister
}

func NewActivityHandler(activityService service.ActivityLister) *ActivityHandler {
	return &ActivityHandler{activityService: activityService}
}

func (h *ActivityHandler) ListByBoard(w http.ResponseWriter, r *http.Request) error {
	boardID, err := httputil.GetUUIDParam(r, "boardID")
	if err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid board ID", err)
	}
	q := r.URL.Query()

	var before *time.Time
	if s := q.Get("before"); s != "" {
		t, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			return httputil.NewAPIError(http.StatusBadRequest, "Invalid 'before' timestamp", err)
		}
		before = &t
	}

	limit := int32(30)
	if s := q.Get("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n <= 0 {
			return httputil.NewAPIError(http.StatusBadRequest, "Invalid 'limit'", err)
		}
		limit = int32(n)
	}

	items, err := h.activityService.List(r.Context(), boardID, before, limit)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to fetch activities", err)
	}

	out := make([]dto.ActivityResponse, len(items))
	for i, it := range items {
		var payload json.RawMessage
		if len(it.Payload) > 0 {
			payload = json.RawMessage(it.Payload)
		} else {
			payload = json.RawMessage("{}")
		}
		out[i] = dto.ActivityResponse{
			ID:         it.ID,
			BoardID:    it.BoardID,
			ActorID:    it.ActorID,
			ActorName:  it.ActorName,
			EventType:  it.EventType,
			EntityType: it.EntityType,
			EntityID:   it.EntityID,
			Payload:    payload,
			CreatedAt:  it.CreatedAt.UTC().Format(time.RFC3339Nano),
		}
	}
	httputil.RespondJSON(w, http.StatusOK, out)
	return nil
}
