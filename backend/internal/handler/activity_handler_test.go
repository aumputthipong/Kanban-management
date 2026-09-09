package handler

import (
	"context"
	"encoding/json"
	"errors"
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

func activityRequest(query string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/boards/"+validBoardID+"/activities"+query, nil)
	return chiCtx(r, "boardID", validBoardID)
}

func TestListActivities_Success(t *testing.T) {
	actor := "Nok"
	entity := validCardID
	svc := &mock.MockActivityLister{
		ListFn: func(ctx context.Context, boardID string, before *time.Time, limit int32) ([]service.ActivityItem, error) {
			return []service.ActivityItem{{
				ID: "act-1", BoardID: boardID, ActorID: validUserID, ActorName: &actor,
				EventType: "card.created", EntityType: "card", EntityID: &entity,
				Payload:   []byte(`{"title":"Buy milk"}`),
				CreatedAt: time.Date(2026, 3, 14, 9, 30, 0, 0, time.UTC),
			}}, nil
		},
	}
	w := httptest.NewRecorder()

	httputil.MakeHandler(NewActivityHandler(svc).ListByBoard)(w, activityRequest(""))

	require.Equal(t, http.StatusOK, w.Code)
	var got []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Len(t, got, 1)
	assert.Equal(t, "act-1", got[0]["id"])
	assert.Equal(t, "card.created", got[0]["event_type"])
	assert.Equal(t, "2026-03-14T09:30:00Z", got[0]["created_at"])
	assert.Equal(t, map[string]any{"title": "Buy milk"}, got[0]["payload"])
}

// An activity with no payload must serialise as {} — the feed renderer indexes
// into payload, and a bare null would throw before any row rendered.
func TestListActivities_EmptyPayload_BecomesObject(t *testing.T) {
	svc := &mock.MockActivityLister{
		ListFn: func(ctx context.Context, boardID string, before *time.Time, limit int32) ([]service.ActivityItem, error) {
			return []service.ActivityItem{{ID: "act-1", EventType: "card.deleted"}}, nil
		},
	}
	w := httptest.NewRecorder()

	httputil.MakeHandler(NewActivityHandler(svc).ListByBoard)(w, activityRequest(""))

	require.Equal(t, http.StatusOK, w.Code)
	var got []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, map[string]any{}, got[0]["payload"])
}

func TestListActivities_DefaultsToThirty(t *testing.T) {
	var gotLimit int32
	var gotBefore *time.Time
	svc := &mock.MockActivityLister{
		ListFn: func(ctx context.Context, boardID string, before *time.Time, limit int32) ([]service.ActivityItem, error) {
			gotLimit, gotBefore = limit, before
			return nil, nil
		},
	}
	w := httptest.NewRecorder()

	httputil.MakeHandler(NewActivityHandler(svc).ListByBoard)(w, activityRequest(""))

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, int32(30), gotLimit)
	assert.Nil(t, gotBefore)
}

func TestListActivities_ForwardsCursor(t *testing.T) {
	var gotBefore *time.Time
	svc := &mock.MockActivityLister{
		ListFn: func(ctx context.Context, boardID string, before *time.Time, limit int32) ([]service.ActivityItem, error) {
			gotBefore = before
			return nil, nil
		},
	}
	w := httptest.NewRecorder()

	httputil.MakeHandler(NewActivityHandler(svc).ListByBoard)(w, activityRequest("?before=2026-03-14T09:30:00Z&limit=5"))

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, gotBefore)
	assert.Equal(t, time.Date(2026, 3, 14, 9, 30, 0, 0, time.UTC), gotBefore.UTC())
}

func TestListActivities_BadInput_Returns400(t *testing.T) {
	cases := map[string]string{
		"malformed before":  "?before=yesterday",
		"non-numeric limit": "?limit=lots",
		"zero limit":        "?limit=0",
		"negative limit":    "?limit=-1",
	}
	for name, query := range cases {
		t.Run(name, func(t *testing.T) {
			svc := &mock.MockActivityLister{} // ListFn nil: reaching the service would panic
			w := httptest.NewRecorder()

			httputil.MakeHandler(NewActivityHandler(svc).ListByBoard)(w, activityRequest(query))

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestListActivities_InvalidBoardID_Returns400(t *testing.T) {
	svc := &mock.MockActivityLister{}
	r := chiCtx(httptest.NewRequest(http.MethodGet, "/boards/nope/activities", nil), "boardID", "nope")
	w := httptest.NewRecorder()

	httputil.MakeHandler(NewActivityHandler(svc).ListByBoard)(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListActivities_ServiceError_Returns500(t *testing.T) {
	svc := &mock.MockActivityLister{
		ListFn: func(ctx context.Context, boardID string, before *time.Time, limit int32) ([]service.ActivityItem, error) {
			return nil, errors.New("db error")
		},
	}
	w := httptest.NewRecorder()

	httputil.MakeHandler(NewActivityHandler(svc).ListByBoard)(w, activityRequest(""))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
