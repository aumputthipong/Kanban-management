package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/jackc/pgx/v5"
)

// Invite-by-email outcomes the handler maps to HTTP codes.
var (
	ErrUserNotFound  = errors.New("no user with that email")
	ErrAlreadyMember = errors.New("user is already a member of this board")
)

type BoardMember struct {
	ID       string
	Role     string
	UserID   string
	Email    string
	FullName string
}

func (s *BoardService) GetBoardMembers(ctx context.Context, boardID string) ([]db.GetBoardMembersRow, error) {
	return s.queries.GetBoardMembers(ctx, boardID)
}

// AddBoardMemberByEmail resolves an exact email to a registered user and adds
// them to the board. Returns ErrUserNotFound when no account has that email
// (the inviter must type it correctly — there is no user list to pick from) and
// ErrAlreadyMember when they're already on the board.
func (s *BoardService) AddBoardMemberByEmail(ctx context.Context, boardID, email, role string) error {
	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("lookup user by email: %w", err)
	}

	// Decided by ON CONFLICT, not a prior membership read: two concurrent invites of the
	// same user would both pass a read and the loser would hit the unique constraint (500).
	_, err = s.queries.JoinBoardMember(ctx, db.JoinBoardMemberParams{
		BoardID: boardID,
		UserID:  user.ID,
		Role:    role,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrAlreadyMember
	}
	return err
}

func (s *BoardService) RemoveBoardMember(ctx context.Context, boardID, userID string) error {
	role, err := s.queries.GetBoardMemberRole(ctx, db.GetBoardMemberRoleParams{
		BoardID: boardID,
		UserID:  userID,
	})
	if err != nil {
		return err
	}
	if role == "owner" {
		return fmt.Errorf("cannot remove the board owner")
	}
	return s.queries.RemoveBoardMember(ctx, db.RemoveBoardMemberParams{
		BoardID: boardID,
		UserID:  userID,
	})
}

func (s *BoardService) UpdateMemberRole(ctx context.Context, boardID, userID string, role string) error {
	currentRole, err := s.queries.GetBoardMemberRole(ctx, db.GetBoardMemberRoleParams{
		BoardID: boardID,
		UserID:  userID,
	})
	if err != nil {
		return err
	}
	if currentRole == "owner" {
		return fmt.Errorf("cannot change the role of the board owner")
	}
	_, err = s.queries.UpdateBoardMemberRole(ctx, db.UpdateBoardMemberRoleParams{
		BoardID: boardID,
		UserID:  userID,
		Role:    role,
	})
	return err
}
