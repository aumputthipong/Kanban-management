package mock

import (
	"context"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
)

// MockDemoService implements service.DemoServicer
type MockDemoService struct {
	CreateSandboxFn func(ctx context.Context, companionEmail string) (service.DemoSandbox, error)
	PurgeExpiredFn  func(ctx context.Context) (int64, error)
}

func (m *MockDemoService) CreateSandbox(ctx context.Context, companionEmail string) (service.DemoSandbox, error) {
	return m.CreateSandboxFn(ctx, companionEmail)
}

func (m *MockDemoService) PurgeExpired(ctx context.Context) (int64, error) {
	if m.PurgeExpiredFn == nil {
		return 0, nil
	}
	return m.PurgeExpiredFn(ctx)
}
