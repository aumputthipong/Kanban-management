// Planning sessions own a flat list of items (REQ/DEC/Q). Items change status but rows
// are never destroyed: drop is reversible and promote keeps a link to the resulting
// card. Promotion is the one cross-table write — card insert plus item stamp in a
// single transaction, so a half-promoted item cannot exist.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/util"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// planningPositionGap mirrors the FE POSITION_GAP — wide enough that appends never
// collide and reorders can use midpoints without renumbering.
const planningPositionGap = 65536.0

type PlanningService struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewPlanningService(pool *pgxpool.Pool, queries *db.Queries) *PlanningService {
	return &PlanningService{pool: pool, queries: queries}
}

var (
	// ErrPlanningItemAlreadyPromoted makes a second promote a 409, not a duplicate card.
	ErrPlanningItemAlreadyPromoted = errors.New("planning item already promoted")
	// ErrPlanningItemDropped is returned when promoting an item the user dropped; it
	// contradicts that decision, so they must un-drop first.
	ErrPlanningItemDropped = errors.New("planning item is dropped")
	// ErrPlanningNoTodoColumn is user-actionable (add a TODO column) — handlers map it to 422.
	ErrPlanningNoTodoColumn = errors.New("board has no TODO column")
	// ErrPlanningNotFound is returned when a session or item lookup hits zero rows.
	ErrPlanningNotFound = errors.New("planning resource not found")
	// ErrPlanningCommentDeleted targets an already soft-deleted comment; 409 so the
	// optimistic UI can revert.
	ErrPlanningCommentDeleted = errors.New("planning comment already deleted")
)

func (s *PlanningService) ListSessionsByBoard(ctx context.Context, boardID string) ([]db.ListPlanningSessionsByBoardRow, error) {
	return s.queries.ListPlanningSessionsByBoard(ctx, boardID)
}

func (s *PlanningService) GetSession(ctx context.Context, sessionID string) (db.PlanningSession, error) {
	return s.queries.GetPlanningSession(ctx, sessionID)
}

func (s *PlanningService) GetSessionBoardID(ctx context.Context, sessionID string) (string, error) {
	return s.queries.GetBoardIDByPlanningSession(ctx, sessionID)
}

func (s *PlanningService) GetItemBoardID(ctx context.Context, itemID string) (string, error) {
	return s.queries.GetBoardIDByPlanningItem(ctx, itemID)
}

// GetItem returns a single planning item. Used by handlers that need the
// row's content for activity log payloads before deleting it.
func (s *PlanningService) GetItem(ctx context.Context, itemID string) (db.PlanningItem, error) {
	return s.queries.GetPlanningItem(ctx, itemID)
}

func (s *PlanningService) ListItems(ctx context.Context, sessionID string) ([]db.PlanningItem, error) {
	return s.queries.ListPlanningItemsBySession(ctx, sessionID)
}

func (s *PlanningService) CreateSession(ctx context.Context, boardID, title string, label, meetingAt *string, createdBy string) (db.PlanningSession, error) {
	return s.queries.CreatePlanningSession(ctx, db.CreatePlanningSessionParams{
		BoardID:   boardID,
		Title:     title,
		Label:     label,
		MeetingAt: util.PtrStringToTimePtr(meetingAt),
		CreatedBy: util.StringToPtr(createdBy),
	})
}

func (s *PlanningService) UpdateSession(ctx context.Context, sessionID string, title, label, meetingAt *string) (db.PlanningSession, error) {
	return s.queries.UpdatePlanningSession(ctx, db.UpdatePlanningSessionParams{
		ID:        sessionID,
		Title:     title,
		Label:     label,
		MeetingAt: util.PtrStringToTimePtr(meetingAt),
	})
}

func (s *PlanningService) DeleteSession(ctx context.Context, sessionID string) error {
	return s.queries.DeletePlanningSession(ctx, sessionID)
}

func (s *PlanningService) CreateItem(ctx context.Context, sessionID, itemType, title string, description *string) (db.PlanningItem, error) {
	maxPos, err := s.queries.GetMaxPlanningItemPosition(ctx, sessionID)
	if err != nil {
		return db.PlanningItem{}, fmt.Errorf("max position: %w", err)
	}
	return s.queries.CreatePlanningItem(ctx, db.CreatePlanningItemParams{
		SessionID:   sessionID,
		Type:        itemType,
		Title:       title,
		Description: description,
		Position:    maxPos + planningPositionGap,
	})
}

func (s *PlanningService) UpdateItem(
	ctx context.Context,
	itemID string,
	itemType, title *string,
	description *string,
	status *string,
	position *float64,
	acceptanceCriteria, implementationNote *string,
) (db.PlanningItem, error) {
	return s.queries.UpdatePlanningItem(ctx, db.UpdatePlanningItemParams{
		ID:                 itemID,
		Type:               itemType,
		Title:              title,
		Description:        description,
		Status:             status,
		Position:           position,
		AcceptanceCriteria: acceptanceCriteria,
		ImplementationNote: implementationNote,
	})
}

func (s *PlanningService) DeleteItem(ctx context.Context, itemID string) error {
	return s.queries.DeletePlanningItem(ctx, itemID)
}

// CardSource backs the card modal's "source" section: the session and item that
// produced this card, plus a few still-open questions from the same meeting.
type CardSource struct {
	SessionID        string
	SessionTitle     string
	SessionLabel     *string
	SessionMeetingAt *time.Time
	SessionBoardID   string
	ItemID           string
	ItemType         string
	ItemTitle        string
	ItemStatus       string
	PendingQuestions []CardSourcePendingQuestion
}

type CardSourcePendingQuestion struct {
	ID    string
	Title string
}

// GetCardSource returns the planning origin of a card, or nil when it was never
// promoted or the source item is gone (promoted_to_card_id is ON DELETE SET NULL, so a
// deleted card orphans the link). pendingLimit caps the open questions surfaced.
func (s *PlanningService) GetCardSource(ctx context.Context, cardID string, pendingLimit int32) (*CardSource, error) {
	row, err := s.queries.GetPlanningSourceByCard(ctx, &cardID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get card source: %w", err)
	}

	if pendingLimit <= 0 {
		pendingLimit = 3
	}
	questions, err := s.queries.ListPendingQuestionsBySession(ctx,
		db.ListPendingQuestionsBySessionParams{
			SessionID: row.SessionID,
			Limit:     pendingLimit,
		})
	if err != nil {
		return nil, fmt.Errorf("list pending questions: %w", err)
	}

	pending := make([]CardSourcePendingQuestion, len(questions))
	for i, q := range questions {
		pending[i] = CardSourcePendingQuestion{ID: q.ID, Title: q.Title}
	}

	return &CardSource{
		SessionID:        row.SessionID,
		SessionTitle:     row.SessionTitle,
		SessionLabel:     row.SessionLabel,
		SessionMeetingAt: row.SessionMeetingAt,
		SessionBoardID:   row.SessionBoardID,
		ItemID:           row.ItemID,
		ItemType:         row.ItemType,
		ItemTitle:        row.ItemTitle,
		ItemStatus:       row.ItemStatus,
		PendingQuestions: pending,
	}, nil
}

// ─── Item comments ────────────────────────────────────────────────────────

// ListItemComments returns the full thread including soft-deleted comments.
// The UI renders deleted rows as a "deleted" placeholder so the thread's position
// doesn't shift on delete.
func (s *PlanningService) ListItemComments(ctx context.Context, itemID string) ([]db.ListPlanningItemCommentsRow, error) {
	return s.queries.ListPlanningItemComments(ctx, itemID)
}

func (s *PlanningService) GetComment(ctx context.Context, commentID string) (db.PlanningItemComment, error) {
	return s.queries.GetPlanningItemComment(ctx, commentID)
}

// GetCommentBoardID resolves comment → item → session → board in one round-
// trip so the handler can re-check membership without hydrating the full
// comment first.
func (s *PlanningService) GetCommentBoardID(ctx context.Context, commentID string) (string, error) {
	return s.queries.GetBoardIDByPlanningComment(ctx, commentID)
}

func (s *PlanningService) CreateComment(ctx context.Context, itemID, authorID, body string) (db.PlanningItemComment, error) {
	return s.queries.CreatePlanningItemComment(ctx, db.CreatePlanningItemCommentParams{
		ItemID:   itemID,
		AuthorID: authorID,
		Body:     body,
	})
}

// EditComment refuses to touch already-soft-deleted rows (the UPDATE WHERE
// deleted_at IS NULL returns zero rows in that case). Pre-deletion edits
// fall through to a clean 200.
func (s *PlanningService) EditComment(ctx context.Context, commentID, body string) (db.PlanningItemComment, error) {
	row, err := s.queries.UpdatePlanningItemComment(ctx, db.UpdatePlanningItemCommentParams{
		ID:   commentID,
		Body: body,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.PlanningItemComment{}, ErrPlanningCommentDeleted
		}
		return db.PlanningItemComment{}, fmt.Errorf("edit comment: %w", err)
	}
	return row, nil
}

func (s *PlanningService) DeleteComment(ctx context.Context, commentID string) error {
	return s.queries.SoftDeletePlanningItemComment(ctx, commentID)
}

// PromoteItem turns a planning item into a Kanban card in the same board's
// first TODO column. Wrapped in a tx so the cards.insert and the item's
// status flip succeed together — partial promotion would leave the item
// looking unpromoted while a stray card sat on the board.
func (s *PlanningService) PromoteItem(ctx context.Context, itemID, userID string) (db.PlanningItem, db.CreateCardRow, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.PlanningItem{}, db.CreateCardRow{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)
	qtx := s.queries.WithTx(tx)

	// Lock the row for the duration of the tx. Without FOR UPDATE, two
	// concurrent promoters on the same item both see status='live' at
	// READ COMMITTED, both pass the "already promoted?" check below,
	// and both go on to create a card — producing duplicate Kanban
	// cards from a single planning item. The lock serializes them so
	// the second caller sees the freshly written status='promoted'
	// once the first commits.
	item, err := qtx.LockPlanningItemForUpdate(ctx, itemID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.PlanningItem{}, db.CreateCardRow{}, ErrPlanningNotFound
		}
		return db.PlanningItem{}, db.CreateCardRow{}, fmt.Errorf("load item: %w", err)
	}
	if item.Status == "promoted" {
		return db.PlanningItem{}, db.CreateCardRow{}, ErrPlanningItemAlreadyPromoted
	}
	// Block dropped items. A dropped status is the user's explicit "not
	// doing this" decision; promoting it would silently flip a no-op into
	// real work. They must un-drop (status → live) first.
	if item.Status == "dropped" {
		return db.PlanningItem{}, db.CreateCardRow{}, ErrPlanningItemDropped
	}

	boardID, err := qtx.GetBoardIDByPlanningSession(ctx, item.SessionID)
	if err != nil {
		return db.PlanningItem{}, db.CreateCardRow{}, fmt.Errorf("resolve board: %w", err)
	}

	col, err := qtx.GetColumnByBoardAndCategory(ctx, db.GetColumnByBoardAndCategoryParams{
		BoardID:  boardID,
		Category: "TODO",
	})
	if err != nil {
		// pgx.ErrNoRows here means the board has no column with
		// category='TODO'. Surface as a typed error so the handler can
		// turn it into a 422 with an actionable message instead of a
		// generic 500.
		if errors.Is(err, pgx.ErrNoRows) {
			return db.PlanningItem{}, db.CreateCardRow{}, ErrPlanningNoTodoColumn
		}
		return db.PlanningItem{}, db.CreateCardRow{}, fmt.Errorf("find TODO column: %w", err)
	}

	// Position 0 lands the card at the column's logical top so the user
	// can triage promoted ideas without scrolling.
	// Carry AC + Note forward so the dev opening the resulting card sees the
	// same context the requirement owner captured during planning. Nil
	// passthrough — the columns stay NULL on cards that had nothing
	// attached.
	card, err := qtx.CreateCard(ctx, db.CreateCardParams{
		ColumnID:           col.ID,
		Title:              item.Title,
		Position:           0,
		CreatedBy:          util.StringToPtr(userID),
		AcceptanceCriteria: item.AcceptanceCriteria,
		ImplementationNote: item.ImplementationNote,
	})
	if err != nil {
		return db.PlanningItem{}, db.CreateCardRow{}, fmt.Errorf("create card: %w", err)
	}

	if err := qtx.SetPlanningItemPromoted(ctx, db.SetPlanningItemPromotedParams{
		ID:               item.ID,
		PromotedToCardID: util.StringToPtr(card.ID),
	}); err != nil {
		return db.PlanningItem{}, db.CreateCardRow{}, fmt.Errorf("mark promoted: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.PlanningItem{}, db.CreateCardRow{}, fmt.Errorf("commit: %w", err)
	}

	item.Status = "promoted"
	item.PromotedToCardID = util.StringToPtr(card.ID)
	return item, card, nil
}
