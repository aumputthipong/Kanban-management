package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/jackc/pgx/v5"
)

type UserSettingsData struct {
	UserID         string
	DefaultLanding string
	ShowAllCards   bool
	Timezone       string
}

type UserSettingsService struct {
	queries *db.Queries
}

func NewUserSettingsService(queries *db.Queries) *UserSettingsService {
	return &UserSettingsService{queries: queries}
}

// Creates a default row on first read, via Upsert so column defaults apply.
func (s *UserSettingsService) Get(ctx context.Context, userID string) (UserSettingsData, error) {
	row, err := s.queries.GetUserSettings(ctx, userID)
	if err == nil {
		return toUserSettingsData(row), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return UserSettingsData{}, fmt.Errorf("get user settings: %w", err)
	}
	created, err := s.queries.UpsertUserSettings(ctx, db.UpsertUserSettingsParams{UserID: userID})
	if err != nil {
		return UserSettingsData{}, fmt.Errorf("init user settings: %w", err)
	}
	return toUserSettingsData(created), nil
}

type UpdateUserSettingsParams struct {
	DefaultLanding *string
	ShowAllCards   *bool
	Timezone       *string
}

var validLandings = map[string]struct{}{
	"today":      {},
	"my_work":    {},
	"all_boards": {},
}

var ErrInvalidLanding = errors.New("invalid default_landing")

func (s *UserSettingsService) Update(ctx context.Context, userID string, p UpdateUserSettingsParams) (UserSettingsData, error) {
	if p.DefaultLanding != nil {
		if _, ok := validLandings[*p.DefaultLanding]; !ok {
			return UserSettingsData{}, ErrInvalidLanding
		}
	}
	row, err := s.queries.UpsertUserSettings(ctx, db.UpsertUserSettingsParams{
		UserID:         userID,
		DefaultLanding: p.DefaultLanding,
		ShowAllCards:   p.ShowAllCards,
		Timezone:       p.Timezone,
	})
	if err != nil {
		return UserSettingsData{}, fmt.Errorf("upsert user settings: %w", err)
	}
	return toUserSettingsData(row), nil
}

func toUserSettingsData(r db.UserSetting) UserSettingsData {
	return UserSettingsData{
		UserID:         r.UserID,
		DefaultLanding: r.DefaultLanding,
		ShowAllCards:   r.ShowAllCards,
		Timezone:       r.Timezone,
	}
}
