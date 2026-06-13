package dto

import "time"

// InviteLinkResponse — the board's active link. The client builds the full URL
// (origin + /invite/<token>); the server only owns the token + expiry.
type InviteLinkResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// AcceptInviteResponse tells the client which board to navigate to after a
// successful join.
type AcceptInviteResponse struct {
	BoardID string `json:"board_id"`
}
