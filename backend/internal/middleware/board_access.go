package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/go-chi/chi/v5"
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

// RequireBoardMember enforces that the authenticated user is a member of the
// board referenced by the {boardID} URL parameter. It depends on RequireAuth
// having already populated UserIDKey in the request context.
//
// On non-membership it returns 404 (rather than 403) — this is deliberate:
// returning 403 would let an attacker enumerate valid board IDs by checking
// which IDs flip from 404 to 403.
//
// On success it injects the user's role into the request context so a chained
// RequireBoardRole middleware can do the privilege check without another
// database round-trip.
func RequireBoardMember(svc service.BoardServicer) func(http.Handler) http.Handler {
	return boardMembershipGate(svc.GetBoardMemberRole)
}

// RequireStashedBoardMember is the mirror of RequireBoardMember for the
// /api/stash routes: it matches only when the board is stashed, so restore /
// permanent-delete operate on stashed boards. RequireBoardMember 404s for
// stashed boards, keeping their normal routes unreachable.
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
