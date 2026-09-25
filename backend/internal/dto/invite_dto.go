package dto

import "time"

// The client builds the full URL; the server only owns the token.
type InviteLinkResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type AcceptInviteResponse struct {
	BoardID string `json:"board_id"`
}
