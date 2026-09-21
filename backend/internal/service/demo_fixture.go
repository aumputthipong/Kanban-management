// internal/service/demo_fixture.go
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
)

// SampleBoardDescription is surfaced in the UI as the demo's "what to try" line.
const SampleBoardDescription = "Sample board — try dragging cards, opening a card, and the Planning tab. Open two tabs to see realtime sync."

// Credentials for the two accounts `cmd/seed` creates. Published in the README so
// a visitor can sign in without registering — not secret by design. They live
// here rather than in cmd/seed because DemoService needs SeedMemberEmail to add
// the companion member to each sandbox.
const (
	SeedDemoEmail      = "demo@turtask.app"
	SeedDemoPassword   = "demodemo123"
	SeedMemberEmail    = "member@turtask.app"
	SeedMemberPassword = "memberdemo123"
)

const sampleBoardColor = "#2563EB"

// SampleBoardDeps is the slice of the service layer the fixture needs. Taking it
// as a struct keeps the fixture usable from both cmd/seed and DemoService without
// either owning the other's wiring.
type SampleBoardDeps struct {
	Queries  *db.Queries
	Boards   *BoardService
	Commands *BoardCommandService
}

// SeedSampleBoard builds the demo "Product Launch" board: four columns, eight
// cards, two of them Done, and a planning session. memberID, when non-nil, joins
// as a second member. Shared by cmd/seed and DemoService so the two never drift.
func SeedSampleBoard(ctx context.Context, d SampleBoardDeps, ownerID string, memberID *string) (string, error) {
	desc := SampleBoardDescription
	color := sampleBoardColor
	boardID, err := d.Boards.CreateBoard(ctx, "Product Launch", &desc, &color, nil, ownerID)
	if err != nil {
		return "", fmt.Errorf("create board: %w", err)
	}

	if memberID != nil {
		if _, err := d.Queries.AddBoardMember(ctx, db.AddBoardMemberParams{
			BoardID: boardID, UserID: *memberID, Role: "member",
		}); err != nil {
			return "", fmt.Errorf("add member: %w", err)
		}
	}

	// CreateBoard seeds the four default columns; look them up by title.
	cols, err := d.Boards.GetColumnsByBoardID(ctx, boardID)
	if err != nil {
		return "", fmt.Errorf("load columns: %w", err)
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
		row, err := d.Queries.CreateCard(ctx, db.CreateCardParams{
			ColumnID:   c.ID,
			Title:      title,
			Position:   pos[colTitle],
			Priority:   sp(priority),
			DueDate:    due,
			AssigneeID: assignee,
			CreatedBy:  &ownerID,
		})
		if err != nil {
			return "", fmt.Errorf("create card %q: %w", title, err)
		}
		return row.ID, nil
	}

	// A nil memberID leaves the member-assigned cards unassigned rather than
	// piling every card on the owner — an all-mine board reads wrong.
	cards := []struct {
		colTitle, title, priority string
		due                       *time.Time
		assignee                  *string
	}{
		{"To Do", "Design onboarding flow", "high", day(2), &ownerID},
		{"To Do", "Write API documentation", "medium", day(5), memberID},
		{"To Do", "Set up product analytics", "low", nil, nil},
		{"In Progress", "Build auth service", "high", day(0), &ownerID},
		{"In Progress", "Realtime WebSocket sync", "high", day(1), &ownerID},
		{"Review", "Kanban drag-and-drop", "medium", day(-1), memberID},
	}
	for _, c := range cards {
		if _, err := mk(c.colTitle, c.title, c.priority, c.due, c.assignee); err != nil {
			return "", err
		}
	}

	// Done cards: create them, then MoveCard into the Done column so the service
	// stamps is_done + completed_at the same way a real move does.
	done := col["Done"]
	for _, title := range []string{"Project scaffolding", "CI pipeline"} {
		id, err := mk("To Do", title, "medium", nil, &ownerID)
		if err != nil {
			return "", err
		}
		if _, err := d.Commands.MoveCard(ctx, id, done.ID, pos["Done"]+65536); err != nil {
			return "", fmt.Errorf("mark %q done: %w", title, err)
		}
		pos["Done"] += 65536
	}

	sess, err := d.Queries.CreatePlanningSession(ctx, db.CreatePlanningSessionParams{
		BoardID: boardID, Title: "Sprint 1 planning", CreatedBy: &ownerID,
	})
	if err != nil {
		return "", fmt.Errorf("create planning session: %w", err)
	}
	items := []struct{ typ, title string }{
		{"REQ", "Users can drag cards between columns"},
		{"DEC", "Use optimistic UI with WebSocket reconcile"},
		{"Q", "Do we need card archiving?"},
	}
	for i, it := range items {
		if _, err := d.Queries.CreatePlanningItem(ctx, db.CreatePlanningItemParams{
			SessionID: sess.ID, Type: it.typ, Title: it.title, Position: float64(i+1) * 65536,
		}); err != nil {
			return "", fmt.Errorf("create planning item: %w", err)
		}
	}

	return boardID, nil
}
