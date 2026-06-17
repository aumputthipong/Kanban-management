package handler

import (
	"errors"
	"net/http"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/dto"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TagHandler struct {
	tagService *service.TagService
}

func NewTagHandler(tagService *service.TagService) *TagHandler {
	return &TagHandler{tagService: tagService}
}

func (h *TagHandler) GetBoardTags(w http.ResponseWriter, r *http.Request) error {
	boardID, err := httputil.GetUUIDParam(r, "boardID")
	if err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid board ID", err)
	}
	tags, err := h.tagService.GetTagsByBoard(r.Context(), boardID)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to fetch tags", err)
	}
	result := make([]dto.TagResponse, len(tags))
	for i, t := range tags {
		result[i] = dto.TagResponse{ID: t.ID, BoardID: t.BoardID, Name: t.Name, Color: t.Color}
	}
	httputil.RespondJSON(w, http.StatusOK, result)
	return nil
}

func (h *TagHandler) CreateBoardTag(w http.ResponseWriter, r *http.Request) error {
	boardID, err := httputil.GetUUIDParam(r, "boardID")
	if err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid board ID", err)
	}
	var req dto.CreateTagRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		return err
	}
	tag, err := h.tagService.CreateTag(r.Context(), boardID, req.Name, req.Color)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTagNameEmpty), errors.Is(err, service.ErrTagNameTooLong):
			// Sentinel messages are safe to surface verbatim.
			return httputil.NewAPIError(http.StatusUnprocessableEntity, err.Error(), err)
		default:
			// Don't leak raw DB errors (constraint names, etc.) to the client.
			return httputil.NewAPIError(http.StatusInternalServerError, "Failed to create tag", err)
		}
	}
	httputil.RespondJSON(w, http.StatusCreated, dto.TagResponse{
		ID: tag.ID, BoardID: tag.BoardID, Name: tag.Name, Color: tag.Color,
	})
	return nil
}

func (h *TagHandler) DeleteBoardTag(w http.ResponseWriter, r *http.Request) error {
	boardID, err := httputil.GetUUIDParam(r, "boardID")
	if err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid board ID", err)
	}
	tagID := chi.URLParam(r, "tagID")
	if _, err := uuid.Parse(tagID); err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid tag ID", err)
	}
	if err := h.tagService.DeleteTag(r.Context(), boardID, tagID); err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to delete tag", err)
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
