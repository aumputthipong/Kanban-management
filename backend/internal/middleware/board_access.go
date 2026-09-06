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

// roleResolver returns the caller's role on the {boardID} board, or
// pgx.ErrNoRows when the board is out of scope for this gate (not a member, or
// the board is in the wrong state — active vs stashed).
type roleResolver func(ctx context.Context, boardID, userID string) (string, error)

type boardContextKey string

// BoardRoleKey is the context.Context key that holds the caller's role on
// the current board (set by RequireBoardMember, read via BoardRoleFromContext).
const BoardRoleKey boardContextKey = "boardRole"

// RequireBoardMember enforces that the authenticated user (from RequireAuth) is
// a member of the {boardID} board, returning 404 on failure and injecting the
// caller's role into the context for a chained RequireBoardRole. The 404-not-403
// choice is anti-enumeration: see docs/adr/0004-membership-gate-returns-404.md.
func RequireBoardMember(svc service.BoardServicer) func(http.Handler) http.Handler {
	return boardMembershipGate(svc.GetBoardMemberRole)
}

// RequireStashedBoardMember mirrors RequireBoardMember for /api/stash: it matches only
// stashed boards, so restore and permanent-delete work while the normal routes stay
// unreachable (RequireBoardMember 404s for stashed boards).
func RequireStashedBoardMember(svc service.BoardServicer) func(http.Handler) http.Handler {
	return boardMembershipGate(svc.GetStashedBoardMemberRole)
}

// boardMembershipGate is the shared body of the board-access middlewares. The
// resolver decides which boards are in scope (active vs stashed); a pgx.ErrNoRows
// from it becomes a 404 (anti-enumeration), not a 403.
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

			// A malformed board ID cannot reference a real board. 404 rather than letting it
			// reach the uuid column, where Postgres 22P02 would surface as a 500 — malformed,
			// nonexistent and not-a-member must stay indistinguishable.
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
