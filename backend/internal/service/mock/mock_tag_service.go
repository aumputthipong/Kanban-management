package mock

import (
	"context"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
)

// Unset Fn fields panic, so a missing stub fails loudly.
type MockTagService struct {
	GetTagsByBoardFn func(ctx context.Context, boardID string) ([]db.Tag, error)
	CreateTagFn      func(ctx context.Context, boardID, name, color string) (db.Tag, error)
	DeleteTagFn      func(ctx context.Context, boardID, tagID string) error
}

func (m *MockTagService) GetTagsByBoard(ctx context.Context, boardID string) ([]db.Tag, error) {
	return m.GetTagsByBoardFn(ctx, boardID)
}

func (m *MockTagService) CreateTag(ctx context.Context, boardID, name, color string) (db.Tag, error) {
	return m.CreateTagFn(ctx, boardID, name, color)
}

func (m *MockTagService) DeleteTag(ctx context.Context, boardID, tagID string) error {
	return m.DeleteTagFn(ctx, boardID, tagID)
}
