package mock

import (
	"context"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
)

type MockAuthService struct {
	RegisterFn           func(ctx context.Context, arg service.RegisterParams) (db.User, error)
	LoginFn              func(ctx context.Context, email, password string) (db.User, error)
	UpsertOAuthUserFn    func(ctx context.Context, email, fullName, provider, providerID string) (db.User, error)
	GetUserByIDFn        func(ctx context.Context, userID string) (db.GetUserByIDRow, error)
	IssueRefreshTokenFn  func(ctx context.Context, userID, userAgent, ip string) (string, error)
	RotateRefreshTokenFn func(ctx context.Context, rawToken, userAgent, ip string) (service.RefreshRotationResult, error)
	RevokeRefreshTokenFn func(ctx context.Context, rawToken string) error
}

func (m *MockAuthService) Register(ctx context.Context, arg service.RegisterParams) (db.User, error) {
	return m.RegisterFn(ctx, arg)
}

func (m *MockAuthService) Login(ctx context.Context, email, password string) (db.User, error) {
	return m.LoginFn(ctx, email, password)
}

func (m *MockAuthService) UpsertOAuthUser(ctx context.Context, email, fullName, provider, providerID string) (db.User, error) {
	return m.UpsertOAuthUserFn(ctx, email, fullName, provider, providerID)
}

func (m *MockAuthService) GetUserByID(ctx context.Context, userID string) (db.GetUserByIDRow, error) {
	if m.GetUserByIDFn == nil {
		return db.GetUserByIDRow{}, nil
	}
	return m.GetUserByIDFn(ctx, userID)
}

func (m *MockAuthService) IssueRefreshToken(ctx context.Context, userID, userAgent, ip string) (string, error) {
	if m.IssueRefreshTokenFn == nil {
		return "mock-refresh", nil
	}
	return m.IssueRefreshTokenFn(ctx, userID, userAgent, ip)
}

func (m *MockAuthService) RotateRefreshToken(ctx context.Context, rawToken, userAgent, ip string) (service.RefreshRotationResult, error) {
	if m.RotateRefreshTokenFn == nil {
		return service.RefreshRotationResult{}, nil
	}
	return m.RotateRefreshTokenFn(ctx, rawToken, userAgent, ip)
}

func (m *MockAuthService) RevokeRefreshToken(ctx context.Context, rawToken string) error {
	if m.RevokeRefreshTokenFn == nil {
		return nil
	}
	return m.RevokeRefreshTokenFn(ctx, rawToken)
}
