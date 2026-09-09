//go:build integration

// Integration tests for BoardService's member methods (member_service.go).
// AddBoardMemberByEmail/RemoveBoardMember/UpdateMemberRole all resolve the
// caller's role by hitting *db.Queries first, then branch on it in Go — a
// mock could confirm the branching but not that the owner-role guard is
// checked against the row actually in the database.
package service_test

import (
	"context"
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

// addPlainMember seeds a second user and adds them to the board directly
// (bypassing AddBoardMemberByEmail, since these tests are about what
// happens to an existing plain member, not about the add path itself).
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
