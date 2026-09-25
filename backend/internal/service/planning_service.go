// Items are never destroyed: drop is reversible, promote keeps a link to the card.
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

// Mirrors the frontend POSITION_GAP — appends never collide, reorders use midpoints.
const planningPositionGap = 65536.0

type PlanningService struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewPlanningService(pool *pgxpool.Pool, queries *db.Queries) *PlanningService {
	return &PlanningService{pool: pool, queries: queries}
}

var (
	ErrPlanningItemAlreadyPromoted = errors.New("planning item already promoted")
	// Promoting a dropped item contradicts that decision — un-drop first.
	ErrPlanningItemDropped = errors.New("planning item is dropped")
	// User-actionable (add a TODO column) — 422.
	ErrPlanningNoTodoColumn = errors.New("board has no TODO column")
	ErrPlanningNotFound     = errors.New("planning resource not found")
	// 409 so the optimistic UI can revert.
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

// nil when never promoted, or the card was deleted (ON DELETE SET NULL).
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

// Item comments

func (s *PlanningService) ListItemComments(ctx context.Context, itemID string) ([]db.ListPlanningItemCommentsRow, error) {
	return s.queries.ListPlanningItemComments(ctx, itemID)
}

func (s *PlanningService) GetComment(ctx context.Context, commentID string) (db.PlanningItemComment, error) {
	return s.queries.GetPlanningItemComment(ctx, commentID)
}

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

// Soft-deleted rows update zero rows → ErrPlanningCommentDeleted.
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

// One transaction: a partial promote would leave a stray card and an unpromoted item.
func (s *PlanningService) PromoteItem(ctx context.Context, itemID, userID string) (db.PlanningItem, db.CreateCardRow, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.PlanningItem{}, db.CreateCardRow{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)
	qtx := s.queries.WithTx(tx)

	// FOR UPDATE: without it two concurrent promoters both see 'live' and both create a card.
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
		if errors.Is(err, pgx.ErrNoRows) {
			return db.PlanningItem{}, db.CreateCardRow{}, ErrPlanningNoTodoColumn
		}
		return db.PlanningItem{}, db.CreateCardRow{}, fmt.Errorf("find TODO column: %w", err)
	}

	// Top of the column so promoted ideas get triaged; AC and note carry forward.
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
