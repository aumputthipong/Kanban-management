// internal/service/demo_service.go
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DemoValidity is how long a sandbox survives before the purge sweep reclaims it.
const DemoValidity = 24 * time.Hour

// purgeBatchSize caps one sweep so a backlog cannot hold a transaction open for
// minutes; the next tick picks up the rest.
const purgeBatchSize = 200

// demoEmailDomain is deliberately not a real domain — nothing should ever try to
// deliver mail to a sandbox account.
const demoEmailDomain = "sandbox.turtask.invalid"

// DemoService mints throwaway sandbox accounts for the "Try demo" button and
// reclaims them once they expire. Per-visitor rather than one shared login —
// see docs/adr/0010.
type DemoService struct {
	pool     *pgxpool.Pool
	queries  *db.Queries
	boards   *BoardService
	commands *BoardCommandService
}

func NewDemoService(pool *pgxpool.Pool, queries *db.Queries) *DemoService {
	return &DemoService{
		pool:     pool,
		queries:  queries,
		boards:   NewBoardService(pool, queries),
		commands: NewBoardCommandService(pool, queries),
	}
}

// DemoSandbox is a freshly minted demo identity plus the board to land them on.
type DemoSandbox struct {
	UserID   string
	Email    string
	FullName string
	BoardID  string
	ExpireAt time.Time
}

// CreateSandbox provisions a demo user and their own copy of the sample board.
// Deliberately not one transaction: SeedSampleBoard's callees open their own, and
// a half-seeded sandbox is throwaway data the purge collects anyway.
func (s *DemoService) CreateSandbox(ctx context.Context, companionEmail string) (DemoSandbox, error) {
	suffix, err := randomSuffix()
	if err != nil {
		return DemoSandbox{}, err
	}

	expiresAt := time.Now().Add(DemoValidity)
	user, err := s.queries.CreateDemoUser(ctx, db.CreateDemoUserParams{
		Email:         fmt.Sprintf("demo-%s@%s", suffix, demoEmailDomain),
		FullName:      "Demo Visitor",
		DemoExpiresAt: &expiresAt,
	})
	if err != nil {
		return DemoSandbox{}, fmt.Errorf("create demo user: %w", err)
	}

	// The shared seed account joins as the second member so avatars, the Members
	// tab and the ownership view have someone other than the visitor in them. Its
	// absence (an unseeded database) is not worth failing the demo over.
	var companionID *string
	if companionEmail != "" {
		if companion, lookupErr := s.queries.GetUserByEmail(ctx, companionEmail); lookupErr == nil {
			companionID = &companion.ID
		}
	}

	boardID, err := SeedSampleBoard(ctx, SampleBoardDeps{
		Queries:  s.queries,
		Boards:   s.boards,
		Commands: s.commands,
	}, user.ID, companionID)
	if err != nil {
		return DemoSandbox{}, fmt.Errorf("seed sandbox board: %w", err)
	}

	return DemoSandbox{
		UserID:   user.ID,
		Email:    user.Email,
		FullName: user.FullName,
		BoardID:  boardID,
		ExpireAt: expiresAt,
	}, nil
}

// PurgeExpired deletes expired sandboxes and returns how many users went.
//
// Order matters: boards first (one cascade takes columns, cards, members, tags,
// planning and that board's activities), then the tables whose user reference has
// no ON DELETE clause and would otherwise block the final delete.
func (s *DemoService) PurgeExpired(ctx context.Context) (int64, error) {
	ids, err := s.queries.ListExpiredDemoUserIDs(ctx, purgeBatchSize)
	if err != nil {
		return 0, fmt.Errorf("list expired demo users: %w", err)
	}
	if len(ids) == 0 {
		return 0, nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)
	qtx := s.queries.WithTx(tx)

	if err := qtx.DeleteBoardsOwnedBy(ctx, ids); err != nil {
		return 0, fmt.Errorf("delete demo boards: %w", err)
	}
	if err := qtx.DeleteActivitiesByActors(ctx, ids); err != nil {
		return 0, fmt.Errorf("delete demo activities: %w", err)
	}
	if err := qtx.DeletePlanningCommentsByAuthors(ctx, ids); err != nil {
		return 0, fmt.Errorf("delete demo planning comments: %w", err)
	}
	if err := qtx.DeleteTimeLogsByUsers(ctx, ids); err != nil {
		return 0, fmt.Errorf("delete demo time logs: %w", err)
	}
	removed, err := qtx.DeleteUsersByIDs(ctx, ids)
	if err != nil {
		return 0, fmt.Errorf("delete demo users: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit purge: %w", err)
	}
	return removed, nil
}

// StartPurgeLoop sweeps expired sandboxes on a ticker until ctx is cancelled.
// A failed sweep only logs — the next tick retries, and a purge backlog degrades
// disk use, not correctness.
func (s *DemoService) StartPurgeLoop(ctx context.Context, every time.Duration) {
	sweep := func() {
		removed, err := s.PurgeExpired(ctx)
		if err != nil {
			slog.Error("demo purge failed", "err", err)
			return
		}
		if removed > 0 {
			slog.Info("demo sandboxes purged", "users", removed)
		}
	}

	sweep()
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweep()
		}
	}
}

// randomSuffix returns 96 bits of hex — enough that two concurrent visitors
// never collide on the users.email unique index.
func randomSuffix() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate demo suffix: %w", err)
	}
	return hex.EncodeToString(b), nil
}
