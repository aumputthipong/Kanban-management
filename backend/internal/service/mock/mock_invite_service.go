package mock

import (
	"context"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
)

// MockInviteService is the function-field mock for service.InviteServicer,
// matching the convention used by the other handler-seam mocks.
type MockInviteService struct {
	CreateInviteFn    func(ctx context.Context, boardID, creatorID string) (service.InviteLink, error)
	GetActiveInviteFn func(ctx context.Context, boardID string) (service.InviteLink, bool, error)
	RevokeInvitesFn   func(ctx context.Context, boardID string) error
	AcceptInviteFn    func(ctx context.Context, token, userID string) (string, error)
}

func (m *MockInviteService) CreateInvite(ctx context.Context, boardID, creatorID string) (service.InviteLink, error) {
	return m.CreateInviteFn(ctx, boardID, creatorID)
}

func (m *MockInviteService) GetActiveInvite(ctx context.Context, boardID string) (service.InviteLink, bool, error) {
	return m.GetActiveInviteFn(ctx, boardID)
}

func (m *MockInviteService) RevokeInvites(ctx context.Context, boardID string) error {
	return m.RevokeInvitesFn(ctx, boardID)
}

func (m *MockInviteService) AcceptInvite(ctx context.Context, token, userID string) (string, error) {
	return m.AcceptInviteFn(ctx, token, userID)
}
