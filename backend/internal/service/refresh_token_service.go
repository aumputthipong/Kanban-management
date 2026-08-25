// Refresh-token rotation (issue / rotate-on-refresh / revoke) lives on
// AuthService; access-token issuance stays in the token package.
// See docs/adr/0001-opaque-refresh-tokens.md.
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
	// ErrRefreshInvalid covers unknown, malformed, or already-replayed tokens.
	// Callers should respond 401 and force re-login; never reveal which.
	ErrRefreshInvalid = errors.New("refresh token invalid")
	// ErrRefreshExpired separates clock-based expiry from active revocation so
	// metrics can distinguish "idle user" from "potential attack".
	ErrRefreshExpired = errors.New("refresh token expired")
)

// IssueRefreshToken mints a new opaque refresh token, stores its sha256 hash,
// and returns the raw token for the caller to put in a Set-Cookie header.
// userAgent and ip are best-effort attribution — useful for audit but not
// trusted for authorisation.
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

// RefreshRotationResult holds the new opaque token and the user identity to
// re-sign the access token with. UserEmail is needed because access-token
// claims include it and the handler does not have a fresh user row.
type RefreshRotationResult struct {
	UserID    string
	UserEmail string
	RawToken  string
}

// rotationRaceWindow is how long after a rotation a replay of the superseded
// token is read as two clients racing rather than as theft. Browser tabs share
// one cookie jar but refresh independently, so a second tab can present the
// token it read a moment before the first tab rotated it. Inside the window
// that caller is rejected (401) but the user's other sessions are left alone;
// outside it, or for a token revoked by logout rather than by rotation, the
// whole family still burns. See docs/adr/0001.
const rotationRaceWindow = 30 * time.Second

// RotateRefreshToken validates the presented raw token, revokes it, inserts a
// replacement, and returns the new token plus the user identity. Replay of an
// already-rotated token is treated as theft and revokes every refresh token
// for the user — the legitimate client is forced to log in again, but so is
// the attacker.
//
// The whole rotation runs in one transaction and locks the token row: without
// the lock two concurrent refreshes of the same token both read it as unused
// and both mint a replacement, which quietly breaks the single-use property
// that replay detection depends on.
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

	// Replay: the token was rotated before. The genuine client should be using
	// the replacement by now, so seeing the old one means it was captured
	// somewhere — unless it lands inside the race window (see above).
	if row.RevokedAt != nil {
		raced := row.ReplacedBy != nil && time.Since(*row.RevokedAt) < rotationRaceWindow
		if !raced {
			if err := qtx.RevokeAllRefreshTokensForUser(ctx, row.UserID); err != nil {
				return RefreshRotationResult{}, fmt.Errorf("revoke token family: %w", err)
			}
			// Commit the burn before returning the error — the deferred
			// rollback would otherwise undo the defence we just triggered.
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

// RevokeRefreshToken is called on logout. Missing or already-revoked tokens
// are silently ignored — logout should never error from the client's view.
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
