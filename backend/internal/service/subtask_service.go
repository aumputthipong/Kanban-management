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

func (s *SubtaskService) CreateSubtask(ctx context.Context, arg db.CreateSubtaskParams) (db.CardSubtask, error) {

	subtask, err := s.queries.CreateSubtask(ctx, arg)
	if err != nil {
		return db.CardSubtask{}, fmt.Errorf("db error: %w", err)
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
func (s *SubtaskService) UpdateSubtask(ctx context.Context, subtaskID string, req dto.UpdateSubtaskRequest) (db.CardSubtask, error) {

	// Read-modify-write: start from the existing row, override provided fields.
	existing, err := s.queries.GetSubtask(ctx, subtaskID)
	if err != nil {
		return db.CardSubtask{}, fmt.Errorf("subtask not found: %w", err)
	}

	title := existing.Title
	if req.Title != nil {
		title = *req.Title
	}

	isDone := existing.IsDone
	if req.IsDone != nil {
		isDone = *req.IsDone
	}

	position := existing.Position
	if req.Position != nil {
		position = *req.Position
	}

	params := db.UpdateSubtaskParams{
		ID:       subtaskID,
		Title:    title,
		IsDone:   isDone,
		Position: position,
	}

	updatedSubtask, err := s.queries.UpdateSubtask(ctx, params)
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
