//go:build integration

// Integration tests for InviteService. CreateInvite runs a transaction
// (revoke-then-create, "a board has at most one live link"); AcceptInvite
// chains three separate reads/writes with idempotent-join semantics. Neither
// property is observable through a mock.
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

// "a board has at most one live link": regenerating must revoke the old
// token, not leave two active ones — that's the whole reason this needs a
// transaction rather than two independent calls.
func TestCreateInvite_Regenerate_OldTokenNoLongerActive(t *testing.T) {
	ctx := context.Background()
	f := newInviteFixture(t)

	first, err := f.svc.CreateInvite(ctx, f.boardID, f.ownerID)
	require.NoError(t, err)
	second, err := f.svc.CreateInvite(ctx, f.boardID, f.ownerID)
	require.NoError(t, err)
	assert.NotEqual(t, first.Token, second.Token)

	_, err = f.svc.AcceptInvite(ctx, first.Token, f.ownerID)
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

	_, err := f.svc.AcceptInvite(ctx, "not-a-real-token", f.ownerID)
	assert.ErrorIs(t, err, service.ErrInviteInvalid)
}

func TestAcceptInvite_Expired_ErrInviteExpired(t *testing.T) {
	ctx := context.Background()
	f := newInviteFixture(t)
	link, err := f.svc.CreateInvite(ctx, f.boardID, f.ownerID)
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx, `UPDATE board_invites SET expires_at = now() - INTERVAL '1 hour' WHERE token = $1`, link.Token)
	require.NoError(t, err)

	_, err = f.svc.AcceptInvite(ctx, link.Token, f.ownerID)
	assert.ErrorIs(t, err, service.ErrInviteExpired)
}

func TestAcceptInvite_NewUser_JoinsAsMember(t *testing.T) {
	ctx := context.Background()
	f := newInviteFixture(t)
	link, err := f.svc.CreateInvite(ctx, f.boardID, f.ownerID)
	require.NoError(t, err)

	joiner, err := f.queries.CreateUser(ctx, db.CreateUserParams{Email: "joiner@test.local", FullName: "Joiner", Provider: "credentials"})
	require.NoError(t, err)

	boardID, err := f.svc.AcceptInvite(ctx, link.Token, joiner.ID)
	require.NoError(t, err)
	assert.Equal(t, f.boardID, boardID)

	role, err := f.queries.GetBoardMemberRole(ctx, db.GetBoardMemberRoleParams{BoardID: f.boardID, UserID: joiner.ID})
	require.NoError(t, err)
	assert.Equal(t, "member", role)
}

// Accepting twice must not error or duplicate the membership row — the
// UNIQUE(board_id, user_id) constraint would reject a naive second insert,
// so this pins that the idempotent-join check actually short-circuits before
// that.
func TestAcceptInvite_AlreadyMember_IdempotentNoDuplicateRow(t *testing.T) {
	ctx := context.Background()
	f := newInviteFixture(t)
	link, err := f.svc.CreateInvite(ctx, f.boardID, f.ownerID)
	require.NoError(t, err)

	boardID, err := f.svc.AcceptInvite(ctx, link.Token, f.ownerID) // owner is already a member
	require.NoError(t, err)
	assert.Equal(t, f.boardID, boardID)

	var count int
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM board_members WHERE board_id = $1 AND user_id = $2`,
		f.boardID, f.ownerID,
	).Scan(&count))
	assert.Equal(t, 1, count)
}
