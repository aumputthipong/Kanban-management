package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/middleware"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withBoardRole(r *http.Request, role string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), middleware.BoardRoleKey, role))
}

// GetBoardMembers

func TestGetBoardMembers_InvalidBoardID_Returns400(t *testing.T) {
	svc := &mock.MockBoardService{
		GetBoardMembersFn: func(ctx context.Context, boardID string) ([]db.GetBoardMembersRow, error) {
			t.Fatal("must not query DB for malformed board ID")
			return nil, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/boards/bad/members", nil)
	req = chiCtx(req, "boardID", "not-a-uuid")
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetBoardMembers)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// AddBoardMember

func TestAddBoardMember_Success(t *testing.T) {
	var (
		gotBoardID string
		gotEmail   string
		gotRole    string
	)
	svc := &mock.MockBoardService{
		AddBoardMemberByEmailFn: func(ctx context.Context, boardID, email, role string) error {
			gotBoardID, gotEmail, gotRole = boardID, email, role
			return nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	body := strings.NewReader(`{"email":"newbie@example.com","role":"member"}`)
	req := httptest.NewRequest(http.MethodPost, "/boards/"+validBoardID+"/members", body)
	req = chiCtx(req, "boardID", validBoardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.AddBoardMember)(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, validBoardID, gotBoardID)
	assert.Equal(t, "newbie@example.com", gotEmail)
	assert.Equal(t, "member", gotRole)
}

func TestAddBoardMember_InvalidBoardID_Returns400(t *testing.T) {
	svc := &mock.MockBoardService{}
	h := NewBoardHandler(svc, nil, nil, nil)

	body := strings.NewReader(`{"email":"newbie@example.com","role":"member"}`)
	req := httptest.NewRequest(http.MethodPost, "/boards/bad/members", body)
	req = chiCtx(req, "boardID", "not-a-uuid")
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.AddBoardMember)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// The validator's `oneof` keeps arbitrary strings out of the role column.
func TestAddBoardMember_InvalidRole_Returns400(t *testing.T) {
	svc := &mock.MockBoardService{
		AddBoardMemberByEmailFn: func(ctx context.Context, boardID, email, role string) error {
			t.Fatal("must reject unknown role at validator, never call service")
			return nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	body := strings.NewReader(`{"email":"newbie@example.com","role":"superadmin"}`)
	req := httptest.NewRequest(http.MethodPost, "/boards/"+validBoardID+"/members", body)
	req = chiCtx(req, "boardID", validBoardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.AddBoardMember)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAddBoardMember_MissingEmail_Returns400(t *testing.T) {
	svc := &mock.MockBoardService{}
	h := NewBoardHandler(svc, nil, nil, nil)

	body := strings.NewReader(`{"role":"member"}`)
	req := httptest.NewRequest(http.MethodPost, "/boards/"+validBoardID+"/members", body)
	req = chiCtx(req, "boardID", validBoardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.AddBoardMember)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAddBoardMember_InvalidEmail_Returns400(t *testing.T) {
	svc := &mock.MockBoardService{
		AddBoardMemberByEmailFn: func(ctx context.Context, boardID, email, role string) error {
			t.Fatal("must reject malformed email at validator, never call service")
			return nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	body := strings.NewReader(`{"email":"not-an-email","role":"member"}`)
	req := httptest.NewRequest(http.MethodPost, "/boards/"+validBoardID+"/members", body)
	req = chiCtx(req, "boardID", validBoardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.AddBoardMember)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAddBoardMember_UserNotFound_Returns404(t *testing.T) {
	svc := &mock.MockBoardService{
		AddBoardMemberByEmailFn: func(ctx context.Context, boardID, email, role string) error {
			return service.ErrUserNotFound
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	body := strings.NewReader(`{"email":"ghost@example.com","role":"member"}`)
	req := httptest.NewRequest(http.MethodPost, "/boards/"+validBoardID+"/members", body)
	req = chiCtx(req, "boardID", validBoardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.AddBoardMember)(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAddBoardMember_AlreadyMember_Returns409(t *testing.T) {
	svc := &mock.MockBoardService{
		AddBoardMemberByEmailFn: func(ctx context.Context, boardID, email, role string) error {
			return service.ErrAlreadyMember
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	body := strings.NewReader(`{"email":"existing@example.com","role":"member"}`)
	req := httptest.NewRequest(http.MethodPost, "/boards/"+validBoardID+"/members", body)
	req = chiCtx(req, "boardID", validBoardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.AddBoardMember)(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestAddBoardMember_ServiceError_Returns500(t *testing.T) {
	svc := &mock.MockBoardService{
		AddBoardMemberByEmailFn: func(ctx context.Context, boardID, email, role string) error {
			return errors.New("constraint violation")
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	body := strings.NewReader(`{"email":"newbie@example.com","role":"member"}`)
	req := httptest.NewRequest(http.MethodPost, "/boards/"+validBoardID+"/members", body)
	req = chiCtx(req, "boardID", validBoardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.AddBoardMember)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// RemoveBoardMember

func TestRemoveBoardMember_Success(t *testing.T) {
	var gotBoardID, gotUserID string
	svc := &mock.MockBoardService{
		RemoveBoardMemberFn: func(ctx context.Context, boardID, userID string) error {
			gotBoardID, gotUserID = boardID, userID
			return nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/boards/"+validBoardID+"/members/"+otherUserID, nil)
	req = chiCtx(req, "boardID", validBoardID, "userID", otherUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.RemoveBoardMember)(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, validBoardID, gotBoardID)
	assert.Equal(t, otherUserID, gotUserID)
}

func TestRemoveBoardMember_InvalidBoardID_Returns400(t *testing.T) {
	svc := &mock.MockBoardService{}
	h := NewBoardHandler(svc, nil, nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/boards/bad/members/"+otherUserID, nil)
	req = chiCtx(req, "boardID", "bad", "userID", otherUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.RemoveBoardMember)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRemoveBoardMember_InvalidUserID_Returns400(t *testing.T) {
	svc := &mock.MockBoardService{
		RemoveBoardMemberFn: func(ctx context.Context, boardID, userID string) error {
			t.Fatal("must not call service with invalid user ID")
			return nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/boards/"+validBoardID+"/members/bad", nil)
	req = chiCtx(req, "boardID", validBoardID, "userID", "bad")
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.RemoveBoardMember)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRemoveBoardMember_ServiceError_Returns500(t *testing.T) {
	svc := &mock.MockBoardService{
		RemoveBoardMemberFn: func(ctx context.Context, boardID, userID string) error {
			return errors.New("db down")
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/boards/"+validBoardID+"/members/"+otherUserID, nil)
	req = chiCtx(req, "boardID", validBoardID, "userID", otherUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.RemoveBoardMember)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// UpdateMemberRole

func TestUpdateMemberRole_Success(t *testing.T) {
	var gotRole string
	svc := &mock.MockBoardService{
		UpdateMemberRoleFn: func(ctx context.Context, boardID, userID, role string) error {
			gotRole = role
			return nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	body := strings.NewReader(`{"role":"manager"}`)
	req := httptest.NewRequest(http.MethodPatch, "/boards/"+validBoardID+"/members/"+otherUserID, body)
	req = chiCtx(req, "boardID", validBoardID, "userID", otherUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateMemberRole)(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "manager", gotRole)
}

// Owner is a singleton — PATCH role=owner would create two or demote one.
func TestUpdateMemberRole_PromoteToOwner_Returns400(t *testing.T) {
	svc := &mock.MockBoardService{
		UpdateMemberRoleFn: func(ctx context.Context, boardID, userID, role string) error {
			t.Fatal("must reject owner promotion before service call")
			return nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	body := strings.NewReader(`{"role":"owner"}`)
	req := httptest.NewRequest(http.MethodPatch, "/boards/"+validBoardID+"/members/"+otherUserID, body)
	req = chiCtx(req, "boardID", validBoardID, "userID", otherUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateMemberRole)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "owner")
}

func TestUpdateMemberRole_InvalidRole_Returns400(t *testing.T) {
	svc := &mock.MockBoardService{}
	h := NewBoardHandler(svc, nil, nil, nil)

	body := strings.NewReader(`{"role":"superadmin"}`)
	req := httptest.NewRequest(http.MethodPatch, "/boards/"+validBoardID+"/members/"+otherUserID, body)
	req = chiCtx(req, "boardID", validBoardID, "userID", otherUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateMemberRole)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateMemberRole_InvalidUserID_Returns400(t *testing.T) {
	svc := &mock.MockBoardService{}
	h := NewBoardHandler(svc, nil, nil, nil)

	body := strings.NewReader(`{"role":"member"}`)
	req := httptest.NewRequest(http.MethodPatch, "/boards/"+validBoardID+"/members/bad", body)
	req = chiCtx(req, "boardID", validBoardID, "userID", "bad")
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateMemberRole)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateMemberRole_ServiceError_Returns500(t *testing.T) {
	svc := &mock.MockBoardService{
		UpdateMemberRoleFn: func(ctx context.Context, boardID, userID, role string) error {
			return errors.New("db down")
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	body := strings.NewReader(`{"role":"manager"}`)
	req := httptest.NewRequest(http.MethodPatch, "/boards/"+validBoardID+"/members/"+otherUserID, body)
	req = chiCtx(req, "boardID", validBoardID, "userID", otherUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateMemberRole)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// LeaveBoard

func TestLeaveBoard_Member_Success(t *testing.T) {
	var removedUser string
	svc := &mock.MockBoardService{
		RemoveBoardMemberFn: func(ctx context.Context, boardID, userID string) error {
			removedUser = userID
			return nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/boards/"+validBoardID+"/leave", nil)
	req = chiCtx(req, "boardID", validBoardID)
	req = withUserID(req, validUserID)
	req = withBoardRole(req, "member")
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.LeaveBoard)(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, validUserID, removedUser, "LeaveBoard must remove the caller, not someone else")
}

func TestLeaveBoard_Manager_Success(t *testing.T) {
	svc := &mock.MockBoardService{
		RemoveBoardMemberFn: func(ctx context.Context, boardID, userID string) error {
			return nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/boards/"+validBoardID+"/leave", nil)
	req = chiCtx(req, "boardID", validBoardID)
	req = withUserID(req, validUserID)
	req = withBoardRole(req, "manager")
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.LeaveBoard)(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// An ownerless board is unreachable — this guard must stay.
func TestLeaveBoard_Owner_Returns403(t *testing.T) {
	svc := &mock.MockBoardService{
		RemoveBoardMemberFn: func(ctx context.Context, boardID, userID string) error {
			t.Fatal("owner must not be removed by LeaveBoard")
			return nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/boards/"+validBoardID+"/leave", nil)
	req = chiCtx(req, "boardID", validBoardID)
	req = withUserID(req, validUserID)
	req = withBoardRole(req, "owner")
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.LeaveBoard)(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, strings.ToLower(w.Body.String()), "transfer ownership", "error must guide user toward the fix")
}

func TestLeaveBoard_MissingUserID_Returns401(t *testing.T) {
	svc := &mock.MockBoardService{}
	h := NewBoardHandler(svc, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/boards/"+validBoardID+"/leave", nil)
	req = chiCtx(req, "boardID", validBoardID)
	req = withBoardRole(req, "member")
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.LeaveBoard)(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// Fail closed if the route is wired without RequireBoardMember.
func TestLeaveBoard_MissingRoleContext_Returns403(t *testing.T) {
	svc := &mock.MockBoardService{
		RemoveBoardMemberFn: func(ctx context.Context, boardID, userID string) error {
			t.Fatal("must not call remove without role context")
			return nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/boards/"+validBoardID+"/leave", nil)
	req = chiCtx(req, "boardID", validBoardID)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.LeaveBoard)(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestLeaveBoard_InvalidBoardID_Returns400(t *testing.T) {
	svc := &mock.MockBoardService{}
	h := NewBoardHandler(svc, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/boards/bad/leave", nil)
	req = chiCtx(req, "boardID", "bad")
	req = withUserID(req, validUserID)
	req = withBoardRole(req, "member")
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.LeaveBoard)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLeaveBoard_ServiceError_Returns500(t *testing.T) {
	svc := &mock.MockBoardService{
		RemoveBoardMemberFn: func(ctx context.Context, boardID, userID string) error {
			return errors.New("db down")
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/boards/"+validBoardID+"/leave", nil)
	req = chiCtx(req, "boardID", validBoardID)
	req = withUserID(req, validUserID)
	req = withBoardRole(req, "member")
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.LeaveBoard)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// BOARD_MEMBERS_UPDATED broadcast

func membersAfterChange() func(ctx context.Context, boardID string) ([]db.GetBoardMembersRow, error) {
	return func(ctx context.Context, boardID string) ([]db.GetBoardMembersRow, error) {
		return []db.GetBoardMembersRow{
			{ID: "m-1", Role: "owner", UserID: validUserID, Email: "o@x.io", FullName: "Owner"},
			{ID: "m-2", Role: "manager", UserID: otherUserID, Email: "b@x.io", FullName: "Bob"},
		}, nil
	}
}

func requireMembersBroadcast(t *testing.T, bc *mock.MockBroadcaster) {
	t.Helper()
	require.Len(t, bc.Sent, 1)
	assert.Equal(t, validBoardID, bc.Sent[0].BoardID)
	var msg struct {
		Type    string `json:"type"`
		Payload struct {
			Members []struct {
				UserID   string `json:"user_id"`
				Role     string `json:"role"`
				FullName string `json:"full_name"`
			} `json:"members"`
		} `json:"payload"`
	}
	require.NoError(t, json.Unmarshal(bc.Sent[0].Message, &msg))
	assert.Equal(t, "BOARD_MEMBERS_UPDATED", msg.Type)
	require.Len(t, msg.Payload.Members, 2, "the full list is sent, not the one changed member")
	assert.Equal(t, "manager", msg.Payload.Members[1].Role)
	assert.Equal(t, "Bob", msg.Payload.Members[1].FullName)
}

func TestMembershipChanges_BroadcastMemberList(t *testing.T) {
	cases := map[string]struct {
		call func(h *BoardHandler) httputil.APIFunc
		req  func() *http.Request
	}{
		"add": {
			call: func(h *BoardHandler) httputil.APIFunc { return h.AddBoardMember },
			req: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":"b@x.io","role":"member"}`))
			},
		},
		"remove": {
			call: func(h *BoardHandler) httputil.APIFunc { return h.RemoveBoardMember },
			req:  func() *http.Request { return httptest.NewRequest(http.MethodDelete, "/", nil) },
		},
		"change role": {
			call: func(h *BoardHandler) httputil.APIFunc { return h.UpdateMemberRole },
			req: func() *http.Request {
				return httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(`{"role":"manager"}`))
			},
		},
		"leave": {
			call: func(h *BoardHandler) httputil.APIFunc { return h.LeaveBoard },
			req: func() *http.Request {
				return withBoardRole(withUserID(httptest.NewRequest(http.MethodDelete, "/", nil), otherUserID), "member")
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			svc := &mock.MockBoardService{
				AddBoardMemberByEmailFn: func(ctx context.Context, boardID, email, role string) error { return nil },
				RemoveBoardMemberFn:     func(ctx context.Context, boardID, userID string) error { return nil },
				UpdateMemberRoleFn:      func(ctx context.Context, boardID, userID, role string) error { return nil },
				GetBoardMembersFn:       membersAfterChange(),
			}
			bc := &mock.MockBroadcaster{}
			h := NewBoardHandler(svc, nil, nil, bc)
			req := chiCtx(tc.req(), "boardID", validBoardID, "userID", otherUserID)
			w := httptest.NewRecorder()

			httputil.MakeHandler(tc.call(h))(w, req)

			require.Less(t, w.Code, 300, w.Body.String())
			requireMembersBroadcast(t, bc)
		})
	}
}

func TestRemoveBoardMember_ServiceError_DoesNotBroadcast(t *testing.T) {
	svc := &mock.MockBoardService{
		RemoveBoardMemberFn: func(ctx context.Context, boardID, userID string) error { return errors.New("db error") },
		GetBoardMembersFn:   membersAfterChange(),
	}
	bc := &mock.MockBroadcaster{}
	h := NewBoardHandler(svc, nil, nil, bc)
	req := chiCtx(httptest.NewRequest(http.MethodDelete, "/", nil), "boardID", validBoardID, "userID", otherUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.RemoveBoardMember)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Empty(t, bc.Sent)
}

// Membership is only checked at the handshake, so removal must evict.
func TestRemoveAndLeave_EvictTheUserFromTheBoardRoom(t *testing.T) {
	svc := &mock.MockBoardService{
		RemoveBoardMemberFn: func(ctx context.Context, boardID, userID string) error { return nil },
		GetBoardMembersFn:   membersAfterChange(),
	}

	t.Run("remove", func(t *testing.T) {
		bc := &mock.MockBroadcaster{}
		h := NewBoardHandler(svc, nil, nil, bc)
		req := chiCtx(httptest.NewRequest(http.MethodDelete, "/", nil), "boardID", validBoardID, "userID", otherUserID)
		w := httptest.NewRecorder()

		httputil.MakeHandler(h.RemoveBoardMember)(w, req)

		require.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, []string{validBoardID + "/" + otherUserID}, bc.Evicted)
		assert.Len(t, bc.Sent, 1, "the member list goes out before the eviction")
	})

	t.Run("leave", func(t *testing.T) {
		bc := &mock.MockBroadcaster{}
		h := NewBoardHandler(svc, nil, nil, bc)
		req := withBoardRole(withUserID(httptest.NewRequest(http.MethodDelete, "/", nil), otherUserID), "member")
		req = chiCtx(req, "boardID", validBoardID)
		w := httptest.NewRecorder()

		httputil.MakeHandler(h.LeaveBoard)(w, req)

		require.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, []string{validBoardID + "/" + otherUserID}, bc.Evicted)
	})
}

func TestUpdateMemberRole_DoesNotEvict(t *testing.T) {
	svc := &mock.MockBoardService{
		UpdateMemberRoleFn: func(ctx context.Context, boardID, userID, role string) error { return nil },
		GetBoardMembersFn:  membersAfterChange(),
	}
	bc := &mock.MockBroadcaster{}
	h := NewBoardHandler(svc, nil, nil, bc)
	req := chiCtx(httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(`{"role":"manager"}`)),
		"boardID", validBoardID, "userID", otherUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateMemberRole)(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, bc.Evicted)
}

// Member activity

func spyMemberRecorder(got *[]service.RecordParams) *spyRecorder {
	return &spyRecorder{record: func(ctx context.Context, p service.RecordParams) error {
		*got = append(*got, p)
		return nil
	}}
}

func memberPayload(t *testing.T, p service.RecordParams) service.MemberChangedPayload {
	t.Helper()
	payload, ok := p.Payload.(service.MemberChangedPayload)
	require.True(t, ok, "member events carry MemberChangedPayload")
	return payload
}

func TestAddBoardMember_RecordsMemberAdded(t *testing.T) {
	var got []service.RecordParams
	svc := &mock.MockBoardService{
		AddBoardMemberByEmailFn: func(ctx context.Context, boardID, email, role string) error { return nil },
		GetBoardMembersFn:       membersAfterChange(),
	}
	h := NewBoardHandler(svc, nil, spyMemberRecorder(&got), &mock.MockBroadcaster{})
	req := withUserID(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":"B@x.io","role":"manager"}`)), validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.AddBoardMember)(w, chiCtx(req, "boardID", validBoardID))

	require.Equal(t, http.StatusCreated, w.Code)
	require.Len(t, got, 1)
	assert.Equal(t, service.EventMemberAdded, got[0].EventType)
	assert.Equal(t, validUserID, got[0].ActorID)
	assert.Equal(t, service.MemberChangedPayload{UserID: otherUserID, Name: "Bob", Role: "manager"}, memberPayload(t, got[0]))
}

// Read before delete, or the feed can't name who was removed.
func TestRemoveBoardMember_RecordsNameReadBeforeRemoval(t *testing.T) {
	var got []service.RecordParams
	removed := false
	svc := &mock.MockBoardService{
		RemoveBoardMemberFn: func(ctx context.Context, boardID, userID string) error { removed = true; return nil },
		GetBoardMembersFn: func(ctx context.Context, boardID string) ([]db.GetBoardMembersRow, error) {
			if removed {
				return []db.GetBoardMembersRow{{UserID: validUserID, Role: "owner", FullName: "Owner"}}, nil
			}
			return membersAfterChange()(ctx, boardID)
		},
	}
	h := NewBoardHandler(svc, nil, spyMemberRecorder(&got), &mock.MockBroadcaster{})
	req := withUserID(httptest.NewRequest(http.MethodDelete, "/", nil), validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.RemoveBoardMember)(w, chiCtx(req, "boardID", validBoardID, "userID", otherUserID))

	require.Equal(t, http.StatusNoContent, w.Code)
	require.Len(t, got, 1)
	assert.Equal(t, service.EventMemberRemoved, got[0].EventType)
	assert.Equal(t, "Bob", memberPayload(t, got[0]).Name)
}

func TestUpdateMemberRole_RecordsPreviousRole(t *testing.T) {
	var got []service.RecordParams
	svc := &mock.MockBoardService{
		UpdateMemberRoleFn: func(ctx context.Context, boardID, userID, role string) error { return nil },
		GetBoardMembersFn:  membersAfterChange(),
	}
	h := NewBoardHandler(svc, nil, spyMemberRecorder(&got), &mock.MockBroadcaster{})
	req := withUserID(httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(`{"role":"member"}`)), validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateMemberRole)(w, chiCtx(req, "boardID", validBoardID, "userID", otherUserID))

	require.Equal(t, http.StatusNoContent, w.Code)
	require.Len(t, got, 1)
	assert.Equal(t, service.EventMemberRole, got[0].EventType)
	assert.Equal(t, service.MemberChangedPayload{UserID: otherUserID, Name: "Bob", Role: "member", PreviousRole: "manager"}, memberPayload(t, got[0]))
}

func TestUpdateMemberRole_SameRole_RecordsNothing(t *testing.T) {
	var got []service.RecordParams
	svc := &mock.MockBoardService{
		UpdateMemberRoleFn: func(ctx context.Context, boardID, userID, role string) error { return nil },
		GetBoardMembersFn:  membersAfterChange(),
	}
	h := NewBoardHandler(svc, nil, spyMemberRecorder(&got), &mock.MockBroadcaster{})
	req := withUserID(httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(`{"role":"manager"}`)), validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateMemberRole)(w, chiCtx(req, "boardID", validBoardID, "userID", otherUserID))

	require.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, got)
}

func TestLeaveBoard_RecordsMemberLeft(t *testing.T) {
	var got []service.RecordParams
	svc := &mock.MockBoardService{
		RemoveBoardMemberFn: func(ctx context.Context, boardID, userID string) error { return nil },
		GetBoardMembersFn:   membersAfterChange(),
	}
	h := NewBoardHandler(svc, nil, spyMemberRecorder(&got), &mock.MockBroadcaster{})
	req := withBoardRole(withUserID(httptest.NewRequest(http.MethodDelete, "/", nil), otherUserID), "manager")
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.LeaveBoard)(w, chiCtx(req, "boardID", validBoardID))

	require.Equal(t, http.StatusNoContent, w.Code)
	require.Len(t, got, 1)
	assert.Equal(t, service.EventMemberLeft, got[0].EventType)
	assert.Equal(t, otherUserID, got[0].ActorID)
	assert.Equal(t, "Bob", memberPayload(t, got[0]).Name)
}
