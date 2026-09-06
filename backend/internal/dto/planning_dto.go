// PATCH semantics for the Update* DTOs below (omit or null = no change; "" on
// a required field = 400). See AGENTS.md, "REST API conventions".
package dto

// PlanningSessionSummary is one row in the sessions list. Counts exclude dropped and
// promoted items, so the badge shows only what is still actionable.
type PlanningSessionSummary struct {
	ID            string  `json:"id"`
	BoardID       string  `json:"board_id"`
	Title         string  `json:"title"`
	Label         *string `json:"label"`
	MeetingAt     *string `json:"meeting_at"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
	ReqCount      int64   `json:"req_count"`
	DecCount      int64   `json:"dec_count"`
	QCount        int64   `json:"q_count"`
	PromotedCount int64   `json:"promoted_count"`
	DroppedCount  int64   `json:"dropped_count"`
}

type PlanningSessionDetail struct {
	ID        string                 `json:"id"`
	BoardID   string                 `json:"board_id"`
	Title     string                 `json:"title"`
	Label     *string                `json:"label"`
	MeetingAt *string                `json:"meeting_at"`
	CreatedAt string                 `json:"created_at"`
	UpdatedAt string                 `json:"updated_at"`
	Items     []PlanningItemResponse `json:"items"`
}

type PlanningItemResponse struct {
	ID                 string  `json:"id"`
	SessionID          string  `json:"session_id"`
	Type               string  `json:"type"`
	Title              string  `json:"title"`
	Description        *string `json:"description"`
	Status             string  `json:"status"`
	PromotedToCardID   *string `json:"promoted_to_card_id"`
	Position           float64 `json:"position"`
	CreatedAt          string  `json:"created_at"`
	AcceptanceCriteria *string `json:"acceptance_criteria"`
	ImplementationNote *string `json:"implementation_note"`
}

type CreatePlanningSessionRequest struct {
	Title     string  `json:"title"     validate:"required,min=1,max=255"`
	Label     *string `json:"label"     validate:"omitempty,max=200"`
	MeetingAt *string `json:"meeting_at" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
}

type UpdatePlanningSessionRequest struct {
	Title     *string `json:"title"     validate:"omitempty,min=1,max=255"`
	Label     *string `json:"label"     validate:"omitempty,max=200"`
	MeetingAt *string `json:"meeting_at" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
}

type CreatePlanningItemRequest struct {
	Type        string  `json:"type"        validate:"required,oneof=REQ DEC Q"`
	Title       string  `json:"title"       validate:"required,min=1,max=500"`
	Description *string `json:"description" validate:"omitempty,max=5000"`
}

type UpdatePlanningItemRequest struct {
	Type               *string  `json:"type"                validate:"omitempty,oneof=REQ DEC Q"`
	Title              *string  `json:"title"               validate:"omitempty,min=1,max=500"`
	Description        *string  `json:"description"         validate:"omitempty,max=5000"`
	Status             *string  `json:"status"              validate:"omitempty,oneof=live selected dropped promoted"`
	Position           *float64 `json:"position"`
	AcceptanceCriteria *string  `json:"acceptance_criteria" validate:"omitempty,max=10000"`
	ImplementationNote *string  `json:"implementation_note" validate:"omitempty,max=10000"`
}

// CardSourceResponse describes which planning session and item a card was promoted
// from. GET /api/cards/{cardID}/source returns null (not 404) when there is none.
// pending_questions is capped server-side and excludes dropped and promoted questions.
type CardSourceResponse struct {
	Session          CardSourceSession    `json:"session"`
	Item             CardSourceItem       `json:"item"`
	PendingQuestions []CardSourceQuestion `json:"pending_questions"`
}

type CardSourceSession struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Label     *string `json:"label"`
	MeetingAt *string `json:"meeting_at"`
}

type CardSourceItem struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

type CardSourceQuestion struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// PlanningCommentResponse is one row in an item's thread. Body is nil once soft-deleted;
// the UI renders an italic placeholder with the original author so the thread does not
// shift as comments are removed.
type PlanningCommentResponse struct {
	ID         string  `json:"id"`
	ItemID     string  `json:"item_id"`
	AuthorID   string  `json:"author_id"`
	AuthorName string  `json:"author_name"`
	Body       *string `json:"body"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
	DeletedAt  *string `json:"deleted_at"`
}

type CreatePlanningCommentRequest struct {
	Body string `json:"body" validate:"required,min=1,max=2000"`
}

type UpdatePlanningCommentRequest struct {
	Body string `json:"body" validate:"required,min=1,max=2000"`
}
