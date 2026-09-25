package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/core"
	"github.com/stretchr/testify/assert"
)

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

// Fail closed when RequireBoardMember wasn't chained.
func TestRequireBoardRole_MissingRoleContext_Returns403(t *testing.T) {
	var called bool
	h := RequireBoardRole(core.RoleMember)(okHandler(&called))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, requestWithRole("", false))

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, called, "next must not run when role context is missing")
}

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

// Unknown roles rank 0 and must lose — fail closed.
func TestRequireBoardRole_UnknownRole_Returns403(t *testing.T) {
	var called bool
	h := RequireBoardRole(core.RoleMember)(okHandler(&called))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, requestWithRole("superadmin", true))

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, called)
}
