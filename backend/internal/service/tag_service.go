package service

import (
	"context"
	"errors"
	"strings"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxTagNameLen = 50

// Sentinel validation errors so handlers can map them to a 422 with a safe,
// user-facing message — instead of leaking the raw DB error via err.Error().
var (
	ErrTagNameEmpty   = errors.New("tag name cannot be empty")
	ErrTagNameTooLong = errors.New("tag name too long (max 50 chars)")
)

type TagService struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewTagService(pool *pgxpool.Pool, queries *db.Queries) *TagService {
	return &TagService{pool: pool, queries: queries}
}

func (s *TagService) GetTagsByBoard(ctx context.Context, boardID string) ([]db.Tag, error) {
	return s.queries.GetTagsByBoardID(ctx, boardID)
}

func (s *TagService) CreateTag(ctx context.Context, boardID, name, color string) (db.Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return db.Tag{}, ErrTagNameEmpty
	}
	if len(name) > maxTagNameLen {
		return db.Tag{}, ErrTagNameTooLong
	}
	return s.queries.CreateTag(ctx, db.CreateTagParams{
		BoardID: boardID,
		Name:    name,
		Color:   color,
	})
}

func (s *TagService) DeleteTag(ctx context.Context, boardID, tagID string) error {
	return s.queries.DeleteTag(ctx, db.DeleteTagParams{ID: tagID, BoardID: boardID})
}
