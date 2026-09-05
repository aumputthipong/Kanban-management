// cmd/api/routes.go
package main

import (
	"context"
	"net/http"
	"sync"
	"time"

	_ "github.com/aumputthipong/mini-erp-kanban/backend/docs" // swagger generated
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
	startedAt       time.Time
}

func setupRoutes(d routerDeps) http.Handler {
	r := chi.NewRouter()

	// Global middleware. SentryRecoverer captures the panic before chi's
	// stdlib Recoverer turns it into a 500 — both run, in this order.
	// RequestLogger replaces chi's default Logger so sensitive query strings
	// (OAuth `code` / `state`, the WS auth ticket) are redacted before they
	// reach any log sink.
	r.Use(chiMiddleware.RequestID)
	r.Use(middleware.RequestLogger)
	r.Use(observability.SentryRecoverer())
	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.SecurityHeaders(d.production))
	r.Use(observability.HTTPMetrics)
	// gzip JSON/text responses. Board payloads (columns + cards + tags) compress
	// 4–8×; skipped for already-compressed types (images, fonts).
	r.Use(chiMiddleware.Compress(5, "application/json", "text/html", "text/css", "text/plain"))

	// Health endpoints — used by load balancers / uptime monitors / k8s probes
	r.Get("/health", healthHandler(d.pool, d.version, d.startedAt))
	r.Get("/healthz", healthHandler(d.pool, d.version, d.startedAt))

	// Prometheus scrape target. Not gated by auth — assume the network layer
	// (firewall / k8s NetworkPolicy) is what restricts access.
	r.Handle("/metrics", observability.MetricsHandler())

	// API docs (Swagger UI). Spec is regenerated via `swag init` — see Makefile.
	// Disabled in production: a public API map gives attackers a free roadmap of
	// endpoints, parameter names, and auth flows. Set ENV != "production" locally
	// or behind a private network to enable.
	if !d.production {
		r.Get("/docs/*", httpSwagger.Handler(httpSwagger.URL("/docs/doc.json")))
	}

	r.Route("/api/auth", func(r chi.Router) {
		r.Use(middleware.AuthRateLimit())

		r.Post("/register", httputil.MakeHandler(d.authHandler.Register))
		r.Post("/login", httputil.MakeHandler(d.authHandler.Login))
		r.Post("/oauth", httputil.MakeHandler(d.authHandler.OAuthCallback))
		r.Post("/logout", httputil.MakeHandler(d.authHandler.Logout))
		// Refresh is unauthenticated: the refresh cookie IS the credential.
		// Rate-limited via the surrounding /api/auth group's AuthRateLimit.
		r.Post("/refresh", httputil.MakeHandler(d.authHandler.Refresh))

		r.Get("/google", httputil.MakeHandler(d.oauthHandler.RedirectToGoogle))
		r.Get("/google/callback", httputil.MakeHandler(d.oauthHandler.HandleGoogleCallback))
	})

	requireBoardMember := middleware.RequireBoardMember(d.boardService)

	// /ws/{boardID} authenticates from a `ticket` query param instead of the
	// auth cookie: a browser cannot put a header on a WebSocket, and on a
	// split-domain deploy the cookie is first-party to the frontend origin and
	// never reaches this host (docs/adr/0005-websocket-ticket-auth.md).
	// Membership is still gated — otherwise any authenticated user could join
	// an arbitrary board's room, receive every broadcast, and send mutating WS
	// messages (handlers don't re-check authz beyond board scoping).
	r.Group(func(r chi.Router) {
		r.Use(middleware.GeneralRateLimit())
		r.Use(middleware.RequireWSTicket)

		r.With(requireBoardMember).Get("/ws/{boardID}", func(w http.ResponseWriter, r *http.Request) {
			boardID := chi.URLParam(r, "boardID")
			websocket.ServeWs(d.hub, w, r, boardID)
		})
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.GeneralRateLimit())
		r.Use(middleware.RequireAuth)

		r.Get("/api/auth/me", httputil.MakeHandler(d.authHandler.Me))

		// Mints the ticket the browser then puts in the WS URL. It lives on the
		// cookie-authed path because that is the only place the auth cookie is
		// actually available to us.
		r.Get("/api/ws-ticket", httputil.MakeHandler(d.authHandler.WSTicket))

		// Accept an invite link — authenticated but NOT board-gated (the caller
		// is joining, not yet a member). A valid token is the authorization.
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

				// Shareable invite link — manager+ only (managing who can join is
				// a privileged action, like adding members directly).
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

				// Planning section — sessions live under their board. Item-
				// level endpoints sit at the top level (/api/planning/...)
				// because the URL only carries the item ID; the handler
				// re-resolves the board for membership check.
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

		r.Route("/api/cards", func(r chi.Router) {
			r.Post("/", httputil.MakeHandler(d.boardHandler.CreateCard))
			r.Patch("/{cardID}", httputil.MakeHandler(d.boardHandler.UpdateCard))
			r.Get("/{cardID}", httputil.MakeHandler(d.boardHandler.GetCard))
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
				// Stashed-only gate: these operate on boards that ARE stashed.
				r.Use(middleware.RequireStashedBoardMember(d.boardService))
				r.Use(middleware.RequireBoardRole(core.RoleOwner))
				r.Delete("/", httputil.MakeHandler(d.boardHandler.HardDelete))
				r.Patch("/restore", httputil.MakeHandler(d.boardHandler.RestoreBoard))
			})
		})
	})

	return r
}

// HealthResponse is the body returned by /health and /healthz.
//
// swagger:model HealthResponse
type HealthResponse struct {
	Status      string `json:"status"          example:"ok"`
	Version     string `json:"version"         example:"v0.3.0"`
	UptimeSecs  int64  `json:"uptime_seconds"  example:"42"`
	DBConnected bool   `json:"db_connected"    example:"true"`
}

// healthHandler returns a JSON probe with build version, uptime, and DB connectivity.
// 503 if the DB ping fails so load balancers can drop the instance.
//
// @Summary  Health probe
// @Tags     ops
// @Produce  json
// @Success  200 {object} HealthResponse
// @Failure  503 {object} HealthResponse "DB unreachable"
// @Router   /healthz [get]
func healthHandler(pool *pgxpool.Pool, version string, startedAt time.Time) http.HandlerFunc {
	type response = HealthResponse

	// Memoize the DB ping for 1 second so an uptime monitor hitting /healthz
	// every 100ms doesn't hammer the pool. Single goroutine writes; mutex
	// keeps readers consistent.
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
