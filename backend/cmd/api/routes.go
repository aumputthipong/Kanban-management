package main

import (
	"context"
	"net/http"
	"sync"
	"time"

	_ "github.com/aumputthipong/mini-erp-kanban/backend/docs"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/core"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/handler"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/middleware"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/observability"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/websocket"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

type routerDeps struct {
	boardService    service.BoardServicer
	boardHandler    *handler.BoardHandler
	boardCmdHandler *handler.BoardCommandHandler
	authHandler     *handler.AuthHandler
	oauthHandler    *handler.OAuthHandler
	subtaskHandler  *handler.SubtaskHandler
	tagHandler      *handler.TagHandler
	activityHandler *handler.ActivityHandler
	planningHandler *handler.PlanningHandler
	settingsHandler *handler.UserSettingsHandler
	inviteHandler   *handler.InviteHandler
	hub             *websocket.Hub
	pool            *pgxpool.Pool
	version         string
	production      bool
	trustedProxies  int
	startedAt       time.Time
}

func setupRoutes(d routerDeps) http.Handler {
	r := chi.NewRouter()

	// SentryRecoverer before Recoverer; RequestLogger redacts OAuth code/state and the WS ticket.
	r.Use(chiMiddleware.RequestID)
	// Before RequestLogger and every rate limiter — they key off the client IP.
	r.Use(middleware.ClientIPResolver(d.trustedProxies))
	r.Use(middleware.RequestLogger)
	r.Use(observability.SentryRecoverer())
	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.SecurityHeaders(d.production))
	r.Use(observability.HTTPMetrics)
	r.Use(chiMiddleware.Compress(5, "application/json", "text/html", "text/css", "text/plain"))

	r.Get("/health", healthHandler(d.pool, d.version, d.startedAt))
	r.Get("/healthz", healthHandler(d.pool, d.version, d.startedAt))

	// Unauthenticated — access is restricted at the network layer.
	r.Handle("/metrics", observability.MetricsHandler())

	// Disabled in production: a public API map is a roadmap for attackers.
	if !d.production {
		r.Get("/docs/*", httpSwagger.Handler(httpSwagger.URL("/docs/doc.json")))
	}

	r.Route("/api/auth", func(r chi.Router) {
		r.Use(middleware.AuthRateLimit())

		r.Post("/register", httputil.MakeHandler(d.authHandler.Register))
		r.Post("/login", httputil.MakeHandler(d.authHandler.Login))
		r.Post("/oauth", httputil.MakeHandler(d.authHandler.OAuthCallback))
		// Own limiter: each call seeds a whole board.
		r.With(middleware.DemoRateLimit()).Post("/demo", httputil.MakeHandler(d.authHandler.Demo))
		r.Post("/logout", httputil.MakeHandler(d.authHandler.Logout))
		// The refresh cookie is the credential.
		r.Post("/refresh", httputil.MakeHandler(d.authHandler.Refresh))

		r.Get("/google", httputil.MakeHandler(d.oauthHandler.RedirectToGoogle))
		r.Get("/google/callback", httputil.MakeHandler(d.oauthHandler.HandleGoogleCallback))
	})

	requireBoardMember := middleware.RequireBoardMember(d.boardService)

	// Ticket auth, not cookie (docs/adr/0005). Membership is still gated.
	r.Group(func(r chi.Router) {
		r.Use(middleware.GeneralRateLimit())
		r.Use(middleware.RequireWSTicket)

		r.With(requireBoardMember).Get("/ws/{boardID}", func(w http.ResponseWriter, r *http.Request) {
			boardID := chi.URLParam(r, "boardID")
			websocket.ServeWs(d.hub, w, r, boardID)
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.GeneralRateLimit())
		r.Use(middleware.RequireAuth)

		r.Get("/api/auth/me", httputil.MakeHandler(d.authHandler.Me))

		// Cookie-authed: the only place the auth cookie is available.
		r.Get("/api/ws-ticket", httputil.MakeHandler(d.authHandler.WSTicket))

		// Not board-gated: the caller is joining. The token is the authorization.
		r.Post("/api/invites/{token}/accept", httputil.MakeHandler(d.inviteHandler.AcceptInvite))

		r.Route("/api/my-tasks", func(r chi.Router) {
			r.Get("/", httputil.MakeHandler(d.boardHandler.GetMyTasks))
			r.Post("/{cardID}/complete", httputil.MakeHandler(d.boardHandler.CompleteMyTask))
		})

		r.Route("/api/me/settings", func(r chi.Router) {
			r.Get("/", httputil.MakeHandler(d.settingsHandler.GetSettings))
			r.Patch("/", httputil.MakeHandler(d.settingsHandler.UpdateSettings))
		})

		r.Route("/api/boards", func(r chi.Router) {
			r.Get("/", httputil.MakeHandler(d.boardHandler.GetAllBoards))
			r.Post("/", httputil.MakeHandler(d.boardHandler.CreateBoard))

			r.Route("/{boardID}", func(r chi.Router) {
				r.Use(requireBoardMember)

				r.Get("/", httputil.MakeHandler(d.boardHandler.GetBoardData))
				r.Get("/activities", httputil.MakeHandler(d.activityHandler.ListByBoard))
				r.Post("/columns", httputil.MakeHandler(d.boardCmdHandler.CreateColumn))

				r.With(middleware.RequireBoardRole(core.RoleManager)).
					Patch("/", httputil.MakeHandler(d.boardHandler.UpdateBoard))

				r.With(middleware.RequireBoardRole(core.RoleOwner)).
					Delete("/", httputil.MakeHandler(d.boardHandler.StashBoard))

				r.Route("/members", func(r chi.Router) {
					r.Get("/", httputil.MakeHandler(d.boardHandler.GetBoardMembers))
					r.Delete("/me", httputil.MakeHandler(d.boardHandler.LeaveBoard))

					r.Group(func(r chi.Router) {
						r.Use(middleware.RequireBoardRole(core.RoleManager))
						r.Post("/", httputil.MakeHandler(d.boardHandler.AddBoardMember))
						r.Delete("/{userID}", httputil.MakeHandler(d.boardHandler.RemoveBoardMember))
						r.Patch("/{userID}", httputil.MakeHandler(d.boardHandler.UpdateMemberRole))
					})
				})

				r.With(middleware.RequireBoardRole(core.RoleManager)).
					Route("/invites", func(r chi.Router) {
						r.Get("/", httputil.MakeHandler(d.inviteHandler.GetActiveInvite))
						r.Post("/", httputil.MakeHandler(d.inviteHandler.CreateInvite))
						r.Delete("/", httputil.MakeHandler(d.inviteHandler.RevokeInvite))
					})

				r.Route("/tags", func(r chi.Router) {
					r.Get("/", httputil.MakeHandler(d.tagHandler.GetBoardTags))

					r.Group(func(r chi.Router) {
						r.Use(middleware.RequireBoardRole(core.RoleManager))
						r.Post("/", httputil.MakeHandler(d.tagHandler.CreateBoardTag))
						r.Delete("/{tagID}", httputil.MakeHandler(d.tagHandler.DeleteBoardTag))
					})
				})

				// Item endpoints sit at the top level and re-resolve the board for the membership check.
				r.Route("/planning/sessions", func(r chi.Router) {
					r.Get("/", httputil.MakeHandler(d.planningHandler.ListSessions))
					r.Post("/", httputil.MakeHandler(d.planningHandler.CreateSession))
				})
			})
		})

		r.Route("/api/planning/sessions/{sessionID}", func(r chi.Router) {
			r.Get("/", httputil.MakeHandler(d.planningHandler.GetSession))
			r.Patch("/", httputil.MakeHandler(d.planningHandler.UpdateSession))
			r.Delete("/", httputil.MakeHandler(d.planningHandler.DeleteSession))
			r.Post("/items", httputil.MakeHandler(d.planningHandler.CreateItem))
		})

		r.Route("/api/planning/items/{itemID}", func(r chi.Router) {
			r.Patch("/", httputil.MakeHandler(d.planningHandler.UpdateItem))
			r.Delete("/", httputil.MakeHandler(d.planningHandler.DeleteItem))
			r.Post("/promote", httputil.MakeHandler(d.planningHandler.PromoteItem))
			r.Get("/comments", httputil.MakeHandler(d.planningHandler.ListComments))
			r.Post("/comments", httputil.MakeHandler(d.planningHandler.CreateComment))
		})

		r.Route("/api/planning/comments/{commentID}", func(r chi.Router) {
			r.Patch("/", httputil.MakeHandler(d.planningHandler.EditComment))
			r.Delete("/", httputil.MakeHandler(d.planningHandler.DeleteComment))
		})

		// WS write handlers remain for older clients (#197).
		r.Route("/api/columns/{columnID}", func(r chi.Router) {
			r.Patch("/", httputil.MakeHandler(d.boardCmdHandler.UpdateColumn))
			r.Delete("/", httputil.MakeHandler(d.boardCmdHandler.DeleteColumn))
		})

		r.Route("/api/cards", func(r chi.Router) {
			r.Post("/", httputil.MakeHandler(d.boardCmdHandler.CreateCard))
			r.Patch("/{cardID}", httputil.MakeHandler(d.boardHandler.UpdateCard))
			r.Get("/{cardID}", httputil.MakeHandler(d.boardHandler.GetCard))
			r.Delete("/{cardID}", httputil.MakeHandler(d.boardCmdHandler.DeleteCard))
			r.Patch("/{cardID}/move", httputil.MakeHandler(d.boardCmdHandler.MoveCard))
			r.Patch("/{cardID}/done", httputil.MakeHandler(d.boardCmdHandler.ToggleCardDone))
			r.Get("/{cardID}/source", httputil.MakeHandler(d.planningHandler.GetCardSource))
			r.Route("/{cardID}/subtasks", func(r chi.Router) {
				r.Post("/", httputil.MakeHandler(d.subtaskHandler.CreateSubtask))
				r.Get("/", httputil.MakeHandler(d.subtaskHandler.GetSubtasks))
				r.Get("/{subtaskID}", httputil.MakeHandler(d.subtaskHandler.GetSubtask))
				r.Patch("/{subtaskID}", httputil.MakeHandler(d.subtaskHandler.UpdateSubtask))
				r.Delete("/{subtaskID}", httputil.MakeHandler(d.subtaskHandler.DeleteSubtask))
			})
		})

		r.Route("/api/stash", func(r chi.Router) {
			r.Get("/", httputil.MakeHandler(d.boardHandler.GetStashedBoards))

			r.Route("/{boardID}", func(r chi.Router) {
				r.Use(middleware.RequireStashedBoardMember(d.boardService))
				r.Use(middleware.RequireBoardRole(core.RoleOwner))
				r.Delete("/", httputil.MakeHandler(d.boardHandler.HardDelete))
				r.Patch("/restore", httputil.MakeHandler(d.boardHandler.RestoreBoard))
			})
		})
	})

	return r
}

// swagger:model HealthResponse
type HealthResponse struct {
	Status      string `json:"status"          example:"ok"`
	Version     string `json:"version"         example:"v0.3.0"`
	UptimeSecs  int64  `json:"uptime_seconds"  example:"42"`
	DBConnected bool   `json:"db_connected"    example:"true"`
}

// @Summary  Health probe
// @Tags     ops
// @Produce  json
// @Success  200 {object} HealthResponse
// @Failure  503 {object} HealthResponse "DB unreachable"
// @Router   /healthz [get]
func healthHandler(pool *pgxpool.Pool, version string, startedAt time.Time) http.HandlerFunc {
	type response = HealthResponse

	// Memoize the DB ping for 1s so a fast uptime monitor doesn't hammer the pool.
	var (
		mu       sync.Mutex
		cachedOK bool
		cachedAt time.Time
		cacheTTL = time.Second
	)
	probe := func(ctx context.Context) bool {
		mu.Lock()
		defer mu.Unlock()
		if time.Since(cachedAt) < cacheTTL {
			return cachedOK
		}
		cachedOK = pool.Ping(ctx) == nil
		cachedAt = time.Now()
		return cachedOK
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		dbOK := probe(ctx)

		body := response{
			Status:      "ok",
			Version:     version,
			UptimeSecs:  int64(time.Since(startedAt).Seconds()),
			DBConnected: dbOK,
		}
		status := http.StatusOK
		if !dbOK {
			body.Status = "degraded"
			status = http.StatusServiceUnavailable
		}
		httputil.RespondJSON(w, status, body)
	}
}
