package mock

import (
	"context"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/dto"
)

type MockSubtaskService struct {
	CreateSubtaskFn       func(ctx context.Context, cardID, title string) (db.CardSubtask, error)
	GetSubtasksByCardIDFn func(ctx context.Context, cardID string) ([]db.CardSubtask, error)
	UpdateSubtaskFn       func(ctx context.Context, subtaskID string, req dto.UpdateSubtaskRequest) (db.CardSubtask, error)
	DeleteSubtaskFn       func(ctx context.Context, subtaskID string) error
	GetSubtaskByIDFn      func(ctx context.Context, subtaskID string) (db.CardSubtask, error)
}

func (m *MockSubtaskService) CreateSubtask(ctx context.Context, cardID, title string) (db.CardSubtask, error) {
	return m.CreateSubtaskFn(ctx, cardID, title)
}

func (m *MockSubtaskService) GetSubtasksByCardID(ctx context.Context, cardID string) ([]db.CardSubtask, error) {
	return m.GetSubtasksByCardIDFn(ctx, cardID)
}

func (m *MockSubtaskService) UpdateSubtask(ctx context.Context, subtaskID string, req dto.UpdateSubtaskRequest) (db.CardSubtask, error) {
	return m.UpdateSubtaskFn(ctx, subtaskID, req)
}

func (m *MockSubtaskService) DeleteSubtask(ctx context.Context, subtaskID string) error {
	return m.DeleteSubtaskFn(ctx, subtaskID)
}

func (m *MockSubtaskService) GetSubtaskByID(ctx context.Context, subtaskID string) (db.CardSubtask, error) {
	return m.GetSubtaskByIDFn(ctx, subtaskID)
}
