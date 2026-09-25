//go:build integration

package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/testutil"
)

type memberFixture struct {
	queries *db.Queries
	svc     *service.BoardService
	seed    *testutil.SeedHelper
	ownerID string
	boardID string
}

func newMemberFixture(t *testing.T) *memberFixture {
	t.Helper()
	ctx := context.Background()
	pool := testutil.NewTestDB(t)
	queries := db.New(pool)
	seed := testutil.NewSeed(t, pool)
	ownerID := seed.User(ctx)
	return &memberFixture{
		queries: queries,
		svc:     service.NewBoardService(pool, queries),
		seed:    seed,
		ownerID: ownerID,
		boardID: seed.Board(ctx, ownerID),
	}
}

func (f *memberFixture) addPlainMember(ctx context.Context, t *testing.T) string {
	t.Helper()
	userID := f.seed.User(ctx)
	_, err := f.queries.AddBoardMember(ctx, db.AddBoardMemberParams{BoardID: f.boardID, UserID: userID, Role: "member"})
	require.NoError(t, err)
	return userID
}

func TestAddBoardMemberByEmail_Success(t *testing.T) {
	ctx := context.Background()
	f := newMemberFixture(t)
	invitee, err := f.queries.CreateUser(ctx, db.CreateUserParams{Email: "invitee@test.local", FullName: "Invitee", Provider: "credentials"})
	require.NoError(t, err)

	err = f.svc.AddBoardMemberByEmail(ctx, f.boardID, "invitee@test.local", "member")
	require.NoError(t, err)

	role, err := f.queries.GetBoardMemberRole(ctx, db.GetBoardMemberRoleParams{BoardID: f.boardID, UserID: invitee.ID})
	require.NoError(t, err)
	assert.Equal(t, "member", role)
}

func TestAddBoardMemberByEmail_UnknownEmail_ErrUserNotFound(t *testing.T) {
	ctx := context.Background()
	f := newMemberFixture(t)

	err := f.svc.AddBoardMemberByEmail(ctx, f.boardID, "nobody@test.local", "member")
	assert.ErrorIs(t, err, service.ErrUserNotFound)
}

func TestAddBoardMemberByEmail_AlreadyMember_ErrAlreadyMember(t *testing.T) {
	ctx := context.Background()
	f := newMemberFixture(t)
	owner, err := f.queries.GetUserByID(ctx, f.ownerID)
	require.NoError(t, err)

	err = f.svc.AddBoardMemberByEmail(ctx, f.boardID, owner.Email, "member")
	assert.ErrorIs(t, err, service.ErrAlreadyMember)
}

// Concurrent adds: one wins, the rest get ErrAlreadyMember, not a 500.
func TestAddBoardMemberByEmail_Concurrent_OneAddsRestAlreadyMember(t *testing.T) {
	ctx := context.Background()
	f := newMemberFixture(t)

	for round := 0; round < 20; round++ {
		email := fmt.Sprintf("racer%d@test.local", round)
		_, err := f.queries.CreateUser(ctx, db.CreateUserParams{Email: email, FullName: "Racer", Provider: "credentials"})
		require.NoError(t, err)

		errs := raceN(8, func() error {
			return f.svc.AddBoardMemberByEmail(ctx, f.boardID, email, "member")
		})

		successes := 0
		for _, err := range errs {
			if err == nil {
				successes++
				continue
			}
			require.ErrorIs(t, err, service.ErrAlreadyMember, "round %d", round)
		}
		require.Equal(t, 1, successes, "round %d", round)
	}
}

func TestRemoveBoardMember_Owner_Rejected(t *testing.T) {
	ctx := context.Background()
	f := newMemberFixture(t)

	err := f.svc.RemoveBoardMember(ctx, f.boardID, f.ownerID)
	assert.Error(t, err, "the board owner must never be removable")

	role, gerr := f.queries.GetBoardMemberRole(ctx, db.GetBoardMemberRoleParams{BoardID: f.boardID, UserID: f.ownerID})
	require.NoError(t, gerr)
	assert.Equal(t, "owner", role, "the rejected call must not have removed the row")
}

func TestRemoveBoardMember_RegularMember_Success(t *testing.T) {
	ctx := context.Background()
	f := newMemberFixture(t)
	member := f.addPlainMember(ctx, t)

	require.NoError(t, f.svc.RemoveBoardMember(ctx, f.boardID, member))

	_, err := f.queries.GetBoardMemberRole(ctx, db.GetBoardMemberRoleParams{BoardID: f.boardID, UserID: member})
	assert.Error(t, err, "the member row must actually be gone")
}

func TestUpdateMemberRole_Owner_Rejected(t *testing.T) {
	ctx := context.Background()
	f := newMemberFixture(t)

	err := f.svc.UpdateMemberRole(ctx, f.boardID, f.ownerID, "member")
	assert.Error(t, err, "the board owner's role must never change via this path")
}

func TestUpdateMemberRole_RegularMember_Success(t *testing.T) {
	ctx := context.Background()
	f := newMemberFixture(t)
	member := f.addPlainMember(ctx, t)

	require.NoError(t, f.svc.UpdateMemberRole(ctx, f.boardID, member, "manager"))

	role, err := f.queries.GetBoardMemberRole(ctx, db.GetBoardMemberRoleParams{BoardID: f.boardID, UserID: member})
	require.NoError(t, err)
	assert.Equal(t, "manager", role)
}
