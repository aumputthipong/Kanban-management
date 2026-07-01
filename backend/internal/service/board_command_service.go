// BoardCommandService holds the write operations invoked from the WebSocket
// layer (card move/create/delete/update, column and subtask writes). It is
// split from BoardService so WS handlers don't depend on *db.Queries directly.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/util"
	"github.com/jackc/pgx/v5/pgxpool"
)

const wsPositionGap = 65536.0

type BoardCommandService struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewBoardCommandService(pool *pgxpool.Pool, queries *db.Queries) *BoardCommandService {
	return &BoardCommandService{pool: pool, queries: queries}
}

// ErrEntityBoardMismatch is returned when a WS handler tries to mutate a card
// or column that does not belong to the board the client is connected to.
// This is a defense-in-depth check on top of route-level board membership —
// without it, a member of board A could send WS messages that mutate cards in
// board B by referencing their UUIDs.
var ErrEntityBoardMismatch = errors.New("entity does not belong to this board")

// VerifyCardInBoard returns nil iff the given card belongs to boardID.
// Returns ErrEntityBoardMismatch for cross-board attempts (whether the card
// is in a different board or simply does not exist) so callers cannot
// distinguish "wrong board" from "no such card".
func (s *BoardCommandService) VerifyCardInBoard(ctx context.Context, cardID, boardID string) error {
	owner, err := s.queries.GetBoardIDByCard(ctx, cardID)
	if err != nil {
		return ErrEntityBoardMismatch
	}
	if owner != boardID {
		return ErrEntityBoardMismatch
	}
	return nil
}

// VerifyColumnInBoard returns nil iff the given column belongs to boardID.
// See VerifyCardInBoard for the rationale on collapsing the error cases.
func (s *BoardCommandService) VerifyColumnInBoard(ctx context.Context, columnID, boardID string) error {
	owner, err := s.queries.GetBoardIDByColumn(ctx, columnID)
	if err != nil {
		return ErrEntityBoardMismatch
	}
	if owner != boardID {
		return ErrEntityBoardMismatch
	}
	return nil
}

// -----------------------------
// Card operations
// -----------------------------

type MoveCardResult struct {
	CardTitle   string
	IsDone      bool
	CompletedAt *time.Time
}

func (s *BoardCommandService) MoveCard(ctx context.Context, cardID, newColumnID string, position float64) (MoveCardResult, error) {
	category, err := s.queries.GetColumnCategory(ctx, newColumnID)
	if err != nil {
		return MoveCardResult{}, fmt.Errorf("get column category: %w", err)
	}
	isDone := category == "DONE"
	var completedAt *time.Time
	if isDone {
		now := time.Now()
		completedAt = &now
	}
	if err := s.queries.UpdateCardColumn(ctx, db.UpdateCardColumnParams{
		ColumnID:    newColumnID,
		Position:    position,
		IsDone:      isDone,
		CompletedAt: util.TimeToTimestamptz(completedAt),
		ID:          cardID,
	}); err != nil {
		return MoveCardResult{}, fmt.Errorf("update card column: %w", err)
	}
	var title string
	if card, err := s.queries.GetCard(ctx, cardID); err == nil {
		title = card.Title
	}
	return MoveCardResult{CardTitle: title, IsDone: isDone, CompletedAt: completedAt}, nil
}

// CreateCardWS creates a card through the WS flow, computing a position if the
// client sent none. The card and its subtasks are created in one transaction,
// so a failed subtask rolls the card back too. Optional fields are nil when
// unset; quick-add passes nil/empty for all of them.
func (s *BoardCommandService) CreateCardWS(ctx context.Context, columnID, creatorID, title, priority string, position float64, assigneeID, dueDate, description *string, subtaskTitles []string) (db.CreateCardRow, []db.CardSubtask, error) {
	if position <= 0 {
		maxPos, err := s.queries.GetMaxPositionInColumn(ctx, columnID)
		if err == nil {
			if v, ok := maxPos.(float64); ok {
				position = v + wsPositionGap
			} else {
				position = wsPositionGap
			}
		} else {
			position = wsPositionGap
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.CreateCardRow{}, nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)
	qtx := s.queries.WithTx(tx)

	card, err := qtx.CreateCard(ctx, db.CreateCardParams{
		ColumnID:    columnID,
		Title:       title,
		Position:    position,
		Priority:    util.StringToPtr(priority),
		AssigneeID:  assigneeID,
		DueDate:     util.PtrStringToTimePtr(dueDate),
		CreatedBy:   &creatorID,
		Description: description,
	})
	if err != nil {
		return db.CreateCardRow{}, nil, fmt.Errorf("create card: %w", err)
	}

	subtasks := make([]db.CardSubtask, 0, len(subtaskTitles))
	for i, t := range subtaskTitles {
		sub, err := qtx.CreateSubtask(ctx, db.CreateSubtaskParams{
			CardID:   card.ID,
			Title:    t,
			Position: wsPositionGap * float64(i+1),
		})
		if err != nil {
			return db.CreateCardRow{}, nil, fmt.Errorf("create subtask %d: %w", i, err)
		}
		subtasks = append(subtasks, sub)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.CreateCardRow{}, nil, fmt.Errorf("commit tx: %w", err)
	}
	return card, subtasks, nil
}

// DeleteCard returns the card title before deleting it, for the activity log.
func (s *BoardCommandService) DeleteCard(ctx context.Context, cardID string) (string, error) {
	var title string
	if card, err := s.queries.GetCard(ctx, cardID); err == nil {
		title = card.Title
	}
	if err := s.queries.DeleteCard(ctx, cardID); err != nil {
		return "", fmt.Errorf("delete card: %w", err)
	}
	return title, nil
}

// UpdateCardBasicParams is the field set the WS layer updates (tags excluded).
type UpdateCardBasicParams struct {
	ID             string
	Title          string
	Description    string
	DueDate        string
	AssigneeID     string
	Priority       string
	EstimatedHours float64
}

func (s *BoardCommandService) UpdateCardBasic(ctx context.Context, p UpdateCardBasicParams) error {
	if _, err := s.queries.UpdateCard(ctx, db.UpdateCardParams{
		ID:             p.ID,
		Title:          p.Title,
		Description:    util.StringToPtr(p.Description),
		DueDate:        util.StringToTimePtr(p.DueDate),
		AssigneeID:     util.StringToPtr(p.AssigneeID),
		Priority:       util.StringToPtr(p.Priority),
		EstimatedHours: util.FloatToPgNumeric(p.EstimatedHours),
	}); err != nil {
		return fmt.Errorf("update card: %w", err)
	}
	return nil
}

type ToggleCardDoneResult struct {
	TargetColumnID string
	CardTitle      string
	CompletedAt    *time.Time
}

func (s *BoardCommandService) ToggleCardDone(ctx context.Context, cardID, boardID string, isDone bool) (ToggleCardDoneResult, error) {
	targetCategory := "TODO"
	if isDone {
		targetCategory = "DONE"
	}
	targetCol, err := s.queries.GetColumnByBoardAndCategory(ctx, db.GetColumnByBoardAndCategoryParams{
		BoardID:  boardID,
		Category: targetCategory,
	})
	if err != nil {
		return ToggleCardDoneResult{}, fmt.Errorf("find %s column: %w", targetCategory, err)
	}
	var completedAt *time.Time
	if isDone {
		now := time.Now()
		completedAt = &now
	}
	if err := s.queries.UpdateCardColumn(ctx, db.UpdateCardColumnParams{
		ColumnID:    targetCol.ID,
		Position:    0,
		IsDone:      isDone,
		CompletedAt: util.TimeToTimestamptz(completedAt),
		ID:          cardID,
	}); err != nil {
		return ToggleCardDoneResult{}, fmt.Errorf("toggle card done: %w", err)
	}
	var title string
	if card, err := s.queries.GetCard(ctx, cardID); err == nil {
		title = card.Title
	}
	return ToggleCardDoneResult{
		TargetColumnID: targetCol.ID,
		CardTitle:      title,
		CompletedAt:    completedAt,
	}, nil
}

// -----------------------------
// Column operations
// -----------------------------

// CreateColumn inserts a new column before the DONE column if one exists.
// category and color come from the create-column modal (quick path passes
// "TODO" and nil).
func (s *BoardCommandService) CreateColumn(ctx context.Context, boardID, title, category string, color *string) (db.CreateColumnRow, error) {
	if category != "DONE" {
		category = "TODO"
	}
	doneCol, err := s.queries.GetColumnByBoardAndCategory(ctx, db.GetColumnByBoardAndCategoryParams{
		BoardID:  boardID,
		Category: "DONE",
	})

	var position float64
	if err != nil {
		// No DONE column — append at the end.
		maxPos, _ := s.queries.GetMaxColumnPositionInBoard(ctx, boardID)
		if v, ok := maxPos.(float64); ok {
			position = v + wsPositionGap
		} else {
			position = wsPositionGap
		}
	} else {
		prevPos, _ := s.queries.GetMaxColumnPositionBeforeDone(ctx, boardID)
		prevPosF, _ := prevPos.(float64)
		position = (prevPosF + doneCol.Position) / 2
	}

	return s.queries.CreateColumn(ctx, db.CreateColumnParams{
		BoardID:  boardID,
		Title:    title,
		Position: position,
		Category: category,
		Color:    color,
	})
}

func (s *BoardCommandService) RenameColumn(ctx context.Context, columnID, title string) error {
	if err := s.queries.RenameColumn(ctx, db.RenameColumnParams{ID: columnID, Title: title}); err != nil {
		return fmt.Errorf("rename column: %w", err)
	}
	return nil
}

func (s *BoardCommandService) DeleteColumn(ctx context.Context, columnID string) error {
	if err := s.queries.DeleteColumn(ctx, columnID); err != nil {
		return fmt.Errorf("delete column: %w", err)
	}
	return nil
}

type UpdateColumnParams struct {
	ID       string
	Title    string
	Category string
	Color    *string
}

func (s *BoardCommandService) UpdateColumn(ctx context.Context, p UpdateColumnParams) error {
	if err := s.queries.UpdateColumn(ctx, db.UpdateColumnParams{
		ID:       p.ID,
		Title:    p.Title,
		Category: p.Category,
		Color:    p.Color,
	}); err != nil {
		return fmt.Errorf("update column: %w", err)
	}
	return nil
}
