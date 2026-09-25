//go:build integration

package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/testutil"
)

type fixture struct {
	pool    *pgxpool.Pool
	seed    *testutil.SeedHelper
	queries *db.Queries
	svc     *service.PlanningService
	userID  string
	boardID string
	todoID  string
	sessID  string
	itemID  string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	ctx := context.Background()
	pool := testutil.NewTestDB(t)
	seed := testutil.NewSeed(t, pool)
	userID := seed.User(ctx)
	boardID := seed.Board(ctx, userID)
	todoID := seed.Column(ctx, boardID, "TODO", 1)
	sessID := seed.PlanningSession(ctx, boardID, userID)
	itemID := seed.PlanningItem(ctx, sessID)
	return &fixture{
		pool:    pool,
		seed:    seed,
		queries: db.New(pool),
		svc:     service.NewPlanningService(pool, db.New(pool)),
		userID:  userID,
		boardID: boardID,
		todoID:  todoID,
		sessID:  sessID,
		itemID:  itemID,
	}
}

func TestPromoteItem_HappyPath_CreatesCardAndFlipsStatus(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)

	item, card, err := f.svc.PromoteItem(ctx, f.itemID, f.userID)
	require.NoError(t, err)

	assert.Equal(t, "promoted", item.Status)
	require.NotNil(t, item.PromotedToCardID)
	assert.Equal(t, card.ID, *item.PromotedToCardID)
	assert.Equal(t, f.todoID, card.ColumnID)

	persisted, err := f.queries.GetCard(ctx, card.ID)
	require.NoError(t, err)
	assert.Equal(t, "Test Item", persisted.Title)
}

func TestPromoteItem_DoublePromote_Returns409SentinelAndDoesNotDuplicate(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)

	_, card1, err := f.svc.PromoteItem(ctx, f.itemID, f.userID)
	require.NoError(t, err)

	_, _, err = f.svc.PromoteItem(ctx, f.itemID, f.userID)
	require.ErrorIs(t, err, service.ErrPlanningItemAlreadyPromoted)

	item, err := f.queries.GetPlanningItem(ctx, f.itemID)
	require.NoError(t, err)
	require.NotNil(t, item.PromotedToCardID)
	assert.Equal(t, card1.ID, *item.PromotedToCardID)
}

func TestPromoteItem_DroppedItem_Returns422Sentinel(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)

	dropped := "dropped"
	_, err := f.svc.UpdateItem(ctx, f.itemID, nil, nil, nil, &dropped, nil, nil, nil)
	require.NoError(t, err)

	_, _, err = f.svc.PromoteItem(ctx, f.itemID, f.userID)
	require.ErrorIs(t, err, service.ErrPlanningItemDropped)

	item, err := f.queries.GetPlanningItem(ctx, f.itemID)
	require.NoError(t, err)
	assert.Equal(t, "dropped", item.Status)
	assert.Nil(t, item.PromotedToCardID)
}

func TestPromoteItem_BoardWithoutTodoColumn_Returns422Sentinel(t *testing.T) {
	ctx := context.Background()
	pool := testutil.NewTestDB(t)
	seed := testutil.NewSeed(t, pool)
	userID := seed.User(ctx)
	boardID := seed.Board(ctx, userID)
	// No TODO column on purpose.
	seed.Column(ctx, boardID, "IN_PROGRESS", 1)
	seed.Column(ctx, boardID, "DONE", 2)
	sessID := seed.PlanningSession(ctx, boardID, userID)
	itemID := seed.PlanningItem(ctx, sessID)

	svc := service.NewPlanningService(pool, db.New(pool))
	_, _, err := svc.PromoteItem(ctx, itemID, userID)
	require.ErrorIs(t, err, service.ErrPlanningNoTodoColumn)
}

func TestPromoteItem_NotFound_ReturnsSentinel(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	bogusID := "00000000-0000-0000-0000-000000000000"

	_, _, err := f.svc.PromoteItem(ctx, bogusID, f.userID)
	require.ErrorIs(t, err, service.ErrPlanningNotFound)
}

// Under READ COMMITTED two promoters can both see 'live' — exactly one may win.
func TestPromoteItem_ConcurrentPromote_ExactlyOneSucceeds(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)

	const goroutines = 8
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results = make([]error, 0, goroutines)
	)
	start := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, _, err := f.svc.PromoteItem(ctx, f.itemID, f.userID)
			mu.Lock()
			results = append(results, err)
			mu.Unlock()
		}()
	}
	close(start)
	wg.Wait()

	successes := 0
	for _, err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, service.ErrPlanningItemAlreadyPromoted):
			// expected loser
		default:
			t.Errorf("unexpected error from concurrent promote: %v", err)
		}
	}

	assert.GreaterOrEqual(t, successes, 1, "at least one promote must succeed")
	assert.Equal(t, 1, successes, "exactly one concurrent promote should succeed — a race created duplicate cards")
}
