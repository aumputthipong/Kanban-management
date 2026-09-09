//go:build integration

// Integration tests for ActivityService — the audit log AGENTS.md calls the
// source of truth. Record hits *db.Queries directly (JSON marshal + insert),
// and RecordAsync hands off to a background worker goroutine a mock cannot
// exercise at all: whether the write actually lands, and whether it survives
// the caller's own context being long gone by the time it runs.
package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/testutil"
)

type activityFixture struct {
	svc     *service.ActivityService
	queries *db.Queries
	userID  string
	boardID string
}

func newActivityFixture(t *testing.T) *activityFixture {
	t.Helper()
	ctx := context.Background()
	pool := testutil.NewTestDB(t)
	queries := db.New(pool)
	seed := testutil.NewSeed(t, pool)
	userID := seed.User(ctx)
	svc := service.NewActivityService(queries)
	t.Cleanup(svc.Stop)
	return &activityFixture{
		svc:     svc,
		queries: queries,
		userID:  userID,
		boardID: seed.Board(ctx, userID),
	}
}

func TestActivityRecord_Success_StoresMarshaledPayload(t *testing.T) {
	ctx := context.Background()
	f := newActivityFixture(t)

	act, err := f.svc.Record(ctx, service.RecordParams{
		BoardID: f.boardID, ActorID: f.userID,
		EventType: service.EventCardCreated, EntityType: service.EntityCard,
		Payload: service.CardCreatedPayload{Title: "New Card", ColumnID: "col-1"},
	})
	require.NoError(t, err)
	assert.Equal(t, service.EventCardCreated, act.EventType)
	// Postgres's jsonb round-trips through its own canonical text form (it
	// adds a space after ":" and ",") rather than preserving Go's compact
	// json.Marshal output byte-for-byte, so compare structurally.
	assert.JSONEq(t, `{"title":"New Card","column_id":"col-1"}`, string(act.Payload))
}

func TestActivityRecord_NilPayload_StoresEmptyObject(t *testing.T) {
	ctx := context.Background()
	f := newActivityFixture(t)

	act, err := f.svc.Record(ctx, service.RecordParams{
		BoardID: f.boardID, ActorID: f.userID,
		EventType: service.EventMemberAdded, EntityType: service.EntityMember,
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{}`, string(act.Payload))
}

// RecordAsync hands the job to a background goroutine and returns immediately
// — there's nothing to assert right after the call. Poll briefly for the row
// to land rather than sleeping a fixed guess, and give up with a clear
// failure if the worker never wrote it.
func TestActivityRecordAsync_JobIsWrittenByTheBackgroundWorker(t *testing.T) {
	ctx := context.Background()
	f := newActivityFixture(t)

	f.svc.RecordAsync(service.RecordParams{
		BoardID: f.boardID, ActorID: f.userID,
		EventType: service.EventCardDeleted, EntityType: service.EntityCard,
		Payload: service.CardDeletedPayload{Title: "Gone"},
	})

	deadline := time.Now().Add(2 * time.Second)
	for {
		items, err := f.svc.List(ctx, f.boardID, nil, 10)
		require.NoError(t, err)
		if len(items) == 1 {
			assert.Equal(t, service.EventCardDeleted, items[0].EventType)
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("async record never landed within 2s (found %d rows)", len(items))
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestActivityList_DefaultsLimitTo30(t *testing.T) {
	ctx := context.Background()
	f := newActivityFixture(t)
	for i := 0; i < 35; i++ {
		_, err := f.svc.Record(ctx, service.RecordParams{
			BoardID: f.boardID, ActorID: f.userID,
			EventType: service.EventCardCreated, EntityType: service.EntityCard,
		})
		require.NoError(t, err)
	}

	items, err := f.svc.List(ctx, f.boardID, nil, 0)
	require.NoError(t, err)
	assert.Len(t, items, 30, "limit<=0 must fall back to the 30-row default, not 0 or unlimited")
}

func TestActivityList_ClampsOverLimitTo30(t *testing.T) {
	ctx := context.Background()
	f := newActivityFixture(t)
	// Need more than 30 rows, or a clamped and an unclamped request would
	// return the same count and this would pass either way.
	for i := 0; i < 40; i++ {
		_, err := f.svc.Record(ctx, service.RecordParams{
			BoardID: f.boardID, ActorID: f.userID,
			EventType: service.EventCardCreated, EntityType: service.EntityCard,
		})
		require.NoError(t, err)
	}

	items, err := f.svc.List(ctx, f.boardID, nil, 500)
	require.NoError(t, err)
	assert.Len(t, items, 30, "a limit over 100 must clamp to the 30-row default, not pass 500 through to the query")
}
