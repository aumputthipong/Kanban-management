package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/dto"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMyTasks_ReturnsCardsAndCounts(t *testing.T) {
	svc := &mock.MockBoardService{
		GetMyWorkFn: func(ctx context.Context, opts service.MyWorkOptions) (service.MyWorkResult, error) {
			assert.Equal(t, validUserID, opts.UserID)
			assert.Equal(t, service.MyWorkFilter("all"), opts.Filter)
			assert.False(t, opts.IncludeUnassigned)
			return service.MyWorkResult{
				Cards: []service.MyTaskData{
					{ID: "c1", Title: "T1", BoardID: "b1", Status: "todo", Group: "overdue"},
					{ID: "c2", Title: "T2", BoardID: "b1", Status: "todo", Group: "today"},
				},
				Counts: service.MyWorkCounts{Overdue: 1, Today: 1, Total: 2},
			}, nil
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)
	req := withUserID(httptest.NewRequest(http.MethodGet, "/my-tasks?filter=all", nil), validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetMyTasks)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var got dto.MyWorkResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Len(t, got.Cards, 2)
	assert.Equal(t, "overdue", got.Cards[0].Group)
	assert.Equal(t, 2, got.Counts.Total)
	assert.Equal(t, 1, got.Counts.Overdue)
	assert.Equal(t, 1, got.Counts.Today)
}

func TestGetMyTasks_FilterFromQuery_IncludeFromSettings(t *testing.T) {
	var captured service.MyWorkOptions
	svc := &mock.MockBoardService{
		GetMyWorkFn: func(ctx context.Context, opts service.MyWorkOptions) (service.MyWorkResult, error) {
			captured = opts
			return service.MyWorkResult{}, nil
		},
	}
	settings := &mock.MockUserSettingsService{
		GetFn: func(ctx context.Context, userID string) (service.UserSettingsData, error) {
			return service.UserSettingsData{
				UserID:         userID,
				DefaultLanding: "today",
				ShowAllCards:   true,
				Timezone:       "Asia/Bangkok",
			}, nil
		},
	}
	h := NewBoardHandler(svc, settings, nil, nil)
	req := withUserID(
		httptest.NewRequest(http.MethodGet, "/my-tasks?filter=today&include_unassigned=false", nil),
		validUserID,
	)
	httputil.MakeHandler(h.GetMyTasks)(httptest.NewRecorder(), req)

	assert.Equal(t, service.MyWorkFilter("today"), captured.Filter)
	// include_unassigned in the URL is intentionally ignored; settings wins.
	assert.True(t, captured.IncludeUnassigned)
	assert.False(t, captured.Today.IsZero(), "service should receive a non-zero Today pivot")
}

func TestGetMyTasks_Unauthorized(t *testing.T) {
	h := NewBoardHandler(&mock.MockBoardService{}, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/my-tasks", nil) // no user ctx
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetMyTasks)(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetMyTasks_ServiceError(t *testing.T) {
	svc := &mock.MockBoardService{
		GetMyWorkFn: func(ctx context.Context, opts service.MyWorkOptions) (service.MyWorkResult, error) {
			return service.MyWorkResult{}, errors.New("db down")
		},
	}
	h := NewBoardHandler(svc, nil, nil, nil)
	req := withUserID(httptest.NewRequest(http.MethodGet, "/my-tasks", nil), validUserID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetMyTasks)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCompleteMyTask_Success_RecordsActivityAndBroadcastsMove(t *testing.T) {
	completedAt := time.Date(2026, 9, 17, 10, 0, 0, 0, time.UTC)
	svc := &mock.MockBoardService{
		CompleteMyTaskFn: func(ctx context.Context, cardID, userID string) (service.CompleteMyTaskResult, error) {
			assert.Equal(t, validCardID, cardID)
			assert.Equal(t, validUserID, userID)
			return service.CompleteMyTaskResult{
				OK:          true,
				BoardID:     validBoardID,
				CardTitle:   "Ship docs",
				ColumnID:    validColumnID,
				Position:    65536,
				CompletedAt: &completedAt,
			}, nil
		},
	}
	var recorded service.RecordParams
	recorder := &spyRecorder{
		record: func(ctx context.Context, p service.RecordParams) error { recorded = p; return nil },
	}
	bc := &mock.MockBroadcaster{}
	h := NewBoardHandler(svc, nil, recorder, bc)
	req := chiCtx(
		withUserID(httptest.NewRequest(http.MethodPost, "/my-tasks/"+validCardID+"/complete", nil), validUserID),
		"cardID", validCardID,
	)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CompleteMyTask)(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, service.EventCardDoneToggled, recorded.EventType)
	assert.Equal(t, validBoardID, recorded.BoardID)
	assert.Equal(t, validUserID, recorded.ActorID)

	// Activity first (audit log is the source of truth), then the move.
	require.Len(t, bc.Sent, 2)
	var activity struct {
		Type string `json:"type"`
	}
	require.NoError(t, json.Unmarshal(bc.Sent[0].Message, &activity))
	assert.Equal(t, "ACTIVITY_CREATED", activity.Type)

	var moved struct {
		Type    string `json:"type"`
		Payload struct {
			CardID      string  `json:"card_id"`
			NewColumnID string  `json:"new_column_id"`
			Position    float64 `json:"position"`
			IsDone      bool    `json:"is_done"`
			CompletedAt string  `json:"completed_at"`
		} `json:"payload"`
	}
	require.NoError(t, json.Unmarshal(bc.Sent[1].Message, &moved))
	assert.Equal(t, validBoardID, bc.Sent[1].BoardID)
	assert.Equal(t, "CARD_MOVED", moved.Type)
	assert.Equal(t, validCardID, moved.Payload.CardID)
	assert.Equal(t, validColumnID, moved.Payload.NewColumnID)
	assert.Equal(t, float64(65536), moved.Payload.Position)
	assert.True(t, moved.Payload.IsDone)
	assert.Equal(t, "2026-09-17T10:00:00Z", moved.Payload.CompletedAt)
}

func TestCompleteMyTask_NotAssignee_404(t *testing.T) {
	svc := &mock.MockBoardService{
		CompleteMyTaskFn: func(ctx context.Context, cardID, userID string) (service.CompleteMyTaskResult, error) {
			return service.CompleteMyTaskResult{OK: false}, nil
		},
	}
	bc := &mock.MockBroadcaster{}
	h := NewBoardHandler(svc, nil, nil, bc)
	req := chiCtx(
		withUserID(httptest.NewRequest(http.MethodPost, "/my-tasks/"+validCardID+"/complete", nil), validUserID),
		"cardID", validCardID,
	)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CompleteMyTask)(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Empty(t, bc.Sent, "nothing moved, so nothing is broadcast")
}

// spyRecorder is a tiny ActivityRecorder used by my-tasks handler tests to
// assert the audit row would have been written. It intentionally lives next
// to the test rather than under internal/service/mock — only one test cares
// about Record calls and the surface is two methods.
type spyRecorder struct {
	record      func(ctx context.Context, p service.RecordParams) error
	recordAsync func(p service.RecordParams)
}

func (s *spyRecorder) Record(ctx context.Context, p service.RecordParams) (db.Activity, error) {
	if s.record != nil {
		return db.Activity{}, s.record(ctx, p)
	}
	return db.Activity{}, nil
}
func (s *spyRecorder) RecordAsync(p service.RecordParams) {
	if s.recordAsync != nil {
		s.recordAsync(p)
	}
}
