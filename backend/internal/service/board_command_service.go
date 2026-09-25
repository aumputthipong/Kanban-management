// BoardCommandService holds the board write operations, keeping WS handlers off *db.Queries.
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

// Defence in depth: stops a member of board A mutating board B by UUID.
var ErrEntityBoardMismatch = errors.New("entity does not belong to this board")

// Missing and cross-board cards return the same error.
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

// Card operations

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

// Card and subtasks share one transaction.
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

// Column operations

// Inserted before the DONE column if one exists.
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
