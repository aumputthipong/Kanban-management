// PATCH semantics for the planning section (Update* request DTOs below):
//
//   - field omitted from JSON       → no change
//   - "field": null                 → no change (same as omit — Go's *string
//                                     cannot distinguish the two after
//                                     json.Unmarshal, so the convention
//                                     collapses them)
//   - "field": "" (nullable column) → stored as empty string. Nullable text
//                                     columns (label, description) treat ""
//                                     and NULL as equivalent at the app
//                                     layer; we don't insert a literal NULL
//                                     because there's no observable
//                                     difference and a "clear" flag would
//                                     bloat the wire payload.
//   - "field": "" (required column) → 400 (title, type, status). The
//                                     validator's `omitempty,min=1` does
//                                     NOT catch this on a *string pointing
//                                     to "" — that path is short-circuited
//                                     by omitempty, so handlers check
//                                     explicitly.
//   - "field": <valid value>        → update
//
// All Update* SQL uses COALESCE(sqlc.narg(...), <existing>) so a nil pointer
// preserves the existing column value end-to-end. See planning_handler.go
// for the explicit required-field empty-string check.
package dto

// PlanningSessionSummary is one row in the sessions list. Counts exclude
// dropped + promoted items so the badge shows what's still actionable
// — anything that's been moved on (to Kanban or to the bin) doesn't add
// to the "still open" signal.
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
	ID        string                `json:"id"`
	BoardID   string                `json:"board_id"`
	Title     string                `json:"title"`
	Label     *string               `json:"label"`
	MeetingAt *string               `json:"meeting_at"`
	CreatedAt string                `json:"created_at"`
	UpdatedAt string                `json:"updated_at"`
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

// CardSourceResponse describes which planning session/item a Kanban card
// was promoted from. Returned by GET /api/cards/{cardID}/source — `null`
// (not 404) when the card wasn't promoted from planning, so the modal can
// render its "ที่มา" section conditionally without an error fork. The
// pending_questions list is capped server-side (default 3) and excludes
// dropped / already-promoted questions — only questions still worth
// re-visiting are surfaced next to the resulting card.
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

// PlanningCommentResponse is one row in an item's comment thread. Body is
// nil when the comment has been soft-deleted — the UI renders that case
// as italic "ถูกลบแล้ว" + the original author + time, so the thread's
// position doesn't shift around as comments are removed.
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
