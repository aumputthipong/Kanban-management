//go:build integration

package service_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/testutil"
)

type inviteFixture struct {
	pool    *pgxpool.Pool
	queries *db.Queries
	svc     *service.InviteService
	ownerID string
	boardID string
}

func newInviteFixture(t *testing.T) *inviteFixture {
	t.Helper()
	ctx := context.Background()
	pool := testutil.NewTestDB(t)
	queries := db.New(pool)
	seed := testutil.NewSeed(t, pool)
	ownerID := seed.User(ctx)
	return &inviteFixture{
		pool:    pool,
		queries: queries,
		svc:     service.NewInviteService(pool, queries),
		ownerID: ownerID,
		boardID: seed.Board(ctx, ownerID),
	}
}

func TestCreateInvite_Success_ReturnsUsableLink(t *testing.T) {
	ctx := context.Background()
	f := newInviteFixture(t)

	link, err := f.svc.CreateInvite(ctx, f.boardID, f.ownerID)
	require.NoError(t, err)
	assert.NotEmpty(t, link.Token)
	assert.True(t, link.ExpiresAt.After(time.Now()))
}

// Regenerating must revoke the old token.
func TestCreateInvite_Regenerate_OldTokenNoLongerActive(t *testing.T) {
	ctx := context.Background()
	f := newInviteFixture(t)

	first, err := f.svc.CreateInvite(ctx, f.boardID, f.ownerID)
	require.NoError(t, err)
	second, err := f.svc.CreateInvite(ctx, f.boardID, f.ownerID)
	require.NoError(t, err)
	assert.NotEqual(t, first.Token, second.Token)

	_, _, err = f.svc.AcceptInvite(ctx, first.Token, f.ownerID)
	assert.ErrorIs(t, err, service.ErrInviteInvalid, "the regenerated-away token must no longer work")

	active, ok, err := f.svc.GetActiveInvite(ctx, f.boardID)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, second.Token, active.Token)
}

func TestGetActiveInvite_NoneCreated_OkFalse(t *testing.T) {
	ctx := context.Background()
	f := newInviteFixture(t)

	_, ok, err := f.svc.GetActiveInvite(ctx, f.boardID)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestAcceptInvite_UnknownToken_ErrInviteInvalid(t *testing.T) {
	ctx := context.Background()
	f := newInviteFixture(t)

	_, _, err := f.svc.AcceptInvite(ctx, "not-a-real-token", f.ownerID)
	assert.ErrorIs(t, err, service.ErrInviteInvalid)
}

func TestAcceptInvite_Expired_ErrInviteExpired(t *testing.T) {
	ctx := context.Background()
	f := newInviteFixture(t)
	link, err := f.svc.CreateInvite(ctx, f.boardID, f.ownerID)
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx, `UPDATE board_invites SET expires_at = now() - INTERVAL '1 hour' WHERE token = $1`, link.Token)
	require.NoError(t, err)

	_, _, err = f.svc.AcceptInvite(ctx, link.Token, f.ownerID)
	assert.ErrorIs(t, err, service.ErrInviteExpired)
}

func TestAcceptInvite_NewUser_JoinsAsMember(t *testing.T) {
	ctx := context.Background()
	f := newInviteFixture(t)
	link, err := f.svc.CreateInvite(ctx, f.boardID, f.ownerID)
	require.NoError(t, err)

	joiner, err := f.queries.CreateUser(ctx, db.CreateUserParams{Email: "joiner@test.local", FullName: "Joiner", Provider: "credentials"})
	require.NoError(t, err)

	boardID, joined, err := f.svc.AcceptInvite(ctx, link.Token, joiner.ID)
	require.NoError(t, err)
	assert.Equal(t, f.boardID, boardID)
	assert.True(t, joined)

	role, err := f.queries.GetBoardMemberRole(ctx, db.GetBoardMemberRoleParams{BoardID: f.boardID, UserID: joiner.ID})
	require.NoError(t, err)
	assert.Equal(t, "member", role)
}

// Must not duplicate the row or downgrade the owner.
func TestAcceptInvite_AlreadyMember_IdempotentNoDuplicateRow(t *testing.T) {
	ctx := context.Background()
	f := newInviteFixture(t)
	link, err := f.svc.CreateInvite(ctx, f.boardID, f.ownerID)
	require.NoError(t, err)

	boardID, joined, err := f.svc.AcceptInvite(ctx, link.Token, f.ownerID)
	require.NoError(t, err)
	assert.Equal(t, f.boardID, boardID)
	assert.False(t, joined)

	var count int
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM board_members WHERE board_id = $1 AND user_id = $2`,
		f.boardID, f.ownerID,
	).Scan(&count))
	assert.Equal(t, 1, count)

	role, err := f.queries.GetBoardMemberRole(ctx, db.GetBoardMemberRoleParams{BoardID: f.boardID, UserID: f.ownerID})
	require.NoError(t, err)
	assert.Equal(t, "owner", role)
}

// Overlapping accepts: all succeed, exactly one reports joined.
func TestAcceptInvite_ConcurrentSameUser_AllSucceedOneRow(t *testing.T) {
	ctx := context.Background()
	f := newInviteFixture(t)
	link, err := f.svc.CreateInvite(ctx, f.boardID, f.ownerID)
	require.NoError(t, err)

	for round := 0; round < 20; round++ {
		joiner, err := f.queries.CreateUser(ctx, db.CreateUserParams{
			Email: fmt.Sprintf("joiner%d@test.local", round), FullName: "Joiner", Provider: "credentials",
		})
		require.NoError(t, err)

		var joins atomic.Int32
		errs := raceN(8, func() error {
			_, joined, err := f.svc.AcceptInvite(ctx, link.Token, joiner.ID)
			if joined {
				joins.Add(1)
			}
			return err
		})
		for _, err := range errs {
			require.NoError(t, err, "round %d", round)
		}
		require.EqualValues(t, 1, joins.Load(), "round %d: exactly one accept may report joined", round)

		var count int
		require.NoError(t, f.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM board_members WHERE board_id = $1 AND user_id = $2`,
			f.boardID, joiner.ID,
		).Scan(&count))
		require.Equal(t, 1, count, "round %d", round)
	}
}
