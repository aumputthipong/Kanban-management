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

const DemoValidity = 24 * time.Hour

// Caps one sweep so a backlog can't hold a transaction open for minutes.
const purgeBatchSize = 200

// Not a real domain — nothing should ever deliver mail to a sandbox.
const demoEmailDomain = "sandbox.turtask.invalid"

// Per-visitor sandboxes rather than one shared login — docs/adr/0010.
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

type DemoSandbox struct {
	UserID   string
	Email    string
	FullName string
	BoardID  string
	ExpireAt time.Time
}

// Not one transaction: SeedSampleBoard's callees open their own; the purge collects half-seeded sandboxes.
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

	// Companion member so the board isn't a one-person view. Missing seed data isn't fatal.
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

// Order matters: boards first (cascade), then tables whose user FK has no ON DELETE.
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

// A failed sweep only logs — the next tick retries.
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

// 96 bits, so concurrent visitors never collide on users.email.
func randomSuffix() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate demo suffix: %w", err)
	}
	return hex.EncodeToString(b), nil
}
