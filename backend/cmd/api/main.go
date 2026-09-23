package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/handler"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/logging"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/middleware"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/migrate"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/observability"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/token"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// Regenerate the OpenAPI spec into backend/docs after editing handler annotations:
// `make swag`, or `go generate ./cmd/api` from backend/. Paths are relative to this
// file's directory, which is where go generate runs the command.
//go:generate swag init -d ../../ -g cmd/api/main.go -o ../../docs --parseDependency --parseInternal

// version is set at build time via -ldflags "-X main.version=..."; defaults to "dev" locally.
var version = "dev"

// @title           Turtask API
// @version         1.0
// @description     Multi-board Kanban + task management with realtime sync.
// @description     Auth uses an HttpOnly `auth_token` cookie issued by /api/auth/login or /api/auth/oauth.
//
// @contact.name    Turtask
//
// @host            localhost:8080
// @BasePath        /
//
// @securityDefinitions.apikey  CookieAuth
// @in                          cookie
// @name                        auth_token

const (
	shutdownTimeout = 30 * time.Second
	dbPoolMaxConns  = 25
	dbPoolMinConns  = 5
	dbPoolMaxIdle   = 5 * time.Minute
	// How often expired demo sandboxes are swept. Sandboxes live 24h, so the
	// exact cadence only decides how long dead rows linger.
	demoPurgeInterval = time.Hour
)

type config struct {
	DBUrl              string
	Port               string
	FrontendURL        string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirect     string
	MigrationsPath     string
	SkipMigrations     bool
	Production         bool
	CrossSite          bool
	TrustedProxies     int
}

// trustedProxyCount reads TRUSTED_PROXY_COUNT: how many proxies sit in front of this
// server. Defaults to 1, matching both deploy shapes in docs/DEPLOY.md (Render, nginx);
// set 0 when the binary is exposed directly, or the X-Forwarded-For it trusts is the
// client's own and rate-limit keys become forgeable.
func trustedProxyCount() int {
	raw := os.Getenv("TRUSTED_PROXY_COUNT")
	if raw == "" {
		return 1
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		slog.Error("TRUSTED_PROXY_COUNT must be a non-negative integer", "value", raw)
		os.Exit(1)
	}
	return n
}

func loadConfig() config {
	cfg := config{
		DBUrl:              os.Getenv("DB_URL"),
		Port:               os.Getenv("PORT"),
		FrontendURL:        os.Getenv("FRONTEND_URL"),
		Production:         os.Getenv("ENV") == "production",
		CrossSite:          os.Getenv("COOKIE_CROSS_SITE") == "true",
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirect:     os.Getenv("GOOGLE_REDIRECT_URL"),
		MigrationsPath:     os.Getenv("MIGRATIONS_PATH"),
		SkipMigrations:     os.Getenv("SKIP_MIGRATIONS") == "true",
		TrustedProxies:     trustedProxyCount(),
	}
	if cfg.DBUrl == "" {
		slog.Error("DB_URL is required but not set")
		os.Exit(1)
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		slog.Error("JWT_SECRET is required but not set")
		os.Exit(1)
	}
	if len(jwtSecret) < token.MinSecretBytes {
		slog.Error("JWT_SECRET is too short",
			"got_bytes", len(jwtSecret),
			"min_bytes", token.MinSecretBytes,
			"hint", "generate with: openssl rand -base64 32",
		)
		os.Exit(1)
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.FrontendURL == "" {
		cfg.FrontendURL = "http://localhost:3000"
	}
	if cfg.MigrationsPath == "" {
		cfg.MigrationsPath = "database/migrations"
	}
	return cfg
}

func initDB(ctx context.Context, dbURL string) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}
	poolCfg.MaxConns = dbPoolMaxConns
	poolCfg.MinConns = dbPoolMinConns
	poolCfg.MaxConnIdleTime = dbPoolMaxIdle

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("could not ping database: %w", err)
	}
	return pool, nil
}

func run(ctx context.Context, cfg config) error {
	if !cfg.SkipMigrations {
		// schema.sql (the sqlc source of truth) bootstraps a fresh DB; migrations
		// evolve an existing one. See migrate.Bootstrap.
		schemaPath := filepath.Join(filepath.Dir(cfg.MigrationsPath), "schema.sql")
		slog.Info("bootstrapping database", "schema", schemaPath, "migrations", cfg.MigrationsPath)
		if err := migrate.Bootstrap(ctx, cfg.DBUrl, schemaPath, cfg.MigrationsPath); err != nil {
			return fmt.Errorf("database bootstrap failed: %w", err)
		}
	} else {
		slog.Info("skipping migrations", "reason", "SKIP_MIGRATIONS=true")
	}

	pool, err := initDB(ctx, cfg.DBUrl)
	if err != nil {
		return fmt.Errorf("database init failed: %w", err)
	}
	defer pool.Close()
	observability.RegisterDBPool(pool)
	slog.Info("connected to postgres",
		"max_conns", dbPoolMaxConns,
		"min_conns", dbPoolMinConns,
		"max_idle", dbPoolMaxIdle.String(),
	)

	queries := db.New(pool)

	activityService := service.NewActivityService(queries)
	boardCmdService := service.NewBoardCommandService(pool, queries)

	hub := websocket.NewHub(cfg.FrontendURL)
	// Handlers type-assert eviction off the Broadcaster; fail the build, not silently, if it drifts.
	var _ handler.RoomEvictor = hub
	go hub.Run()

	boardService := service.NewBoardService(pool, queries)
	authService := service.NewAuthService(pool, queries)
	subtaskService := service.NewSubtaskService(pool)
	tagService := service.NewTagService(pool, queries)
	planningService := service.NewPlanningService(pool, queries)
	settingsService := service.NewUserSettingsService(queries)
	inviteService := service.NewInviteService(pool, queries)
	demoService := service.NewDemoService(pool, queries)

	subtaskHandler := handler.NewSubtaskHandler(subtaskService, boardService, activityService, hub)
	boardHandler := handler.NewBoardHandler(boardService, settingsService, activityService, hub)
	boardCmdHandler := handler.NewBoardCommandHandler(boardCmdService, boardService, activityService, hub)
	tagHandler := handler.NewTagHandler(tagService, hub)
	activityHandler := handler.NewActivityHandler(activityService)
	planningHandler := handler.NewPlanningHandler(planningService, boardService, activityService)
	authHandler := handler.NewAuthHandler(authService, demoService, cfg.Production, cfg.CrossSite)
	settingsHandler := handler.NewUserSettingsHandler(settingsService)
	inviteHandler := handler.NewInviteHandler(inviteService, boardService, activityService, hub)
	oauthHandler := handler.NewOAuthHandler(
		cfg.GoogleClientID,
		cfg.GoogleClientSecret,
		cfg.GoogleRedirect,
		cfg.FrontendURL,
		authService,
		cfg.Production,
		cfg.CrossSite,
	)

	startedAt := time.Now()
	router := setupRoutes(routerDeps{
		boardService:    boardService,
		boardHandler:    boardHandler,
		boardCmdHandler: boardCmdHandler,
		authHandler:     authHandler,
		oauthHandler:    oauthHandler,
		subtaskHandler:  subtaskHandler,
		tagHandler:      tagHandler,
		activityHandler: activityHandler,
		planningHandler: planningHandler,
		settingsHandler: settingsHandler,
		inviteHandler:   inviteHandler,
		hub:             hub,
		pool:            pool,
		version:         version,
		production:      cfg.Production,
		trustedProxies:  cfg.TrustedProxies,
		startedAt:       startedAt,
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           middleware.CORS(cfg.FrontendURL, router),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Reclaim expired demo sandboxes. Runs in-process rather than as a cron job
	// because a sweep is a handful of deletes; ctx cancellation stops it.
	go demoService.StartPurgeLoop(ctx, demoPurgeInterval)

	go func() {
		slog.Info("server listening", "port", cfg.Port, "version", version, "trusted_proxies", cfg.TrustedProxies)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("listen failed", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutdown signal received — draining", "timeout", shutdownTimeout.String())

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}
	// HTTP listener is drained; now close any active WS connections so their
	// pumps can exit cleanly before we return and the pool is closed.
	hub.Shutdown()
	// Drain any queued audit writes before pool.Close() pulls the rug out.
	activityService.Stop()
	slog.Info("server stopped cleanly")
	return nil
}

func main() {
	logging.Init()

	if err := godotenv.Load(); err != nil {
		slog.Debug("no .env file found, using system environment", "err", err)
	}

	cfg := loadConfig()

	observability.InitSentry(version)
	defer observability.FlushSentry(2 * time.Second)

	if err := run(context.Background(), cfg); err != nil {
		slog.Error("run failed", "err", err)
		os.Exit(1)
	}
}
