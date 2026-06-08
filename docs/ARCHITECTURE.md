# Architecture

This doc explains how the system is laid out and *why* — the constraints behind each major decision. Read this if you're contributing or trying to understand the rationale before changing code.

## High-level flow

```
                 ┌──────────────────┐  HTTP (REST, JWT cookie)   ┌─────────────────────┐
   Browser ───▶  │  Next.js 16      │ ─────────────────────────▶ │   Go API            │
                 │  (App Router)    │ ◀──── WebSocket  ────────  │   chi + WS Hub      │
                 │  React 19        │   broadcast (CARD_*,       │                     │
                 │  Zustand store   │    COLUMN_*, ACTIVITY_*)   │                     │
                 └──────────────────┘                            └──────────┬──────────┘
                                                                            │ pgx/v5
                                                                            ▼
                                                                   ┌──────────────────┐
                                                                   │ PostgreSQL       │
                                                                   │ (sqlc-typed)     │
                                                                   └──────────────────┘
```

- **REST** for reads + writes initiated by the user (CRUD).
- **WebSocket** for fan-out: the writer's own action goes through REST + optimistic store mutation; everyone *else* in the same board receives the change via the hub.

---

## Backend

### Layered design

```
handler/  ─── HTTP boundary; decode + validate, call service, encode response
service/  ─── business logic, permission checks, transaction boundaries
db/       ─── sqlc-generated repository (do NOT edit by hand)
```

The split exists so the rules of the domain ("only owners can hard-delete a board") live in `service/` and don't leak into either the HTTP layer or the SQL layer. Handlers stay short — most are 10–30 lines and read like a script. Services accept `context.Context` for cancellation and use `pgxpool.Pool` so they can wrap multi-statement work in transactions.

### Why sqlc, not GORM

sqlc reads `database/queries.sql`, executes them against a real Postgres at codegen time, and emits Go code with concrete return types. The trade-off is verbose SQL files in exchange for:
- compile-time errors when a column is renamed
- zero ORM magic / N+1 surprises
- queries you can copy-paste into `psql` to debug

If a query gets ugly enough that hand-rolled SQL would help, write it as a service-level method that calls sqlc generated code under the hood — but don't reach for an ORM.

### Permission model

Three roles per board: `owner` > `manager` > `member` (see [`internal/core`](../backend/internal/core/)). Enforced in two layers:

1. **Membership gate** — [`middleware.RequireBoardMember`](../backend/internal/middleware/board_access.go) returns 404 (not 403) if the caller is not a member. The 404 prevents enumeration of board IDs.
2. **Role gate** — [`middleware.RequireBoardRole(core.RoleManager)`](../backend/internal/middleware/board_role.go) returns 403 if the caller is a member but lacks the required role.

Action matrix:

| Action                      | owner | manager | member |
|-----------------------------|:-----:|:-------:|:------:|
| Read board / activities     | ✅    | ✅      | ✅     |
| Create / update / move card | ✅    | ✅      | ✅     |
| Update board title / budget | ✅    | ✅      |        |
| Add / remove members        | ✅    | ✅      |        |
| Change member roles         | ✅    | ✅      |        |
| Manage tags                 | ✅    | ✅      |        |
| Move board to trash         | ✅    |         |        |
| Restore / hard-delete       | ✅    |         |        |
| Leave board                 |       | ✅      | ✅     |

The frontend mirrors this matrix for UI gating (hide buttons), but the backend is the source of truth — never trust the client.

### WebSocket hub

[`internal/websocket`](../backend/internal/websocket/) implements a simple hub with one channel per board. Each connection is bound to a `boardID` at handshake time; broadcasts only fan out to clients in that room. Activities (`ACTIVITY_CREATED`) are recorded server-side via [`ActivityService`](../backend/internal/service/activity_service.go) before the broadcast — single source of truth for the audit trail.

Failure modes:
- **Client disconnect** → client's `useWebSocket` reconnects with exponential backoff (1s → 30s, 8 attempts).
- **Server restart** → all clients reconnect on next message; the hub starts fresh — there's no in-memory state to recover.
- **Idle Cloudflare cut** (~100s) → not yet handled; planned via WS heartbeat (P1, see plan).

### Health, shutdown, migrations

- `/health` and `/healthz` return JSON `{status, version, uptime_seconds, db_connected}`. They `Ping()` the pool with a 2-second timeout — if it fails, the response is 503. Use this for k8s liveness probes / Render / Railway / UptimeRobot.
- Graceful shutdown drains in-flight requests for up to 30s on `SIGTERM`/`SIGINT` (see `main.go`).
- Migrations run automatically on startup via [`internal/migrate`](../backend/internal/migrate/) (golang-migrate + pgx/v5 driver). `SKIP_MIGRATIONS=true` opts out — useful when running migrations out-of-band in a separate job.

---

## Frontend

### App Router layout

```
src/app/
├── (auth)/            # /login, /register — own minimal layout
├── (landing)/         # / public landing page
├── (project)/         # /board, /my-tasks, /trash — sidebar + auth-gated layout
├── error.tsx          # route-level boundary (Week 2)
├── global-error.tsx   # root boundary (replaces layout if it crashes)
└── layout.tsx         # global providers (Toast, Auth, theme)
```

### State

- **Zustand** (`src/store/useBoardStore.ts`) — single store per board view; cards/columns/members live here. WebSocket messages mutate it directly; React rerenders subscribed components.
- **No Redux / no Context for board data** — Zustand's selector hooks are enough; using Context for high-frequency board mutations would re-render the entire tree.
- **Toast store** (`useToastStore`) — global UI feedback channel; the API client posts to it on 403 / 5xx so handlers don't repeat error UX.
- **Activity store** — separate from board store because it grows append-only and doesn't need to resync when switching boards.

### Optimistic UI pattern

Drag-and-drop and "complete task" actions follow:

1. Apply the change locally (`store.moveCard(...)`)
2. Fire the mutation to the API
3. On error, **revert** to the previous state (`store.setBoard(prev)`)

The WebSocket event for the same action is *idempotent* on the local store — receiving `CARD_MOVED` for a card already at that position is a no-op. This means the writer doesn't need to filter out their own broadcasts.

### Realtime hook

[`useWebSocket`](../frontend/src/hooks/useWebSocket.ts) owns:
- connection lifecycle (connect, reconnect with exp backoff, cancel on unmount)
- message dispatch into `useBoardStore` / `useActivityStore`
- exposed `status: 'connecting' | 'open' | 'reconnecting' | 'closed'` so a UI banner can react

[`BoardWebSocketContext`](../frontend/src/contexts/BoardWebSocketContext.tsx) wraps `useWebSocket` so any descendant of the board page can call `sendMessage` without re-instantiating the socket.

---

## Planning section

A meeting-notes-style capture surface that lives next to the Kanban board.
The session → item → promote pipeline lets a requirement owner write down
what was discussed and selectively push the actionable rows onto the
board as cards, with the source link preserved both ways.

```
                                            ┌──────────────────────────┐
                                            │ planning_sessions        │
                                            │   per board, has many ↓  │
                                            └─────────────┬────────────┘
                                                          │
                          ┌───────────────────────────────┼───────────────────────────────┐
                          │                               │                               │
                          ▼                               ▼                               ▼
            ┌──────────────────────────┐   ┌──────────────────────────┐   ┌──────────────────────────┐
            │ planning_items           │   │ planning_items           │   │ planning_items           │
            │ type=REQ status=live     │   │ type=DEC status=live     │   │ type=Q   status=live     │
            └─────────────┬────────────┘   └──────────────────────────┘   └──────────────────────────┘
                          │ PromoteItem (tx)
                          ▼
            ┌──────────────────────────┐                                  ┌──────────────────────────┐
            │ cards                    │ ◀── promoted_to_card_id ──────── │ planning_items           │
            │ board.first TODO column  │      (back-link FK)              │ status=promoted          │
            │ AC + Note copied over    │                                  │ claim auto-released      │
            └──────────────────────────┘                                  └──────────────────────────┘
```

### Entities

- **`planning_sessions`** — one per meeting. Scoped per board, soft-cascades on board delete.
- **`planning_items`** — atomic rows. Three types (REQ / DEC / Q) and four statuses (live → selected → promoted, or live → dropped). Carry optional acceptance criteria + implementation note that survive promotion to a card.
- **`planning_item_comments`** — per-item thread with soft delete so a deleted comment leaves a tombstone in place rather than shifting the thread.

### Critical paths

- **PromoteItem** is the only cross-table mutation in the section. Inside one transaction it:
  1. Takes `SELECT ... FOR UPDATE` on the planning item (concurrent promoters serialise — see the integration test `TestPromoteItem_ConcurrentPromote_ExactlyOneSucceeds`).
  2. Inserts a `cards` row in the board's first `TODO` column, copying `acceptance_criteria` + `implementation_note` forward.
  3. Sets `status='promoted'` + `promoted_to_card_id=<new card>`.
  4. Auto-releases any claim (clears `claimed_by_user_id` + `claimed_at`).
- **ClaimItem** is atomic without a transaction: the `WHERE claimed_by_user_id IS NULL` predicate in the UPDATE is the serialisation point. Concurrent claimers see one success + N-1 `ErrPlanningItemAlreadyClaimed`.
- **Card backlink** uses partial index `idx_planning_items_promoted_card WHERE promoted_to_card_id IS NOT NULL` so the reverse lookup `GET /api/cards/:cardID/source` stays cheap even as the cards table grows.

### REST surface (all under `/api/planning`)

```
GET    /boards/:boardID/planning/sessions       list sessions on a board
POST   /boards/:boardID/planning/sessions       create session
GET    /planning/sessions/:sessionID            session detail (items inline)
PATCH  /planning/sessions/:sessionID            edit session (PATCH semantics)
DELETE /planning/sessions/:sessionID            delete session

POST   /planning/sessions/:sessionID/items      capture item
PATCH  /planning/items/:itemID                  edit item (type/title/status/AC/Note)
DELETE /planning/items/:itemID                  delete item
POST   /planning/items/:itemID/promote          → card

GET    /planning/items/:itemID/comments         list thread (includes tombstones)
POST   /planning/items/:itemID/comments         add comment
PATCH  /planning/comments/:commentID            edit (author only)
DELETE /planning/comments/:commentID            soft delete (author OR board owner/manager)

POST   /planning/items/:itemID/claim            claim ("I'm looking at this")
DELETE /planning/items/:itemID/claim            release (own OR force-release by owner/manager)

GET    /cards/:cardID/source                    reverse: which planning row produced this card?
```

Permission gate is the standard 404-not-403 pattern. The handler resolves
`boardID` from the request's nested ID (item → session → board for most
routes; comment → item → session → board for the comment edit/delete
paths) and calls `requireMembership` before any mutation. Non-members
get 404 with no signal about whether the resource exists.

### Activity event types

Every mutation logs into `activities` after the underlying write commits.
The full set the feed renderer knows about:

```
planning.session_{created,updated,deleted}
planning.item_{created,updated,deleted,promoted}
planning.comment_{created,edited,deleted}
planning.item_{claimed,released}
planning.claim_auto_released_on_promote
```

The comment + claim payloads carry a body preview / item title so the
team feed (`TeamTabContent.describeActivity`) renders human-readable
lines without joining back to the source tables.

### Frontend layout

- Page: `frontend/src/app/(project)/board/[boardId]/planning/[sessionId]/page.tsx`
- Data hook: `useSessionItems(boardId, sessionId)` — initial fetch + optimistic CRUD over the items array, plus claim/release helpers.
- Per-item thread hook: `usePlanningComments(itemId, currentUserId)` — lazy, never fetches until the consumer calls `load()`. Multi-instance design documented in the hook header.
- Composition: `SessionCaptureView` orchestrates filter chips, items list, capture input, sidebar, deep-link scroll. `ItemRow` extracts a popover, claim affordance, action button cluster, and details panel (see `frontend/src/components/board/planning/`).

---

## Cross-cutting concerns

| Concern                  | Where it lives                                                            |
|--------------------------|---------------------------------------------------------------------------|
| Auth (JWT + cookie)      | `internal/token` issues, `middleware.RequireAuth` validates               |
| Input validation         | `httputil.DecodeAndValidate` + `validate:"..."` tags on DTOs              |
| Rate limiting            | `middleware.AuthRateLimit` (20/min/IP) + `GeneralRateLimit` (300/min/IP)  |
| Security headers         | `middleware.SecurityHeaders(production)` — HSTS only in prod              |
| CORS                     | `middleware.CORS(frontendURL, ...)` — single allowed origin from env      |
| Permission gating        | `middleware.RequireBoardMember` + `RequireBoardRole`                      |
| HTTP error envelope      | `httputil.NewAPIError` + `MakeHandler` → `{error, code}` JSON             |

---

## What's intentionally **not** here

- **No ORM** — sqlc + service layer is enough.
- **No GraphQL** — REST + WebSocket cover all current use cases without a schema layer to maintain.
- **No microservices** — a single Go binary; if scale becomes an issue, the WS hub is the obvious split point (Redis pub/sub broker), not the auth or board domain.
- **No Redis cache** — board reads are fast against the typed sqlc queries; introduce caching only when an actual hot path appears.
- **No event-sourcing** — `activities` table is append-only audit, not a rebuild-state log.

If any of these get added, document the trigger in `docs/decisions/` so future contributors know what changed.
