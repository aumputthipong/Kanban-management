package middleware

import (
	"net/http"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/core"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
)

func roleRank(r core.BoardRole) int {
	switch r {
	case core.RoleOwner:
		return 3
	case core.RoleManager:
		return 2
	case core.RoleMember:
		return 1
	}
	return 0
}

// Must be chained after RequireBoardMember. 403 on insufficient role.
func RequireBoardRole(minRole core.BoardRole) func(http.Handler) http.Handler {
	minRank := roleRank(minRole)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := BoardRoleFromContext(r.Context())
			if !ok {
				httputil.RespondError(w, http.StatusForbidden, "Forbidden")
				return
			}
			if roleRank(core.BoardRole(role)) < minRank {
				httputil.RespondError(w, http.StatusForbidden, "You do not have permission to perform this action")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
