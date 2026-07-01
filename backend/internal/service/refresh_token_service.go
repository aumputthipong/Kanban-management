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

// RotateRefreshToken validates the presented raw token, marks it revoked,
// inserts a new one, and returns the new token plus the user identity. If
// the presented token was already revoked we treat that as replay and revoke
// every refresh token for the user — the legitimate client will be forced to
// log in again, but so will the attacker.
func (s *AuthService) RotateRefreshToken(ctx context.Context, rawToken, userAgent, ip string) (RefreshRotationResult, error) {
	if rawToken == "" {
		return RefreshRotationResult{}, ErrRefreshInvalid
	}
	hash := token.HashRefreshToken(rawToken)
	row, err := s.queries.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		return RefreshRotationResult{}, ErrRefreshInvalid
	}

	// Replay: the token was rotated before. The genuine client should be
	// using the replacement by now, so seeing the old one means it was
	// captured somewhere. Burn every session for this user.
	if row.RevokedAt != nil {
		_ = s.queries.RevokeAllRefreshTokensForUser(ctx, row.UserID)
		return RefreshRotationResult{}, ErrRefreshInvalid
	}

	if time.Now().After(row.ExpiresAt) {
		return RefreshRotationResult{}, ErrRefreshExpired
	}

	user, err := s.queries.GetUserByID(ctx, row.UserID)
	if err != nil {
		return RefreshRotationResult{}, fmt.Errorf("load user for rotation: %w", err)
	}

	// Mint the replacement first; only revoke the old row after we know we
	// have something to hand back to the client.
	newRaw, err := token.GenerateRefreshToken()
	if err != nil {
		return RefreshRotationResult{}, fmt.Errorf("generate refresh token: %w", err)
	}
	newID, err := s.queries.InsertRefreshToken(ctx, db.InsertRefreshTokenParams{
		UserID:    row.UserID,
		TokenHash: token.HashRefreshToken(newRaw),
		ExpiresAt: time.Now().Add(token.RefreshTokenDuration()),
		UserAgent: util.StringToPtr(userAgent),
		Ip:        util.StringToPtr(ip),
	})
	if err != nil {
		return RefreshRotationResult{}, fmt.Errorf("insert rotated refresh token: %w", err)
	}
	if err := s.queries.RevokeRefreshToken(ctx, db.RevokeRefreshTokenParams{
		ID:         row.ID,
		ReplacedBy: util.StringToPtr(newID),
	}); err != nil {
		return RefreshRotationResult{}, fmt.Errorf("revoke prior refresh token: %w", err)
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
