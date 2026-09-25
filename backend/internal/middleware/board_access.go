package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Returns pgx.ErrNoRows when the board is out of scope (not a member, or wrong state).
type roleResolver func(ctx context.Context, boardID, userID string) (string, error)

type boardContextKey string

const BoardRoleKey boardContextKey = "boardRole"

// 404, not 403: anti-enumeration (docs/adr/0004).
func RequireBoardMember(svc service.BoardServicer) func(http.Handler) http.Handler {
	return boardMembershipGate(svc.GetBoardMemberRole)
}

// Matches stashed boards only; RequireBoardMember 404s for them.
func RequireStashedBoardMember(svc service.BoardServicer) func(http.Handler) http.Handler {
	return boardMembershipGate(svc.GetStashedBoardMemberRole)
}

func boardMembershipGate(resolve roleResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := r.Context().Value(UserIDKey).(string)
			if !ok || userID == "" {
				httputil.RespondError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}

			boardID := chi.URLParam(r, "boardID")
			if boardID == "" {
				httputil.RespondError(w, http.StatusBadRequest, "Missing board ID")
				return
			}

			// Malformed, nonexistent and not-a-member must be the same 404 (not a 22P02 500).
			if _, err := uuid.Parse(boardID); err != nil {
				httputil.RespondError(w, http.StatusNotFound, "Not found")
				return
			}

			role, err := resolve(r.Context(), boardID, userID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					httputil.RespondError(w, http.StatusNotFound, "Not found")
					return
				}
				httputil.RespondError(w, http.StatusInternalServerError, "Failed to check board access")
				return
			}

			ctx := contextWithBoardRole(r.Context(), role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
