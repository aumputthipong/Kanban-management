package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── CreateInvite ───────────────────────────────────────────────────────────

func TestCreateInvite_Success(t *testing.T) {
	svc := &mock.MockInviteService{
		CreateInviteFn: func(ctx context.Context, boardID, creatorID string) (service.InviteLink, error) {
			assert.Equal(t, validBoardID, boardID)
			assert.Equal(t, validUserID, creatorID)
			return service.InviteLink{Token: "tok123", ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
	}
	h := NewInviteHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/boards/"+validBoardID+"/invites", nil)
	req = chiCtx(req, "boardID", validBoardID)
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateInvite)(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "tok123", body["token"])
}

func TestCreateInvite_InvalidBoardID_Returns400(t *testing.T) {
	h := NewInviteHandler(&mock.MockInviteService{})
	req := httptest.NewRequest(http.MethodPost, "/boards/bad/invites", nil)
	req = chiCtx(req, "boardID", "not-a-uuid")
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateInvite)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── GetActiveInvite ────────────────────────────────────────────────────────

func TestGetActiveInvite_None_Returns204(t *testing.T) {
	svc := &mock.MockInviteService{
		GetActiveInviteFn: func(ctx context.Context, boardID string) (service.InviteLink, bool, error) {
			return service.InviteLink{}, false, nil
		},
	}
	h := NewInviteHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/boards/"+validBoardID+"/invites", nil)
	req = chiCtx(req, "boardID", validBoardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetActiveInvite)(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestGetActiveInvite_Found_Returns200(t *testing.T) {
	svc := &mock.MockInviteService{
		GetActiveInviteFn: func(ctx context.Context, boardID string) (service.InviteLink, bool, error) {
			return service.InviteLink{Token: "abc", ExpiresAt: time.Now().Add(time.Hour)}, true, nil
		},
	}
	h := NewInviteHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/boards/"+validBoardID+"/invites", nil)
	req = chiCtx(req, "boardID", validBoardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetActiveInvite)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "abc", body["token"])
}

// ─── AcceptInvite ───────────────────────────────────────────────────────────

func TestAcceptInvite_Success(t *testing.T) {
	svc := &mock.MockInviteService{
		AcceptInviteFn: func(ctx context.Context, token, userID string) (string, error) {
			assert.Equal(t, "tok123", token)
			assert.Equal(t, validUserID, userID)
			return validBoardID, nil
		},
	}
	h := NewInviteHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/invites/tok123/accept", nil)
	req = chiCtx(req, "token", "tok123")
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.AcceptInvite)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, validBoardID, body["board_id"])
}

func TestAcceptInvite_Unauthorized_Returns401(t *testing.T) {
	h := NewInviteHandler(&mock.MockInviteService{})
	req := httptest.NewRequest(http.MethodPost, "/invites/tok123/accept", nil)
	req = chiCtx(req, "token", "tok123") // no userID in context
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.AcceptInvite)(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAcceptInvite_Invalid_Returns404(t *testing.T) {
	svc := &mock.MockInviteService{
		AcceptInviteFn: func(ctx context.Context, token, userID string) (string, error) {
			return "", service.ErrInviteInvalid
		},
	}
	h := NewInviteHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/invites/bad/accept", nil)
	req = chiCtx(req, "token", "bad")
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.AcceptInvite)(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAcceptInvite_Expired_Returns410(t *testing.T) {
	svc := &mock.MockInviteService{
		AcceptInviteFn: func(ctx context.Context, token, userID string) (string, error) {
			return "", service.ErrInviteExpired
		},
	}
	h := NewInviteHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/invites/old/accept", nil)
	req = chiCtx(req, "token", "old")
	req = withUserID(req, validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.AcceptInvite)(w, req)

	assert.Equal(t, http.StatusGone, w.Code)
}
