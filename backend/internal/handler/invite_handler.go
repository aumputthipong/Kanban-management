package handler

import (
	"errors"
	"net/http"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/dto"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/middleware"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type InviteHandler struct {
	invites service.InviteServicer
}

func NewInviteHandler(invites service.InviteServicer) *InviteHandler {
	return &InviteHandler{invites: invites}
}

// CreateInvite (re)generates the board's shareable invite link. Manager+ only
// (gated by the route).
func (h *InviteHandler) CreateInvite(w http.ResponseWriter, r *http.Request) error {
	boardID, err := httputil.GetUUIDParam(r, "boardID")
	if err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid board ID", err)
	}
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		return httputil.NewAPIError(http.StatusUnauthorized, "Unauthorized", nil)
	}
	link, err := h.invites.CreateInvite(r.Context(), boardID, userID)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to create invite link", err)
	}
	httputil.RespondJSON(w, http.StatusCreated, dto.InviteLinkResponse{
		Token:     link.Token,
		ExpiresAt: link.ExpiresAt,
	})
	return nil
}

// GetActiveInvite returns the board's current link, or 204 when there isn't one.
func (h *InviteHandler) GetActiveInvite(w http.ResponseWriter, r *http.Request) error {
	boardID, err := httputil.GetUUIDParam(r, "boardID")
	if err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid board ID", err)
	}
	link, ok, err := h.invites.GetActiveInvite(r.Context(), boardID)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to load invite link", err)
	}
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
	httputil.RespondJSON(w, http.StatusOK, dto.InviteLinkResponse{
		Token:     link.Token,
		ExpiresAt: link.ExpiresAt,
	})
	return nil
}

// RevokeInvite turns off the board's active link. Manager+ only.
func (h *InviteHandler) RevokeInvite(w http.ResponseWriter, r *http.Request) error {
	boardID, err := httputil.GetUUIDParam(r, "boardID")
	if err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid board ID", err)
	}
	if err := h.invites.RevokeInvites(r.Context(), boardID); err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to revoke invite link", err)
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// AcceptInvite joins the authenticated caller to the board the token points at.
// Deliberately NOT board-gated — the caller isn't a member yet; auth plus a
// valid, unexpired, unrevoked token is the gate.
func (h *InviteHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) error {
	token := chi.URLParam(r, "token")
	if token == "" {
		return httputil.NewAPIError(http.StatusBadRequest, "Missing invite token", nil)
	}
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		return httputil.NewAPIError(http.StatusUnauthorized, "Unauthorized", nil)
	}
	boardID, err := h.invites.AcceptInvite(r.Context(), token, userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInviteInvalid):
			return httputil.NewAPIError(http.StatusNotFound, "ลิงก์เชิญไม่ถูกต้องหรือถูกปิดแล้ว", err)
		case errors.Is(err, service.ErrInviteExpired):
			return httputil.NewAPIError(http.StatusGone, "ลิงก์เชิญหมดอายุแล้ว", err)
		default:
			return httputil.NewAPIError(http.StatusInternalServerError, "Failed to accept invite", err)
		}
	}
	httputil.RespondJSON(w, http.StatusOK, dto.AcceptInviteResponse{BoardID: boardID})
	return nil
}
