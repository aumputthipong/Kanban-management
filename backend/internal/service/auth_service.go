// internal/service/auth_service.go
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/util"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailTaken   = errors.New("email already in use")
	ErrInvalidCreds = errors.New("invalid email or password")
	ErrOAuthOnly    = errors.New("account registered via OAuth, please use Google or GitHub")
)

type AuthService struct {
	queries *db.Queries
}

func NewAuthService(queries *db.Queries) *AuthService {
	return &AuthService{queries: queries}
}

type RegisterParams struct {
	Email    string
	FullName string
	Password string
}

func (s *AuthService) Register(ctx context.Context, arg RegisterParams) (db.User, error) {
	_, err := s.queries.GetUserByEmail(ctx, arg.Email)
	if err == nil {
		// A nil error means the query found a user with this email already.
		return db.User{}, ErrEmailTaken
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(arg.Password), bcrypt.DefaultCost)
	if err != nil {
		return db.User{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.queries.CreateUser(ctx, db.CreateUserParams{
		Email:        arg.Email,
		FullName:     arg.FullName,
		PasswordHash: util.StringToPtr(string(hash)),
		Provider:     "credentials",
		ProviderID:   nil,
	})
	if err != nil {
		return db.User{}, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (db.User, error) {
	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return db.User{}, ErrInvalidCreds
	}

	if user.Provider != "credentials" {
		return db.User{}, ErrOAuthOnly
	}

	if user.PasswordHash == nil {
		return db.User{}, ErrInvalidCreds
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); err != nil {
		return db.User{}, ErrInvalidCreds
	}

	return user, nil
}

func (s *AuthService) GetUserByID(ctx context.Context, userID string) (db.GetUserByIDRow, error) {
	return s.queries.GetUserByID(ctx, userID)
}

func (s *AuthService) UpsertOAuthUser(ctx context.Context, email, fullName, provider, providerID string) (db.User, error) {
	user, err := s.queries.UpsertOAuthUser(ctx, db.UpsertOAuthUserParams{
		Email:      email,
		FullName:   fullName,
		Provider:   provider,
		ProviderID: util.StringToPtr(providerID),
	})
	if err != nil {
		return db.User{}, fmt.Errorf("upsert oauth user: %w", err)
	}
	return user, nil
}
