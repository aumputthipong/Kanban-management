package mock

import (
	"context"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
)

// Unset Fn fields panic, so a missing stub fails loudly.
type MockActivityLister struct {
	ListFn func(ctx context.Context, boardID string, before *time.Time, limit int32) ([]service.ActivityItem, error)
}

func (m *MockActivityLister) List(ctx context.Context, boardID string, before *time.Time, limit int32) ([]service.ActivityItem, error) {
	return m.ListFn(ctx, boardID, before, limit)
}
