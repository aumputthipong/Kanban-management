package handler

import (
	"errors"
	"net/http"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/dto"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/middleware"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
)

type UserSettingsHandler struct {
	service service.UserSettingsServicer
}

func NewUserSettingsHandler(s service.UserSettingsServicer) *UserSettingsHandler {
	return &UserSettingsHandler{service: s}
}

// @Summary  My workspace settings
// @Tags     me
// @Produce  json
// @Security CookieAuth
// @Success  200 {object} dto.UserSettingsResponse
// @Failure  401 {object} httputil.ErrorResponse
// @Router   /api/me/settings [get]
func (h *UserSettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) error {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		return httputil.NewAPIError(http.StatusUnauthorized, "Unauthorized", nil)
	}
	s, err := h.service.Get(r.Context(), userID)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to load settings", err)
	}
	httputil.RespondJSON(w, http.StatusOK, toSettingsResponse(s))
	return nil
}

// @Summary  Update my workspace settings
// @Tags     me
// @Accept   json
// @Produce  json
// @Param    body body dto.UpdateUserSettingsRequest true "fields to change"
// @Security CookieAuth
// @Success  200 {object} dto.UserSettingsResponse
// @Failure  400 {object} httputil.ErrorResponse
// @Failure  401 {object} httputil.ErrorResponse
// @Router   /api/me/settings [patch]
func (h *UserSettingsHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) error {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		return httputil.NewAPIError(http.StatusUnauthorized, "Unauthorized", nil)
	}

	var req dto.UpdateUserSettingsRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid request body", err)
	}

	updated, err := h.service.Update(r.Context(), userID, service.UpdateUserSettingsParams{
		DefaultLanding: req.DefaultLanding,
		ShowAllCards:   req.ShowAllCards,
		Timezone:       req.Timezone,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidLanding) {
			return httputil.NewAPIError(http.StatusBadRequest, "default_landing must be today, my_work, or all_boards", err)
		}
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to update settings", err)
	}
	httputil.RespondJSON(w, http.StatusOK, toSettingsResponse(updated))
	return nil
}

func toSettingsResponse(s service.UserSettingsData) dto.UserSettingsResponse {
	return dto.UserSettingsResponse{
		DefaultLanding: s.DefaultLanding,
		ShowAllCards:   s.ShowAllCards,
		Timezone:       s.Timezone,
	}
}
