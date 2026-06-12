//go:build integration

package testutil

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
)

// SeedHelper bundles the minimum factory functions integration tests need.
// Each helper inserts a row with sensible defaults and returns the resulting
// ID; pass non-zero fields to override. Callers fail the test on any error
// — there's no recoverable seed failure in a fresh template-cloned DB.
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

// Board creates a board and adds the given user as owner (mirrors the real
// CreateBoard flow in board_service which inserts the row + owner member
// in one transaction).
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

// Column creates a column with the given category ("TODO", "IN_PROGRESS",
// "DONE"). Position auto-increments per call so multiple columns in the
// same board don't collide.
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

// PlanningItem creates a live item with default type REQ. Use ItemWithType
// for a specific type.
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
