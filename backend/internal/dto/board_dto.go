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
	// Optional; omitted = column default.
	Description *string `json:"description" validate:"omitempty,max=160"`
	Color       *string `json:"color"       validate:"omitempty,hexcolor"`
	Icon        *string `json:"icon"        validate:"omitempty,oneof=board rocket target bolt bug"`
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
	// Omitted/null = no change. "" is a valid clear for description.
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
	// Exact email only — no user directory is exposed.
	Email string `json:"email" validate:"required,email"`
	Role  string `json:"role"  validate:"required,oneof=owner manager member"`
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=owner manager member"`
}

// stashed_at is boards.deleted_at.
type StashedBoardDTO struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Color       string    `json:"color"`
	Icon        string    `json:"icon"`
	StashedAt   time.Time `json:"stashed_at"`
}
