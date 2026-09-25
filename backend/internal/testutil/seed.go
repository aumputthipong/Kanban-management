//go:build integration

package testutil

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
)

// Factories insert a row with defaults and return its id; any error fails the test.
type SeedHelper struct {
	t       *testing.T
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewSeed(t *testing.T, pool *pgxpool.Pool) *SeedHelper {
	t.Helper()
	return &SeedHelper{t: t, pool: pool, queries: db.New(pool)}
}

func (s *SeedHelper) User(ctx context.Context) string {
	s.t.Helper()
	email := "u-" + uuid.NewString()[:8] + "@test.local"
	u, err := s.queries.CreateUser(ctx, db.CreateUserParams{
		Email:    email,
		FullName: "Test User",
		Provider: "credentials",
	})
	if err != nil {
		s.t.Fatalf("seed user: %v", err)
	}
	return u.ID
}

func (s *SeedHelper) Board(ctx context.Context, ownerID string) string {
	s.t.Helper()
	b, err := s.queries.CreateBoard(ctx, db.CreateBoardParams{Title: "Test Board"})
	if err != nil {
		s.t.Fatalf("seed board: %v", err)
	}
	if _, err := s.queries.AddBoardMember(ctx, db.AddBoardMemberParams{
		BoardID: b.ID,
		UserID:  ownerID,
		Role:    "owner",
	}); err != nil {
		s.t.Fatalf("seed owner member: %v", err)
	}
	return b.ID
}

func (s *SeedHelper) Column(ctx context.Context, boardID, category string, position float64) string {
	s.t.Helper()
	c, err := s.queries.CreateColumn(ctx, db.CreateColumnParams{
		BoardID:  boardID,
		Title:    category,
		Position: position,
		Category: category,
	})
	if err != nil {
		s.t.Fatalf("seed column: %v", err)
	}
	return c.ID
}

func (s *SeedHelper) PlanningSession(ctx context.Context, boardID, createdBy string) string {
	s.t.Helper()
	createdByPtr := &createdBy
	sess, err := s.queries.CreatePlanningSession(ctx, db.CreatePlanningSessionParams{
		BoardID:   boardID,
		Title:     "Test Session",
		CreatedBy: createdByPtr,
	})
	if err != nil {
		s.t.Fatalf("seed session: %v", err)
	}
	return sess.ID
}

func (s *SeedHelper) PlanningItem(ctx context.Context, sessionID string) string {
	return s.PlanningItemWithType(ctx, sessionID, "REQ")
}

func (s *SeedHelper) PlanningItemWithType(ctx context.Context, sessionID, itemType string) string {
	s.t.Helper()
	it, err := s.queries.CreatePlanningItem(ctx, db.CreatePlanningItemParams{
		SessionID: sessionID,
		Type:      itemType,
		Title:     "Test Item",
		Position:  65536,
	})
	if err != nil {
		s.t.Fatalf("seed item: %v", err)
	}
	return it.ID
}

func (s *SeedHelper) Card(ctx context.Context, columnID string) string {
	s.t.Helper()
	c, err := s.queries.CreateCard(ctx, db.CreateCardParams{
		ColumnID: columnID,
		Title:    "Test Card",
		Position: 65536,
	})
	if err != nil {
		s.t.Fatalf("seed card: %v", err)
	}
	return c.ID
}

func (s *SeedHelper) Subtask(ctx context.Context, cardID string) string {
	s.t.Helper()
	st, err := s.queries.AppendSubtask(ctx, db.AppendSubtaskParams{
		CardID: cardID,
		Title:  "Test Subtask",
	})
	if err != nil {
		s.t.Fatalf("seed subtask: %v", err)
	}
	return st.ID
}

func (s *SeedHelper) Tag(ctx context.Context, boardID, name string) string {
	s.t.Helper()
	tag, err := s.queries.CreateTag(ctx, db.CreateTagParams{
		BoardID: boardID,
		Name:    name,
		Color:   "slate",
	})
	if err != nil {
		s.t.Fatalf("seed tag: %v", err)
	}
	return tag.ID
}
