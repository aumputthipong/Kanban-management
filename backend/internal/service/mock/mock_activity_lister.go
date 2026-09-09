package mock

import (
	"context"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
)

// MockActivityLister implements service.ActivityLister. Each test sets only the
// Fn fields it needs; unset methods panic if called so a missing stub surfaces
// loudly instead of returning a zero value the assertions would silently pass.
type MockActivityLister struct {
	ListFn func(ctx context.Context, boardID string, before *time.Time, limit int32) ([]service.ActivityItem, error)
}

func (m *MockActivityLister) List(ctx context.Context, boardID string, before *time.Time, limit int32) ([]service.ActivityItem, error) {
	return m.ListFn(ctx, boardID, before, limit)
}
