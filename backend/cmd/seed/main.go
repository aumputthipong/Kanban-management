// Command seed populates a database with a demo account and a sample board so
// the deployed app opens onto real content instead of an empty screen. It is
// idempotent: if the demo user already exists it exits without changes, so it
// is safe to run on every deploy.
//
// Run:  DB_URL=postgres://... go run ./cmd/seed
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/logging"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// Demo credentials — surfaced in the README so visitors can log in without
// registering. Not secret by design.
const (
	demoEmail      = "demo@turtask.app"
	demoPassword   = "demodemo123"
	memberEmail    = "member@turtask.app"
	memberPassword = "memberdemo123"
)

func main() {
	logging.Init()
	_ = godotenv.Load()

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		slog.Error("DB_URL is required but not set")
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		slog.Error("connect to database failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := seed(ctx, pool); err != nil {
		slog.Error("seed failed", "err", err)
		os.Exit(1)
	}
}

func seed(ctx context.Context, pool *pgxpool.Pool) error {
	queries := db.New(pool)
	auth := service.NewAuthService(pool, queries)
	boards := service.NewBoardService(pool, queries)
	cmds := service.NewBoardCommandService(pool, queries)

	// Register hashes the password (so the demo account can actually log in) and
	// returns ErrEmailTaken if it already exists — our idempotency guard.
	demo, err := auth.Register(ctx, service.RegisterParams{
		Email: demoEmail, FullName: "Demo User", Password: demoPassword,
	})
	if errors.Is(err, service.ErrEmailTaken) {
		slog.Info("demo user already exists — nothing to seed")
		return nil
	}
	if err != nil {
		return fmt.Errorf("create demo user: %w", err)
	}

	member, err := auth.Register(ctx, service.RegisterParams{
		Email: memberEmail, FullName: "Team Member", Password: memberPassword,
	})
	if err != nil {
		return fmt.Errorf("create member user: %w", err)
	}

	desc := "Sample board — try dragging cards, opening a card, and the Planning tab. Open two tabs to see realtime sync."
	color := "#2563EB"
	boardID, err := boards.CreateBoard(ctx, "Product Launch", &desc, &color, nil, demo.ID)
	if err != nil {
		return fmt.Errorf("create board: %w", err)
	}
	if _, err := queries.AddBoardMember(ctx, db.AddBoardMemberParams{
		BoardID: boardID, UserID: member.ID, Role: "member",
	}); err != nil {
		return fmt.Errorf("add member: %w", err)
	}

	// CreateBoard seeds the four default columns; look them up by title.
	cols, err := boards.GetColumnsByBoardID(ctx, boardID)
	if err != nil {
		return fmt.Errorf("load columns: %w", err)
	}
	col := map[string]db.Column{}
	for _, c := range cols {
		col[c.Title] = c
	}

	now := time.Now()
	day := func(n int) *time.Time { t := now.AddDate(0, 0, n); return &t }
	sp := func(s string) *string { return &s }

	// position counters per column so cards keep a stable order.
	pos := map[string]float64{}
	mk := func(colTitle, title, priority string, due *time.Time, assignee *string) (string, error) {
		c := col[colTitle]
		pos[colTitle] += 65536
		row, err := queries.CreateCard(ctx, db.CreateCardParams{
			ColumnID:   c.ID,
			Title:      title,
			Position:   pos[colTitle],
			Priority:   sp(priority),
			DueDate:    due,
			AssigneeID: assignee,
			CreatedBy:  &demo.ID,
		})
		if err != nil {
			return "", fmt.Errorf("create card %q: %w", title, err)
		}
		return row.ID, nil
	}

	cards := []struct {
		colTitle, title, priority string
		due                       *time.Time
		assignee                  *string
	}{
		{"To Do", "Design onboarding flow", "high", day(2), &demo.ID},
		{"To Do", "Write API documentation", "medium", day(5), &member.ID},
		{"To Do", "Set up product analytics", "low", nil, nil},
		{"In Progress", "Build auth service", "high", day(0), &demo.ID},
		{"In Progress", "Realtime WebSocket sync", "high", day(1), &demo.ID},
		{"Review", "Kanban drag-and-drop", "medium", day(-1), &member.ID},
	}
	for _, c := range cards {
		if _, err := mk(c.colTitle, c.title, c.priority, c.due, c.assignee); err != nil {
			return err
		}
	}

	// Done cards: create them, then MoveCard into the Done column so the service
	// stamps is_done + completed_at the same way a real move does.
	done := col["Done"]
	for _, title := range []string{"Project scaffolding", "CI pipeline"} {
		id, err := mk("To Do", title, "medium", nil, &demo.ID)
		if err != nil {
			return err
		}
		if _, err := cmds.MoveCard(ctx, id, done.ID, pos["Done"]+65536); err != nil {
			return fmt.Errorf("mark %q done: %w", title, err)
		}
		pos["Done"] += 65536
	}

	// A small Planning session so the planning → board pipeline has content.
	sess, err := queries.CreatePlanningSession(ctx, db.CreatePlanningSessionParams{
		BoardID: boardID, Title: "Sprint 1 planning", CreatedBy: &demo.ID,
	})
	if err != nil {
		return fmt.Errorf("create planning session: %w", err)
	}
	items := []struct{ typ, title string }{
		{"REQ", "Users can drag cards between columns"},
		{"DEC", "Use optimistic UI with WebSocket reconcile"},
		{"Q", "Do we need card archiving?"},
	}
	for i, it := range items {
		if _, err := queries.CreatePlanningItem(ctx, db.CreatePlanningItemParams{
			SessionID: sess.ID, Type: it.typ, Title: it.title, Position: float64(i+1) * 65536,
		}); err != nil {
			return fmt.Errorf("create planning item: %w", err)
		}
	}

	slog.Info("seed complete",
		"board_id", boardID,
		"demo_login", demoEmail,
		"demo_password", demoPassword,
	)
	return nil
}
