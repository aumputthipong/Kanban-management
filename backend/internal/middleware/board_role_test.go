package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/core"
	"github.com/stretchr/testify/assert"
)

// requestWithRole builds a request whose context already carries a board role,
// mimicking what RequireBoardMember would have injected upstream.
func requestWithRole(role string, hasRole bool) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if hasRole {
		r = r.WithContext(contextWithBoardRole(r.Context(), role))
	}
	return r
}

func okHandler(called *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*called = true
		w.WriteHeader(http.StatusOK)
	})
}

// TestRequireBoardRole_Matrix exhaustively verifies the role hierarchy.
// owner > manager > member. Each caller role tries each minimum-role gate;
// pass iff caller rank >= gate rank.
func TestRequireBoardRole_Matrix(t *testing.T) {
	type cell struct {
		caller     core.BoardRole
		minRole    core.BoardRole
		wantStatus int
		wantCalled bool
	}

	cases := []cell{
		// owner — passes every gate
		{core.RoleOwner, core.RoleOwner, http.StatusOK, true},
		{core.RoleOwner, core.RoleManager, http.StatusOK, true},
		{core.RoleOwner, core.RoleMember, http.StatusOK, true},

		// manager — passes manager + member, blocked by owner
		{core.RoleManager, core.RoleOwner, http.StatusForbidden, false},
		{core.RoleManager, core.RoleManager, http.StatusOK, true},
		{core.RoleManager, core.RoleMember, http.StatusOK, true},

		// member — only passes member gate
		{core.RoleMember, core.RoleOwner, http.StatusForbidden, false},
		{core.RoleMember, core.RoleManager, http.StatusForbidden, false},
		{core.RoleMember, core.RoleMember, http.StatusOK, true},
	}

	for _, c := range cases {
		name := string(c.caller) + "_vs_min_" + string(c.minRole)
		t.Run(name, func(t *testing.T) {
			var called bool
			h := RequireBoardRole(c.minRole)(okHandler(&called))

			w := httptest.NewRecorder()
			h.ServeHTTP(w, requestWithRole(string(c.caller), true))

			assert.Equal(t, c.wantStatus, w.Code)
			assert.Equal(t, c.wantCalled, called)
		})
	}
}

// TestRequireBoardRole_MissingRoleContext_Returns403 guards against accidental
// misuse: if a route forgets to chain RequireBoardMember first, the role
// context will be absent. Failing closed (403) is the safe default — the
// alternative would be silently letting anonymous traffic through.
func TestRequireBoardRole_MissingRoleContext_Returns403(t *testing.T) {
	var called bool
	h := RequireBoardRole(core.RoleMember)(okHandler(&called))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, requestWithRole("", false))

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, called, "next must not run when role context is missing")
}

// TestRequireBoardRole_EmptyRoleString_Returns403 — defense-in-depth for
// the same case: even if the key is present but the value is "",
// BoardRoleFromContext returns ok=false, so we still 403.
func TestRequireBoardRole_EmptyRoleString_Returns403(t *testing.T) {
	var called bool
	h := RequireBoardRole(core.RoleMember)(okHandler(&called))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = r.WithContext(context.WithValue(r.Context(), BoardRoleKey, ""))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, called)
}

// A role string outside the known enum (corruption, or a role not yet deployed) ranks 0
// and must lose against any non-zero gate. Failing closed prevents privilege escalation
// if the role column is ever influenced.
func TestRequireBoardRole_UnknownRole_Returns403(t *testing.T) {
	var called bool
	h := RequireBoardRole(core.RoleMember)(okHandler(&called))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, requestWithRole("superadmin", true))

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, called)
}
