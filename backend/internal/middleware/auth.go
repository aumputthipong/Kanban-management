// Package middleware contains chi-compatible HTTP middleware: auth, board
// membership/role gating, CORS, rate limiting and security headers. Order matters —
// RequireAuth → RequireBoardMember → RequireBoardRole(minRole) → handler — because
// each stage reads what the previous one injected into the context.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/token"
)

// contextKey is unexported so callers cannot collide with our context values.
type contextKey string

// UserIDKey is the context.Context key that holds the authenticated user's ID.
// Populated by RequireAuth; read via r.Context().Value(UserIDKey).(string).
const UserIDKey contextKey = "userID"

// RequireAuth verifies a JWT from the auth_token cookie (HttpOnly, preferred) or an
// Authorization: Bearer header, and stores the user ID under UserIDKey. It fails with
// a bare 401 rather than leaking which check failed.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tokenStr string

		cookie, err := r.Cookie("auth_token")
		if err == nil {
			tokenStr = cookie.Value
		} else {
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				tokenStr = strings.TrimPrefix(auth, "Bearer ")
			}
		}

		if tokenStr == "" {
			httputil.RespondError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		claims, err := token.Parse(tokenStr)
		if err != nil {
			httputil.RespondError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireWSTicket authenticates a WebSocket handshake from the `ticket` query param,
// injecting the user ID under UserIDKey so the membership gate composes unchanged.
// The handshake cannot carry the cookie — see docs/adr/0005-websocket-ticket-auth.md.
func RequireWSTicket(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ticket := r.URL.Query().Get("ticket")
		if ticket == "" {
			httputil.RespondError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		claims, err := token.ParseWSTicket(ticket)
		if err != nil {
			httputil.RespondError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
