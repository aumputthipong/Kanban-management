package middleware

import (
	"net/http"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/core"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
)

// Must be chained after RequireBoardMember. 403 on insufficient role.
func RequireBoardRole(minRole core.BoardRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := BoardRoleFromContext(r.Context())
			if !ok {
				httputil.RespondError(w, http.StatusForbidden, "Forbidden")
				return
			}
			if !core.BoardRole(role).AtLeast(minRole) {
				httputil.RespondError(w, http.StatusForbidden, "You do not have permission to perform this action")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
