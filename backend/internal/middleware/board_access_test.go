package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service/mock"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testBoardID = "452ae618-9e69-49f5-88a9-47728a5f17ac"
	testUserID  = "550e8400-e29b-41d4-a716-446655440000"
)

func buildRequest(userID, boardID string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/boards/"+boardID, nil)
	if userID != "" {
		r = r.WithContext(context.WithValue(r.Context(), UserIDKey, userID))
	}
	rctx := chi.NewRouteContext()
	if boardID != "" {
		rctx.URLParams.Add("boardID", boardID)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func captureRole(captured *string, called *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*called = true
		if role, ok := BoardRoleFromContext(r.Context()); ok {
			*captured = role
		}
		w.WriteHeader(http.StatusOK)
	})
}

func TestRequireBoardMember_Member_PassesAndInjectsRole(t *testing.T) {
	svc := &mock.MockBoardService{
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			assert.Equal(t, testBoardID, boardID)
			assert.Equal(t, testUserID, userID)
			return "admin", nil
		},
	}

	var role string
	var called bool
	h := RequireBoardMember(svc)(captureRole(&role, &called))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, buildRequest(testUserID, testBoardID))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called, "next handler should have been called")
	assert.Equal(t, "admin", role, "role should be injected into context")
}

// Must stay 404 — a 403 reveals that the board exists.
func TestRequireBoardMember_NonMember_Returns404(t *testing.T) {
	svc := &mock.MockBoardService{
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			return "", pgx.ErrNoRows
		},
	}

	var role string
	var called bool
	h := RequireBoardMember(svc)(captureRole(&role, &called))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, buildRequest(testUserID, testBoardID))

	assert.Equal(t, http.StatusNotFound, w.Code, "non-member MUST get 404, never 403")
	assert.NotEqual(t, http.StatusForbidden, w.Code, "anti-enumeration: see AGENTS.md")
	assert.False(t, called, "next handler must not run when access is denied")

	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.NotContains(t, body["error"], "member", "error message must not leak membership state")
}

func TestRequireBoardMember_NonExistentBoard_Returns404(t *testing.T) {
	// One join query: missing board and non-member are both ErrNoRows.
	svc := &mock.MockBoardService{
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			return "", pgx.ErrNoRows
		},
	}

	var role string
	var called bool
	h := RequireBoardMember(svc)(captureRole(&role, &called))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, buildRequest(testUserID, "00000000-0000-0000-0000-000000000000"))

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.False(t, called)
}

func TestRequireBoardMember_MalformedBoardID_Returns404(t *testing.T) {
	svc := &mock.MockBoardService{
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			t.Fatal("service must not be queried with a malformed board ID")
			return "", nil
		},
	}

	var role string
	var called bool
	h := RequireBoardMember(svc)(captureRole(&role, &called))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, buildRequest(testUserID, testBoardID+"f"))

	assert.Equal(t, http.StatusNotFound, w.Code, "malformed board ID must 404, not 500")
	assert.False(t, called, "next handler must not run for a malformed board ID")
}

func TestRequireBoardMember_MissingUserID_Returns401(t *testing.T) {
	svc := &mock.MockBoardService{
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			t.Fatal("service must not be called when auth context is missing")
			return "", nil
		},
	}

	var role string
	var called bool
	h := RequireBoardMember(svc)(captureRole(&role, &called))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, buildRequest("", testBoardID))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, called)
}

func TestRequireBoardMember_EmptyUserID_Returns401(t *testing.T) {
	// An empty user id must be rejected, not queried.
	svc := &mock.MockBoardService{
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			t.Fatal("service must not be called when userID is empty")
			return "", nil
		},
	}

	r := httptest.NewRequest(http.MethodGet, "/boards/"+testBoardID, nil)
	r = r.WithContext(context.WithValue(r.Context(), UserIDKey, ""))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("boardID", testBoardID)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	var role string
	var called bool
	h := RequireBoardMember(svc)(captureRole(&role, &called))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, called)
}

func TestRequireBoardMember_MissingBoardID_Returns400(t *testing.T) {
	svc := &mock.MockBoardService{
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			t.Fatal("service must not be called when boardID URL param is missing")
			return "", nil
		},
	}

	var role string
	var called bool
	h := RequireBoardMember(svc)(captureRole(&role, &called))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, buildRequest(testUserID, ""))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, called)
}

func TestRequireBoardMember_DBError_Returns500(t *testing.T) {
	svc := &mock.MockBoardService{
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			return "", errors.New("connection refused")
		},
	}

	var role string
	var called bool
	h := RequireBoardMember(svc)(captureRole(&role, &called))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, buildRequest(testUserID, testBoardID))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.False(t, called)
}

func TestRequireStashedBoardMember_StashedOwner_Passes(t *testing.T) {
	svc := &mock.MockBoardService{
		GetStashedBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			assert.Equal(t, testBoardID, boardID)
			return "owner", nil
		},
		GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			t.Fatal("stash gate must use GetStashedBoardMemberRole, not the active one")
			return "", nil
		},
	}

	var role string
	var called bool
	h := RequireStashedBoardMember(svc)(captureRole(&role, &called))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, buildRequest(testUserID, testBoardID))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
	assert.Equal(t, "owner", role)
}

// Each board is reachable through exactly one of the two gates.
func TestRequireStashedBoardMember_ActiveBoard_Returns404(t *testing.T) {
	svc := &mock.MockBoardService{
		GetStashedBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
			return "", pgx.ErrNoRows
		},
	}

	var role string
	var called bool
	h := RequireStashedBoardMember(svc)(captureRole(&role, &called))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, buildRequest(testUserID, testBoardID))

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.False(t, called)
}

func TestRequireBoardMember_RoleVariants(t *testing.T) {
	roles := []string{"owner", "admin", "member", "viewer"}
	for _, want := range roles {
		t.Run(want, func(t *testing.T) {
			svc := &mock.MockBoardService{
				GetBoardMemberRoleFn: func(ctx context.Context, boardID, userID string) (string, error) {
					return want, nil
				},
			}

			var got string
			var called bool
			h := RequireBoardMember(svc)(captureRole(&got, &called))

			w := httptest.NewRecorder()
			h.ServeHTTP(w, buildRequest(testUserID, testBoardID))

			assert.Equal(t, http.StatusOK, w.Code)
			assert.True(t, called)
			assert.Equal(t, want, got)
		})
	}
}
