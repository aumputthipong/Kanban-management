package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/jackc/pgx/v5"
)

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

// ErrUserNotFound: no account has that email. ErrAlreadyMember: already on the board.
func (s *BoardService) AddBoardMemberByEmail(ctx context.Context, boardID, email, role string) error {
	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("lookup user by email: %w", err)
	}

	// Decided by ON CONFLICT — a prior read would let concurrent invites 500.
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
