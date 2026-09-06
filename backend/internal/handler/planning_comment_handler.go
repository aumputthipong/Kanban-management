// Item comment thread handlers, split out so planning_handler.go stays focused on the
// session and item surface. Permissions follow the 404-not-403 anti-enumeration pattern
// (docs/adr/0004). Activity body previews are truncated to 80 chars.
package handler

import (
	"errors"
	"net/http"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/core"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/dto"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const commentBodyPreviewLen = 80

func commentToResponse(row db.ListPlanningItemCommentsRow) dto.PlanningCommentResponse {
	out := dto.PlanningCommentResponse{
		ID:         row.ID,
		ItemID:     row.ItemID,
		AuthorID:   row.AuthorID,
		AuthorName: row.AuthorName,
		CreatedAt:  row.CreatedAt.Format(timeFormat),
		UpdatedAt:  row.UpdatedAt.Format(timeFormat),
	}
	// Soft-deleted rows return deleted_at with a nil body so the UI can render a
	// placeholder. Returning the body would leak content after a delete.
	if row.DeletedAt != nil {
		ts := row.DeletedAt.Format(timeFormat)
		out.DeletedAt = &ts
	} else {
		body := row.Body
		out.Body = &body
	}
	return out
}

const timeFormat = "2006-01-02T15:04:05Z07:00"

func bodyPreview(s string) string {
	if len(s) <= commentBodyPreviewLen {
		return s
	}
	return s[:commentBodyPreviewLen] + "…"
}

// ListComments returns the full thread including soft-deleted rows.
func (h *PlanningHandler) ListComments(w http.ResponseWriter, r *http.Request) error {
	itemID := chi.URLParam(r, "itemID")
	if _, err := uuid.Parse(itemID); err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid item ID", err)
	}
	userID, apiErr := userIDFrom(r)
	if apiErr != nil {
		return apiErr
	}
	boardID, err := h.planning.GetItemBoardID(r.Context(), itemID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.NewAPIError(http.StatusNotFound, "Not found", nil)
		}
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to resolve board", err)
	}
	if _, apiErr := h.requireMembership(r, boardID, userID); apiErr != nil {
		return apiErr
	}
	rows, err := h.planning.ListItemComments(r.Context(), itemID)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to list comments", err)
	}
	out := make([]dto.PlanningCommentResponse, len(rows))
	for i, row := range rows {
		out[i] = commentToResponse(row)
	}
	httputil.RespondJSON(w, http.StatusOK, out)
	return nil
}

func (h *PlanningHandler) CreateComment(w http.ResponseWriter, r *http.Request) error {
	itemID := chi.URLParam(r, "itemID")
	if _, err := uuid.Parse(itemID); err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid item ID", err)
	}
	userID, apiErr := userIDFrom(r)
	if apiErr != nil {
		return apiErr
	}
	boardID, err := h.planning.GetItemBoardID(r.Context(), itemID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.NewAPIError(http.StatusNotFound, "Not found", nil)
		}
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to resolve board", err)
	}
	if _, apiErr := h.requireMembership(r, boardID, userID); apiErr != nil {
		return apiErr
	}
	var req dto.CreatePlanningCommentRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		return err
	}
	row, err := h.planning.CreateComment(r.Context(), itemID, userID, req.Body)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to create comment", err)
	}

	h.recordActivity(r, boardID, userID,
		service.EventPlanningCommentCreated, service.EntityPlanningComment,
		strPtr(row.ID),
		service.PlanningCommentCreatedPayload{ItemID: itemID, BodyPreview: bodyPreview(req.Body)},
	)

	// Hydrate author_name so the frontend can append the row without a refetch.
	body := row.Body
	resp := dto.PlanningCommentResponse{
		ID:         row.ID,
		ItemID:     row.ItemID,
		AuthorID:   row.AuthorID,
		AuthorName: "", // filled below if we can resolve
		Body:       &body,
		CreatedAt:  row.CreatedAt.Format(timeFormat),
		UpdatedAt:  row.UpdatedAt.Format(timeFormat),
	}
	if u, err := h.boards.GetAllUsers(r.Context()); err == nil {
		for _, m := range u {
			if m.ID == userID {
				resp.AuthorName = m.FullName
				break
			}
		}
	}
	httputil.RespondJSON(w, http.StatusCreated, resp)
	return nil
}

func (h *PlanningHandler) EditComment(w http.ResponseWriter, r *http.Request) error {
	commentID := chi.URLParam(r, "commentID")
	if _, err := uuid.Parse(commentID); err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid comment ID", err)
	}
	userID, apiErr := userIDFrom(r)
	if apiErr != nil {
		return apiErr
	}
	existing, err := h.planning.GetComment(r.Context(), commentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.NewAPIError(http.StatusNotFound, "Not found", nil)
		}
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to load comment", err)
	}
	boardID, err := h.planning.GetCommentBoardID(r.Context(), commentID)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to resolve board", err)
	}
	if _, apiErr := h.requireMembership(r, boardID, userID); apiErr != nil {
		return apiErr
	}
	// Edit-own only, and a 404 so probing other people's comment IDs reveals nothing.
	if existing.AuthorID != userID {
		return httputil.NewAPIError(http.StatusNotFound, "Not found", nil)
	}
	if existing.DeletedAt != nil {
		return httputil.NewAPIError(http.StatusConflict, "ความคิดเห็นถูกลบไปแล้ว", nil)
	}

	var req dto.UpdatePlanningCommentRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		return err
	}

	row, err := h.planning.EditComment(r.Context(), commentID, req.Body)
	if err != nil {
		if errors.Is(err, service.ErrPlanningCommentDeleted) {
			return httputil.NewAPIError(http.StatusConflict, "ความคิดเห็นถูกลบไปแล้ว", nil)
		}
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to edit comment", err)
	}

	h.recordActivity(r, boardID, userID,
		service.EventPlanningCommentEdited, service.EntityPlanningComment,
		strPtr(row.ID),
		service.PlanningCommentEditedPayload{ItemID: row.ItemID, BodyPreview: bodyPreview(req.Body)},
	)

	body := row.Body
	httputil.RespondJSON(w, http.StatusOK, dto.PlanningCommentResponse{
		ID:        row.ID,
		ItemID:    row.ItemID,
		AuthorID:  row.AuthorID,
		Body:      &body,
		CreatedAt: row.CreatedAt.Format(timeFormat),
		UpdatedAt: row.UpdatedAt.Format(timeFormat),
	})
	return nil
}

func (h *PlanningHandler) DeleteComment(w http.ResponseWriter, r *http.Request) error {
	commentID := chi.URLParam(r, "commentID")
	if _, err := uuid.Parse(commentID); err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid comment ID", err)
	}
	userID, apiErr := userIDFrom(r)
	if apiErr != nil {
		return apiErr
	}
	existing, err := h.planning.GetComment(r.Context(), commentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httputil.NewAPIError(http.StatusNotFound, "Not found", nil)
		}
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to load comment", err)
	}
	boardID, err := h.planning.GetCommentBoardID(r.Context(), commentID)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to resolve board", err)
	}
	role, apiErr := h.requireMembership(r, boardID, userID)
	if apiErr != nil {
		return apiErr
	}
	// Delete = own, or board owner/manager. Anything else is a 404 (see ADR 0004).
	isOwn := existing.AuthorID == userID
	canForceDelete := role == core.RoleOwner || role == core.RoleManager
	if !isOwn && !canForceDelete {
		return httputil.NewAPIError(http.StatusNotFound, "Not found", nil)
	}
	if existing.DeletedAt != nil {
		// Idempotent — a re-delete is a no-op rather than an error.
		w.WriteHeader(http.StatusNoContent)
		return nil
	}

	if err := h.planning.DeleteComment(r.Context(), commentID); err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to delete comment", err)
	}

	h.recordActivity(r, boardID, userID,
		service.EventPlanningCommentDeleted, service.EntityPlanningComment,
		strPtr(commentID),
		service.PlanningCommentDeletedPayload{ItemID: existing.ItemID},
	)

	w.WriteHeader(http.StatusNoContent)
	return nil
}
