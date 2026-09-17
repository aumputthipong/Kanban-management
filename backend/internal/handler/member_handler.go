package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/core"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/dto"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/middleware"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *BoardHandler) GetBoardMembers(w http.ResponseWriter, r *http.Request) error {
	boardID, err := httputil.GetUUIDParam(r, "boardID")
	if err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid board ID", err)
	}

	members, err := h.boardService.GetBoardMembers(r.Context(), boardID)
	if err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to fetch members", err)
	}

	result := toMemberResponses(members)

	// Board membership changes infrequently; short private cache cuts the
	// refetch storm when a user opens multiple board tabs.
	w.Header().Set("Cache-Control", "private, max-age=30")
	httputil.RespondJSON(w, http.StatusOK, result)
	return nil
}

func toMemberResponses(members []db.GetBoardMembersRow) []dto.BoardMemberResponse {
	result := make([]dto.BoardMemberResponse, 0, len(members))
	for _, m := range members {
		result = append(result, dto.BoardMemberResponse{
			ID:       m.ID,
			Role:     m.Role,
			UserID:   m.UserID,
			Email:    m.Email,
			FullName: m.FullName,
		})
	}
	return result
}

// broadcastMembers sends the board's full member list after a membership change. Open
// boards use it for the assignee picker, the member filter and client-side edit rights.
// It returns the list it sent (nil on a failed read) so callers can name the member.
func broadcastMembers(ctx context.Context, svc service.BoardServicer, b Broadcaster, boardID string) []db.GetBoardMembersRow {
	if b == nil {
		return nil
	}
	members, err := svc.GetBoardMembers(ctx, boardID)
	if err != nil {
		slog.Error("load members for broadcast failed", "board_id", boardID, "err", err)
		return nil
	}
	emitTo(b, boardID, core.WSBoardMembersUpdated, map[string]any{"members": toMemberResponses(members)})
	return members
}

// memberSnapshot reads one member before a change that removes or alters them. A failed
// read only costs the feed a name, so it is logged and a bare row returned.
func memberSnapshot(ctx context.Context, svc service.BoardServicer, rec service.ActivityRecorder, boardID, userID string) db.GetBoardMembersRow {
	if rec == nil {
		return db.GetBoardMembersRow{UserID: userID}
	}
	members, err := svc.GetBoardMembers(ctx, boardID)
	if err != nil {
		slog.Error("load member for activity failed", "board_id", boardID, "err", err)
		return db.GetBoardMembersRow{UserID: userID}
	}
	for _, m := range members {
		if m.UserID == userID {
			return m
		}
	}
	return db.GetBoardMembersRow{UserID: userID}
}

func memberActivity(boardID, actorID, event string, payload service.MemberChangedPayload) service.RecordParams {
	return service.RecordParams{
		BoardID: boardID, ActorID: actorID, EventType: event,
		EntityType: service.EntityMember, EntityID: &payload.UserID, Payload: payload,
	}
}

func (h *BoardHandler) AddBoardMember(w http.ResponseWriter, r *http.Request) error {
	boardID, err := httputil.GetUUIDParam(r, "boardID")
	if err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid board ID", err)
	}

	var req dto.AddMemberRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		return err
	}

	if err := h.boardService.AddBoardMemberByEmail(r.Context(), boardID, req.Email, req.Role); err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			return httputil.NewAPIError(http.StatusNotFound, "ไม่พบผู้ใช้ที่มีอีเมลนี้", err)
		case errors.Is(err, service.ErrAlreadyMember):
			return httputil.NewAPIError(http.StatusConflict, "ผู้ใช้นี้เป็นสมาชิกบอร์ดอยู่แล้ว", err)
		default:
			return httputil.NewAPIError(http.StatusInternalServerError, "Failed to add member", err)
		}
	}

	members := broadcastMembers(r.Context(), h.boardService, h.broadcaster, boardID)
	for _, m := range members {
		if strings.EqualFold(m.Email, req.Email) {
			actorID, _ := r.Context().Value(middleware.UserIDKey).(string)
			recordActivity(r.Context(), h.activity, h.broadcaster, memberActivity(boardID, actorID, service.EventMemberAdded,
				service.MemberChangedPayload{UserID: m.UserID, Name: m.FullName, Role: m.Role}))
			break
		}
	}
	w.WriteHeader(http.StatusCreated)
	return nil
}

func (h *BoardHandler) RemoveBoardMember(w http.ResponseWriter, r *http.Request) error {
	boardID, err := httputil.GetUUIDParam(r, "boardID")
	if err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid board ID", err)
	}

	userIDStr := chi.URLParam(r, "userID")
	if _, err := uuid.Parse(userIDStr); err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid user ID", err)
	}

	removed := memberSnapshot(r.Context(), h.boardService, h.activity, boardID, userIDStr)
	if err := h.boardService.RemoveBoardMember(r.Context(), boardID, userIDStr); err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to remove member", err)
	}
	// Broadcast first: the removed user's client needs the new list to leave the board.
	broadcastMembers(r.Context(), h.boardService, h.broadcaster, boardID)
	evictFromBoard(h.broadcaster, boardID, userIDStr)
	actorID, _ := r.Context().Value(middleware.UserIDKey).(string)
	recordActivity(r.Context(), h.activity, h.broadcaster, memberActivity(boardID, actorID, service.EventMemberRemoved,
		service.MemberChangedPayload{UserID: userIDStr, Name: removed.FullName, Role: removed.Role}))

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (h *BoardHandler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) error {
	boardID, err := httputil.GetUUIDParam(r, "boardID")
	if err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid board ID", err)
	}

	userIDStr := chi.URLParam(r, "userID")
	if _, err := uuid.Parse(userIDStr); err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid user ID", err)
	}

	var req dto.UpdateMemberRoleRequest
	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		return err
	}
	if core.BoardRole(req.Role) == core.RoleOwner {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid role — cannot change to owner", nil)
	}

	before := memberSnapshot(r.Context(), h.boardService, h.activity, boardID, userIDStr)
	if err := h.boardService.UpdateMemberRole(r.Context(), boardID, userIDStr, req.Role); err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to update role", err)
	}
	broadcastMembers(r.Context(), h.boardService, h.broadcaster, boardID)
	if before.Role != req.Role {
		actorID, _ := r.Context().Value(middleware.UserIDKey).(string)
		recordActivity(r.Context(), h.activity, h.broadcaster, memberActivity(boardID, actorID, service.EventMemberRole,
			service.MemberChangedPayload{UserID: userIDStr, Name: before.FullName, Role: req.Role, PreviousRole: before.Role}))
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

// LeaveBoard removes the current user from the board. An owner must transfer
// ownership before they can leave.
func (h *BoardHandler) LeaveBoard(w http.ResponseWriter, r *http.Request) error {
	boardID, err := httputil.GetUUIDParam(r, "boardID")
	if err != nil {
		return httputil.NewAPIError(http.StatusBadRequest, "Invalid board ID", err)
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		return httputil.NewAPIError(http.StatusUnauthorized, "Unauthorized", nil)
	}

	role, ok := middleware.BoardRoleFromContext(r.Context())
	if !ok {
		return httputil.NewAPIError(http.StatusForbidden, "Forbidden", nil)
	}
	if core.BoardRole(role) == core.RoleOwner {
		return httputil.NewAPIError(http.StatusForbidden, "Owner cannot leave — transfer ownership first", nil)
	}

	leaving := memberSnapshot(r.Context(), h.boardService, h.activity, boardID, userID)
	if err := h.boardService.RemoveBoardMember(r.Context(), boardID, userID); err != nil {
		return httputil.NewAPIError(http.StatusInternalServerError, "Failed to leave board", err)
	}
	broadcastMembers(r.Context(), h.boardService, h.broadcaster, boardID)
	evictFromBoard(h.broadcaster, boardID, userID)
	recordActivity(r.Context(), h.activity, h.broadcaster, memberActivity(boardID, userID, service.EventMemberLeft,
		service.MemberChangedPayload{UserID: userID, Name: leaving.FullName, Role: leaving.Role}))

	w.WriteHeader(http.StatusNoContent)
	return nil
}
