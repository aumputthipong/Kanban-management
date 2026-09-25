package mock

import (
	"context"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
)

type MockUserSettingsService struct {
	GetFn    func(ctx context.Context, userID string) (service.UserSettingsData, error)
	UpdateFn func(ctx context.Context, userID string, p service.UpdateUserSettingsParams) (service.UserSettingsData, error)
}

func (m *MockUserSettingsService) Get(ctx context.Context, userID string) (service.UserSettingsData, error) {
	return m.GetFn(ctx, userID)
}

func (m *MockUserSettingsService) Update(ctx context.Context, userID string, p service.UpdateUserSettingsParams) (service.UserSettingsData, error) {
	return m.UpdateFn(ctx, userID, p)
}
