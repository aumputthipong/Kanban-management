package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Accept-path outcomes the handler maps to HTTP codes.
var (
	ErrInviteInvalid = errors.New("invite link is invalid or revoked")
	ErrInviteExpired = errors.New("invite link has expired")
)

// How long a freshly minted invite link stays valid.
const inviteValidity = 7 * 24 * time.Hour

// InviteService owns shareable board invite links. Kept separate from
// BoardService so neither grows unwieldy; it needs the pool because regenerating
// a link (revoke-old + create-new) is a single transaction.
type InviteService struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewInviteService(pool *pgxpool.Pool, queries *db.Queries) *InviteService {
	return &InviteService{pool: pool, queries: queries}
}

// InviteLink is the usable link surfaced to the inviter — the opaque token (the
// frontend builds the full URL) plus when it stops working.
type InviteLink struct {
	Token     string
	ExpiresAt time.Time
}

// CreateInvite (re)generates the board's link: revokes any active link then
// mints a fresh one, atomically, so a board has at most one live link.
func (s *InviteService) CreateInvite(ctx context.Context, boardID, creatorID string) (InviteLink, error) {
	token, err := generateInviteToken()
	if err != nil {
		return InviteLink{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return InviteLink{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)
	qtx := s.queries.WithTx(tx)

	if err := qtx.RevokeActiveBoardInvites(ctx, boardID); err != nil {
		return InviteLink{}, fmt.Errorf("revoke existing invites: %w", err)
	}
	row, err := qtx.CreateBoardInvite(ctx, db.CreateBoardInviteParams{
		BoardID:   boardID,
		Token:     token,
		CreatedBy: &creatorID,
		ExpiresAt: time.Now().Add(inviteValidity),
	})
	if err != nil {
		return InviteLink{}, fmt.Errorf("create invite: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return InviteLink{}, fmt.Errorf("commit: %w", err)
	}
	return InviteLink{Token: row.Token, ExpiresAt: row.ExpiresAt}, nil
}

// GetActiveInvite returns the board's current usable link, or ok=false when
// there's none (never created / expired / revoked).
func (s *InviteService) GetActiveInvite(ctx context.Context, boardID string) (InviteLink, bool, error) {
	row, err := s.queries.GetActiveBoardInvite(ctx, boardID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return InviteLink{}, false, nil
		}
		return InviteLink{}, false, fmt.Errorf("get active invite: %w", err)
	}
	return InviteLink{Token: row.Token, ExpiresAt: row.ExpiresAt}, true, nil
}

// RevokeInvites turns off every active link for the board ("ปิดลิงก์").
func (s *InviteService) RevokeInvites(ctx context.Context, boardID string) error {
	return s.queries.RevokeActiveBoardInvites(ctx, boardID)
}

// AcceptInvite validates a token and joins the caller to the board as a member.
// Idempotent — an existing member just gets the board id back. Returns
// ErrInviteInvalid (unknown / revoked) or ErrInviteExpired.
func (s *InviteService) AcceptInvite(ctx context.Context, token, userID string) (string, error) {
	inv, err := s.queries.GetBoardInviteByToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrInviteInvalid
		}
		return "", fmt.Errorf("lookup invite: %w", err)
	}
	if inv.RevokedAt != nil {
		return "", ErrInviteInvalid
	}
	if !inv.ExpiresAt.After(time.Now()) {
		return "", ErrInviteExpired
	}

	// Already a member → idempotent no-op join.
	if _, err := s.queries.GetBoardMemberRole(ctx, db.GetBoardMemberRoleParams{
		BoardID: inv.BoardID,
		UserID:  userID,
	}); err == nil {
		return inv.BoardID, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("check membership: %w", err)
	}

	if _, err := s.queries.AddBoardMember(ctx, db.AddBoardMemberParams{
		BoardID: inv.BoardID,
		UserID:  userID,
		Role:    "member",
	}); err != nil {
		return "", fmt.Errorf("add member: %w", err)
	}
	return inv.BoardID, nil
}

// generateInviteToken returns an opaque, URL-safe random token (192 bits).
func generateInviteToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate invite token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
