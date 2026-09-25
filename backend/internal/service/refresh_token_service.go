// Refresh-token rotation — see docs/adr/0001.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/token"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/util"
)

var (
	// 401 and force re-login; never reveal which case it was.
	ErrRefreshInvalid = errors.New("refresh token invalid")
	// Separate from invalid so metrics can tell idle users from attacks.
	ErrRefreshExpired = errors.New("refresh token expired")
)

// Stores only the sha256 hash. userAgent and ip are audit-only.
func (s *AuthService) IssueRefreshToken(ctx context.Context, userID, userAgent, ip string) (string, error) {
	raw, err := token.GenerateRefreshToken()
	if err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	_, err = s.queries.InsertRefreshToken(ctx, db.InsertRefreshTokenParams{
		UserID:    userID,
		TokenHash: token.HashRefreshToken(raw),
		ExpiresAt: time.Now().Add(token.RefreshTokenDuration()),
		UserAgent: util.StringToPtr(userAgent),
		Ip:        util.StringToPtr(ip),
	})
	if err != nil {
		return "", fmt.Errorf("insert refresh token: %w", err)
	}
	return raw, nil
}

// UserEmail is needed for the access-token claims.
type RefreshRotationResult struct {
	UserID    string
	UserEmail string
	RawToken  string
}

// Inside this window a replay is two tabs racing; outside it burns the family (docs/adr/0001).
const rotationRaceWindow = 30 * time.Second

// One transaction with a row lock — without it concurrent refreshes both mint.
func (s *AuthService) RotateRefreshToken(ctx context.Context, rawToken, userAgent, ip string) (RefreshRotationResult, error) {
	if rawToken == "" {
		return RefreshRotationResult{}, ErrRefreshInvalid
	}
	hash := token.HashRefreshToken(rawToken)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return RefreshRotationResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)
	qtx := s.queries.WithTx(tx)

	row, err := qtx.LockRefreshTokenByHash(ctx, hash)
	if err != nil {
		return RefreshRotationResult{}, ErrRefreshInvalid
	}

	if row.RevokedAt != nil {
		raced := row.ReplacedBy != nil && time.Since(*row.RevokedAt) < rotationRaceWindow
		if !raced {
			if err := qtx.RevokeAllRefreshTokensForUser(ctx, row.UserID); err != nil {
				return RefreshRotationResult{}, fmt.Errorf("revoke token family: %w", err)
			}
			// Commit the burn first — the deferred rollback would undo it.
			if err := tx.Commit(ctx); err != nil {
				return RefreshRotationResult{}, fmt.Errorf("commit token family revoke: %w", err)
			}
		}
		return RefreshRotationResult{}, ErrRefreshInvalid
	}

	if time.Now().After(row.ExpiresAt) {
		return RefreshRotationResult{}, ErrRefreshExpired
	}

	user, err := qtx.GetUserByID(ctx, row.UserID)
	if err != nil {
		return RefreshRotationResult{}, fmt.Errorf("load user for rotation: %w", err)
	}

	newRaw, err := token.GenerateRefreshToken()
	if err != nil {
		return RefreshRotationResult{}, fmt.Errorf("generate refresh token: %w", err)
	}
	newID, err := qtx.InsertRefreshToken(ctx, db.InsertRefreshTokenParams{
		UserID:    row.UserID,
		TokenHash: token.HashRefreshToken(newRaw),
		ExpiresAt: time.Now().Add(token.RefreshTokenDuration()),
		UserAgent: util.StringToPtr(userAgent),
		Ip:        util.StringToPtr(ip),
	})
	if err != nil {
		return RefreshRotationResult{}, fmt.Errorf("insert rotated refresh token: %w", err)
	}
	if err := qtx.RevokeRefreshToken(ctx, db.RevokeRefreshTokenParams{
		ID:         row.ID,
		ReplacedBy: util.StringToPtr(newID),
	}); err != nil {
		return RefreshRotationResult{}, fmt.Errorf("revoke prior refresh token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return RefreshRotationResult{}, fmt.Errorf("commit rotation: %w", err)
	}

	return RefreshRotationResult{
		UserID:    user.ID,
		UserEmail: user.Email,
		RawToken:  newRaw,
	}, nil
}

// Missing or already-revoked tokens are ignored — logout never errors.
func (s *AuthService) RevokeRefreshToken(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	row, err := s.queries.GetRefreshTokenByHash(ctx, token.HashRefreshToken(rawToken))
	if err != nil {
		return nil
	}
	if row.RevokedAt != nil {
		return nil
	}
	return s.queries.RevokeRefreshToken(ctx, db.RevokeRefreshTokenParams{
		ID:         row.ID,
		ReplacedBy: nil,
	})
}
