// internal/dto/board_dto.go
package dto

import "time"

type ColumnResponse struct {
	ID       string         `json:"id"`
	Title    string         `json:"title"`
	Position float64        `json:"position"`
	Category string         `json:"category"`
	Color    *string        `json:"color,omitempty"`
	Cards    []CardResponse `json:"cards"`
}

type CreateBoardRequest struct {
	Title string `json:"title" validate:"required,min=1,max=120"`
}
type MemberSummary struct {
	UserID   string `json:"user_id"`
	FullName string `json:"full_name"`
}

type BoardSummaryResponse struct {
	ID             string          `json:"id"`
	Title          string          `json:"title"`
	Description    string          `json:"description"`
	Color          string          `json:"color"`
	Icon           string          `json:"icon"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	LastAccessedAt *time.Time      `json:"last_accessed_at,omitempty"`
	TotalCards     int             `json:"total_cards"`
	DoneCards      int             `json:"done_cards"`
	Members        []MemberSummary `json:"members"`
}

// BoardResponse is the clean shape returned after a board mutation (PATCH).
// It avoids leaking pgtype fields from db.Board onto the wire.
type BoardResponse struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Color       string   `json:"color"`
	Icon        string   `json:"icon"`
	Budget      *float64 `json:"budget,omitempty"`
}

type UserResponse struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

type UpdateBoardRequest struct {
	Title  *string  `json:"title"  validate:"omitempty,min=1,max=120"`
	Budget *float64 `json:"budget" validate:"omitempty,gte=0"`
	// Appearance. Description "" is a valid clear (column is NOT NULL DEFAULT '');
	// color must be a hex string; icon is one of the known glyph keys mirrored
	// on the client (lib/boardAppearance). omit/null = no change (COALESCE-style
	// read-modify-write in the service).
	Description *string `json:"description" validate:"omitempty,max=160"`
	Color       *string `json:"color"       validate:"omitempty,hexcolor"`
	Icon        *string `json:"icon"        validate:"omitempty,oneof=board rocket target bolt bug"`
}

type BoardMemberResponse struct {
	ID       string `json:"id"`
	Role     string `json:"role"`
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

type AddMemberRequest struct {
	UserID string `json:"user_id" validate:"required,uuid"`
	Role   string `json:"role"    validate:"required,oneof=owner manager member"`
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=owner manager member"`
}

// StashedBoardDTO is one row in the stash (คลังบอร์ด). stashed_at is the
// time the board was stashed — backed by the generic boards.deleted_at column.
// Appearance fields mirror the project-list cards so the row renders the same
// glyph + description.
type StashedBoardDTO struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Color       string    `json:"color"`
	Icon        string    `json:"icon"`
	StashedAt   time.Time `json:"stashed_at"`
}
