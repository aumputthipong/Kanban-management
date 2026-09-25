// Command seed creates the demo account and its sample board. Idempotent.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/logging"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
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

	// ErrEmailTaken means the demo user already exists.
	demo, err := auth.Register(ctx, service.RegisterParams{
		Email: service.SeedDemoEmail, FullName: "Demo User", Password: service.SeedDemoPassword,
	})
	if errors.Is(err, service.ErrEmailTaken) {
		slog.Info("demo user already exists — nothing to seed")
		return nil
	}
	if err != nil {
		return fmt.Errorf("create demo user: %w", err)
	}

	member, err := auth.Register(ctx, service.RegisterParams{
		Email: service.SeedMemberEmail, FullName: "Team Member", Password: service.SeedMemberPassword,
	})
	if err != nil {
		return fmt.Errorf("create member user: %w", err)
	}

	boardID, err := service.SeedSampleBoard(ctx, service.SampleBoardDeps{
		Queries: queries, Boards: boards, Commands: cmds,
	}, demo.ID, &member.ID)
	if err != nil {
		return err
	}

	slog.Info("seed complete",
		"board_id", boardID,
		"demo_login", service.SeedDemoEmail,
		"demo_password", service.SeedDemoPassword,
	)
	return nil
}
