//go:build integration

// Integration tests for DemoService. CreateSandbox writes across users, boards,
// columns, cards and planning; PurgeExpired must then delete all of it in an
// order Postgres accepts — activities.actor_id and planning_item_comments.author_id
// are bare references that block a user delete. Neither is observable through a mock.
package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/testutil"
)

type demoFixture struct {
	pool    *pgxpool.Pool
	queries *db.Queries
	svc     *service.DemoService
}

func newDemoFixture(t *testing.T) *demoFixture {
	t.Helper()
	pool := testutil.NewTestDB(t)
	queries := db.New(pool)
	return &demoFixture{pool: pool, queries: queries, svc: service.NewDemoService(pool, queries)}
}

// expireAll ages every sandbox past its deadline. Reaching past the service is
// the point — the alternative is sleeping out DemoValidity.
func (f *demoFixture) expireAll(ctx context.Context, t *testing.T) {
	t.Helper()
	_, err := f.pool.Exec(ctx, "UPDATE users SET demo_expires_at = now() - interval '1 hour' WHERE is_demo")
	require.NoError(t, err)
}

func TestCreateSandbox_SeedsOwnBoardWithCards(t *testing.T) {
	ctx := context.Background()
	f := newDemoFixture(t)

	sandbox, err := f.svc.CreateSandbox(ctx, "")
	require.NoError(t, err)
	assert.NotEmpty(t, sandbox.BoardID)
	assert.True(t, sandbox.ExpireAt.After(time.Now()))

	cols, err := f.queries.GetColumnsByBoardID(ctx, sandbox.BoardID)
	require.NoError(t, err)
	require.Len(t, cols, 4, "sandbox should get the four default columns")

	ids := make([]string, len(cols))
	for i, c := range cols {
		ids[i] = c.ID
	}
	cards, err := f.queries.GetCardsByColumnIDs(ctx, ids)
	require.NoError(t, err)
	assert.Len(t, cards, 8, "the sample fixture seeds eight cards")
}

// Two visitors landing at the same moment must not collide on users.email.
func TestCreateSandbox_Twice_IsolatedBoards(t *testing.T) {
	ctx := context.Background()
	f := newDemoFixture(t)

	first, err := f.svc.CreateSandbox(ctx, "")
	require.NoError(t, err)
	second, err := f.svc.CreateSandbox(ctx, "")
	require.NoError(t, err)

	assert.NotEqual(t, first.UserID, second.UserID)
	assert.NotEqual(t, first.Email, second.Email)
	assert.NotEqual(t, first.BoardID, second.BoardID,
		"each visitor gets their own board — that is the whole point over a shared demo login")
}

func TestPurgeExpired_RemovesUserAndBoard(t *testing.T) {
	ctx := context.Background()
	f := newDemoFixture(t)

	sandbox, err := f.svc.CreateSandbox(ctx, "")
	require.NoError(t, err)
	f.expireAll(ctx, t)

	removed, err := f.svc.PurgeExpired(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, removed)

	_, err = f.queries.GetUserByID(ctx, sandbox.UserID)
	assert.Error(t, err, "the demo user should be gone")

	_, err = f.queries.GetBoardByID(ctx, sandbox.BoardID)
	assert.Error(t, err, "the sandbox board should have gone with its owner")
}

// The purge has to survive a sandbox that recorded audit rows: activities.actor_id
// has no ON DELETE clause, so a stale row would abort the user delete.
func TestPurgeExpired_WithActivities_StillDeletesUser(t *testing.T) {
	ctx := context.Background()
	f := newDemoFixture(t)

	sandbox, err := f.svc.CreateSandbox(ctx, "")
	require.NoError(t, err)

	_, err = f.queries.CreateActivity(ctx, db.CreateActivityParams{
		BoardID:    sandbox.BoardID,
		ActorID:    sandbox.UserID,
		EventType:  "card.created",
		EntityType: "card",
		Payload:    []byte(`{}`),
	})
	require.NoError(t, err)

	f.expireAll(ctx, t)

	removed, err := f.svc.PurgeExpired(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, removed)
}

func TestPurgeExpired_UnexpiredSandbox_IsKept(t *testing.T) {
	ctx := context.Background()
	f := newDemoFixture(t)

	sandbox, err := f.svc.CreateSandbox(ctx, "")
	require.NoError(t, err)

	removed, err := f.svc.PurgeExpired(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 0, removed)

	_, err = f.queries.GetUserByID(ctx, sandbox.UserID)
	assert.NoError(t, err, "a live sandbox must survive the sweep")
}

// Real users have is_demo = false and must never be swept, whatever else happens.
func TestPurgeExpired_RealUser_Untouched(t *testing.T) {
	ctx := context.Background()
	f := newDemoFixture(t)

	realUser, err := f.queries.CreateUser(ctx, db.CreateUserParams{
		Email: "real@example.com", FullName: "Real User", Provider: "credentials",
	})
	require.NoError(t, err)

	_, err = f.svc.CreateSandbox(ctx, "")
	require.NoError(t, err)
	f.expireAll(ctx, t)

	_, err = f.svc.PurgeExpired(ctx)
	require.NoError(t, err)

	_, err = f.queries.GetUserByID(ctx, realUser.ID)
	assert.NoError(t, err, "the sweep must be scoped to is_demo rows")
}
