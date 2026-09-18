package service

import (
	"context"
	"fmt"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/dto"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SubtaskService struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewSubtaskService(pool *pgxpool.Pool) *SubtaskService {
	return &SubtaskService{
		queries: db.New(pool),
		pool:    pool,
	}
}

// CreateSubtask appends a subtask after the card's last one. The card row is locked so
// concurrent appends to one card serialize instead of reading the same MAX(position).
func (s *SubtaskService) CreateSubtask(ctx context.Context, cardID, title string) (db.CardSubtask, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.CardSubtask{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)
	if _, err := qtx.LockCardForUpdate(ctx, cardID); err != nil {
		return db.CardSubtask{}, fmt.Errorf("lock card: %w", err)
	}
	subtask, err := qtx.AppendSubtask(ctx, db.AppendSubtaskParams{CardID: cardID, Title: title})
	if err != nil {
		return db.CardSubtask{}, fmt.Errorf("append subtask: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return db.CardSubtask{}, fmt.Errorf("commit tx: %w", err)
	}
	return subtask, nil
}

func (s *SubtaskService) GetSubtasksByCardID(ctx context.Context, cardID string) ([]db.CardSubtask, error) {

	subtasks, err := s.queries.GetSubtasksByCardID(ctx, cardID)
	if err != nil {
		return nil, fmt.Errorf("get subtasks failed: %w", err)
	}
	if subtasks == nil {
		return []db.CardSubtask{}, nil
	}

	return subtasks, nil
}
// UpdateSubtask patches only the fields present in req. The merge happens in SQL
// (COALESCE) — reading the row into Go first lets concurrent edits of different fields clobber each other.
func (s *SubtaskService) UpdateSubtask(ctx context.Context, subtaskID string, req dto.UpdateSubtaskRequest) (db.CardSubtask, error) {
	updatedSubtask, err := s.queries.UpdateSubtask(ctx, db.UpdateSubtaskParams{
		ID:       subtaskID,
		Title:    req.Title,
		IsDone:   req.IsDone,
		Position: req.Position,
	})
	if err != nil {
		return db.CardSubtask{}, fmt.Errorf("failed to update subtask in db: %w", err)
	}

	return updatedSubtask, nil
}

// DeleteSubtask removes the subtask row.
func (s *SubtaskService) DeleteSubtask(ctx context.Context, subtaskID string) error {
	err := s.queries.DeleteSubtask(ctx, subtaskID)
	if err != nil {
		return fmt.Errorf("failed to delete subtask: %w", err)
	}

	return nil
}

// GetSubtaskByID returns a single subtask by ID.
func (s *SubtaskService) GetSubtaskByID(ctx context.Context, subtaskID string) (db.CardSubtask, error) {
	subtask, err := s.queries.GetSubtask(ctx, subtaskID)
	if err != nil {
		return db.CardSubtask{}, fmt.Errorf("get subtask failed: %w", err)
	}

	return subtask, nil
}
