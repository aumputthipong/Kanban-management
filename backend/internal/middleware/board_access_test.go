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

// buildRequest assembles a request that mimics the canonical chain
// RequireAuth → RequireBoardMember: userID already in context, boardID in chi URL params.
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

// captureRole is a terminal handler that records what RequireBoardMember
// injected into the context so the test can assert role propagation.
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

// TestRequireBoardMember_NonMember_Returns404 is the anti-enumeration regression:
// AGENTS.md explicitly forbids 403 here — flipping to 403 would let an attacker
// distinguish "board exists but I'm not a member" from "board does not exist".
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
	// Service cannot distinguish "board doesn't exist" from "user isn't a member" —
	// both produce pgx.ErrNoRows from the same join query. This is intentional and
	// is what makes 404-on-non-member sufficient to prevent enumeration.
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

// TestRequireBoardMember_MalformedBoardID_Returns404 is the regression for a
// mistyped / malformed board ID (e.g. a UUID with an extra char). Before the
// gate validated the UUID, the bad value reached the uuid column and Postgres
// rejected it (22P02) as a non-ErrNoRows error -> 500 "Failed to check board
// access", which both looked broken and leaked "this ID is shaped wrong" vs
// "this board doesn't exist". Both must now be an indistinguishable 404.
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
	// testBoardID with an extra trailing char -> 13-char final group, invalid UUID.
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
	// Defense-in-depth: if RequireAuth ever stored an empty string instead of
	// skipping the key entirely, we must still reject — not query the DB with "".
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

// TestRequireStashedBoardMember_StashedOwner_Passes confirms the stash gate
// admits the owner of a stashed board (so restore / permanent-delete work).
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

// TestRequireStashedBoardMember_ActiveBoard_Returns404 is the consistency
// regression: an active (non-stashed) board has no row in the stashed gate, so
// the stash routes 404 — the inverse of RequireBoardMember 404ing for stashed
// boards. Together they keep each board reachable through exactly one surface.
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

// TestRequireBoardMember_RoleVariants checks that every role string the
// service may return is propagated verbatim — RequireBoardRole downstream
// depends on exact string match.
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
