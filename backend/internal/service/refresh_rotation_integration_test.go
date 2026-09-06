//go:build integration

// Integration tests for AuthService.RotateRefreshToken against a real Postgres. Rotation
// is a read-check-write across two rows, and a mock cannot observe whether concurrent
// refreshes both mint, or whether a failed step leaves the old token usable (ADR 0001).
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

type refreshFixture struct {
	pool    *pgxpool.Pool
	queries *db.Queries
	svc     *service.AuthService
	userID  string
}

func newRefreshFixture(t *testing.T) *refreshFixture {
	t.Helper()
	ctx := context.Background()
	pool := testutil.NewTestDB(t)
	queries := db.New(pool)
	seed := testutil.NewSeed(t, pool)
	return &refreshFixture{
		pool:    pool,
		queries: queries,
		svc:     service.NewAuthService(pool, queries),
		userID:  seed.User(ctx),
	}
}

// liveTokenCount counts refresh tokens for the user that are still usable.
func (f *refreshFixture) liveTokenCount(ctx context.Context, t *testing.T) int {
	t.Helper()
	var n int
	err := f.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM refresh_tokens WHERE user_id = $1 AND revoked_at IS NULL`,
		f.userID,
	).Scan(&n)
	require.NoError(t, err)
	return n
}

func TestRotateRefreshToken_HappyPath_OldTokenStopsWorking(t *testing.T) {
	ctx := context.Background()
	f := newRefreshFixture(t)

	first, err := f.svc.IssueRefreshToken(ctx, f.userID, "test-agent", "127.0.0.1")
	require.NoError(t, err)

	rotated, err := f.svc.RotateRefreshToken(ctx, first, "test-agent", "127.0.0.1")
	require.NoError(t, err)
	assert.NotEqual(t, first, rotated.RawToken, "rotation must hand back a different token")
	assert.Equal(t, f.userID, rotated.UserID)

	assert.Equal(t, 1, f.liveTokenCount(ctx, t), "exactly one token should be live after rotation")
}

// The test that motivated the transaction plus FOR UPDATE. Without the row lock both
// callers read the token as un-revoked, both insert a replacement, and one token yields
// two valid sessions — the state replay detection exists to make impossible.
func TestRotateRefreshToken_ConcurrentRefresh_OnlyOneMintsAReplacement(t *testing.T) {
	ctx := context.Background()
	f := newRefreshFixture(t)

	raw, err := f.svc.IssueRefreshToken(ctx, f.userID, "test-agent", "127.0.0.1")
	require.NoError(t, err)

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
			_, err := f.svc.RotateRefreshToken(ctx, raw, "test-agent", "127.0.0.1")
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
		case errors.Is(err, service.ErrRefreshInvalid):
			// expected loser: the lock made it read the token as already used
		default:
			t.Errorf("unexpected error from concurrent rotation: %v", err)
		}
	}

	assert.Equal(t, 1, successes, "exactly one concurrent refresh may mint a replacement")
	assert.Equal(t, 1, f.liveTokenCount(ctx, t), "a single token must never yield two live sessions")
}

// A replay that arrives inside rotationRaceWindow is two tabs racing, not
// theft: the caller is rejected but the user's other sessions survive.
func TestRotateRefreshToken_ReplayInsideRaceWindow_KeepsOtherSessionsAlive(t *testing.T) {
	ctx := context.Background()
	f := newRefreshFixture(t)

	raw, err := f.svc.IssueRefreshToken(ctx, f.userID, "tab-1", "127.0.0.1")
	require.NoError(t, err)
	_, err = f.svc.RotateRefreshToken(ctx, raw, "tab-1", "127.0.0.1")
	require.NoError(t, err)

	// The second tab presents the token it read before tab 1 rotated it.
	_, err = f.svc.RotateRefreshToken(ctx, raw, "tab-2", "127.0.0.1")
	assert.ErrorIs(t, err, service.ErrRefreshInvalid, "the racing caller must be rejected")

	assert.Equal(t, 1, f.liveTokenCount(ctx, t),
		"a tab race must not log the user out of their other sessions")
}

// A replay of a token rotated long ago is treated as theft: every refresh
// token for the user is revoked, so the attacker and the victim both lose the
// session and the victim is forced to re-authenticate.
func TestRotateRefreshToken_ReplayAfterRaceWindow_BurnsWholeFamily(t *testing.T) {
	ctx := context.Background()
	f := newRefreshFixture(t)

	stolen, err := f.svc.IssueRefreshToken(ctx, f.userID, "victim", "127.0.0.1")
	require.NoError(t, err)
	_, err = f.svc.RotateRefreshToken(ctx, stolen, "victim", "127.0.0.1")
	require.NoError(t, err)

	// Age the revoke past the race window so the replay reads as theft.
	_, err = f.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = revoked_at - INTERVAL '10 minutes'
		 WHERE user_id = $1 AND revoked_at IS NOT NULL`,
		f.userID,
	)
	require.NoError(t, err)

	_, err = f.svc.RotateRefreshToken(ctx, stolen, "attacker", "10.0.0.1")
	assert.ErrorIs(t, err, service.ErrRefreshInvalid)

	assert.Equal(t, 0, f.liveTokenCount(ctx, t),
		"replay outside the race window must revoke every session for the user")
}

// The revoke of the old token and the insert of its replacement must land
// together. If they could not, a caller that saw an error would still be
// holding a usable token.
func TestRotateRefreshToken_ExpiredToken_LeavesNoReplacementBehind(t *testing.T) {
	ctx := context.Background()
	f := newRefreshFixture(t)

	raw, err := f.svc.IssueRefreshToken(ctx, f.userID, "test-agent", "127.0.0.1")
	require.NoError(t, err)

	_, err = f.pool.Exec(ctx,
		`UPDATE refresh_tokens SET expires_at = now() - INTERVAL '1 hour' WHERE user_id = $1`,
		f.userID,
	)
	require.NoError(t, err)

	_, err = f.svc.RotateRefreshToken(ctx, raw, "test-agent", "127.0.0.1")
	assert.ErrorIs(t, err, service.ErrRefreshExpired)

	var total int
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM refresh_tokens WHERE user_id = $1`, f.userID,
	).Scan(&total))
	assert.Equal(t, 1, total, "a rejected rotation must not leave an orphan replacement row")
}
