# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Repository polish: `CHANGELOG.md`, `CONTRIBUTING.md`, top-level `Makefile`.
- `docs/CODE-NOTES.md` — per-area rationale moved out of code comments.

### Changed

- Comment convention: default to no comment ([ADR 0011](docs/adr/0011-minimal-comments.md), supersedes 0006). `make check-comments` caps blocks at 2 lines; ~2,000 narration lines removed across frontend and backend.

### Removed

- `LICENSE` (was MIT). Project no longer ships with an explicit license — see GitHub's
  default copyright defaults (all rights reserved) until a new license is chosen.

## [0.4.0] — 2026-05-06

Deployment readiness Week 1–4. Containerization, validation, observability scaffolding, and a CI pipeline.

### Added

- **Containerization** — multi-stage Dockerfiles for backend (`distroless/static:nonroot`) and frontend (Next.js standalone), `.dockerignore` for both, `docker-compose.prod.yml` with db healthcheck, db→backend dependency wiring.
- **Database migrations on startup** — `internal/migrate` runs `golang-migrate` (pgx/v5 driver) on every boot; `SKIP_MIGRATIONS=true` opts out.
- **Health endpoint** — `/health` and `/healthz` return JSON `{status, version, uptime_seconds, db_connected}`; 503 on DB failure.
- **Bootstrap hardening** — explicit `pgxpool` config (`MaxConns=25`, `MinConns=5`, `MaxConnIdleTime=5m`), 30 s graceful shutdown timeout, `ReadHeaderTimeout` to mitigate slowloris, ldflags-injected `version` surfaced via healthz.
- **Input validation** — `httputil.DecodeAndValidate` (`go-playground/validator/v10`) with field-level error messages on every write endpoint; `validate:"..."` tags on all request DTOs (uuid / oneof / email / min / max / hexcolor / datetime).
- **Rate limiting** — `httprate`-based middleware: 20 req/min/IP on `/api/auth/*`, 300 req/min/IP on protected routes.
- **Security headers middleware** — `X-Content-Type-Options`, `X-Frame-Options: DENY`, `Referrer-Policy`, `Permissions-Policy`, `Strict-Transport-Security` (production only), CSP baseline.
- **Frontend resilience** — `useWebSocket` reconnect with exponential backoff (1s → 30s, max 8 attempts) plus exposed `status` for UI banners; route-level `app/error.tsx` and root `app/global-error.tsx`.
- **Documentation** — `docs/ARCHITECTURE.md` (layered design, permission matrix, optimistic UI pattern), `docs/DATABASE.md` (Mermaid ERD covering 10 tables, migration rules), `docs/DEPLOY.md` (pre-flight checklist, three deploy paths, rollback playbook).
- **OpenAPI / Swagger UI** — `swaggo/swag` annotations on ~22 endpoints; spec generated at `backend/docs/swagger.{json,yaml}`; interactive UI served at `/docs/index.html`; `go generate ./cmd/api` regenerates after annotation changes.
- **godoc / JSDoc sweep** — package and exported-symbol comments on Go `core`, `token`, `middleware`, `service/interfaces`; JSDoc on key frontend hooks (`useBoardStore`, `useWebSocket`, `useDashboardStats`, `useBoardActions`, `useBoardData`, `useBoardRole`, `useCanEdit`, `useDragActions`, `useActivityFeed`, `useToastStore`, `useActivityStore`), `BoardWebSocketContext`, `apiClient`, and `utils/{avatar,date_helper}`.
- **GitHub Actions CI** — `.github/workflows/ci.yml`: backend (`go vet`, race tests + coverage upload, prod binary build), frontend (`tsc`, `lint`, `vitest`, `next build`), docker (buildx + GHA cache for both images); concurrency group cancels superseded PR runs. CI status badge in README.
- **Dependabot** — `.github/dependabot.yml`: weekly Mon scans of go modules, npm, github-actions; minor + patch grouped to reduce PR noise.
- **Structured logging** — `internal/logging` configures `slog` (JSON in production, text in dev); `LOG_LEVEL` env override; `main.go` callsites migrated from `log` to `slog` with structured key=value pairs.
- **Frontend logger wrapper** — `src/lib/logger.ts` silences `info`/`debug` in production unless `NEXT_PUBLIC_LOG_LEVEL=debug`; `useWebSocket` migrated to use it.

### Changed

- README rewritten — hero, tech stack table, ASCII architecture diagram, Quick Start (Docker + local), env-vars overview, link-out to `docs/`, common-tasks table, deployment notes.
- Backend `.env.example` complete (replaces stale `.example.env`); frontend `.env.example` added; both stacks document every env var the code reads.
- `httputil.ErrorResponse` is now a typed struct (was an inline `map[string]string`) so the swagger spec can render the failure schema.

## [0.3.0] — 2026-04 (estimated)

Cross-board "My Tasks" experience and unified task views / board overview.

### Added

- `/my-tasks` page — cross-board task list grouped by date or by project; column-aware rows showing real workflow position; 3 date tabs (Due soon / Overdue / No date) with sub-grouping by day; status summary cards.
- Project Board "Tasks" tab redesign — single Up Next panel with urgency buckets (Overdue / Today / Tomorrow / This week), assignee + column + priority badges per row, "Just mine" toggle.
- Unified visual language across `/my-tasks`, by-project view, and board Tasks tab — same gradient-bar header + bordered card pattern.

## [0.2.0] — 2026-03 (estimated)

Permission system and team collaboration UX.

### Added

- Role-based permission system — `owner` / `manager` / `member` matrix enforced at middleware (`RequireBoardMember` returns 404 for non-members to prevent enumeration; `RequireBoardRole` returns 403); UI gates mirror the matrix.
- Restore-from-trash + leave-board flows.
- Team workload + activity feed widgets on the project overview.
- Navbar redesign with `UserMenu` dropdown.

## [0.1.0] — 2026-01 (estimated)

Initial public-facing version.

### Added

- Multi-board Kanban with drag-and-drop, optimistic UI, `@dnd-kit/core`.
- Real-time sync via WebSocket hub (per-board rooms, broadcasts card / column / activity events).
- Type-safe SQL with `sqlc` against PostgreSQL; layered `handler → service → repository` architecture.
- JWT cookie auth + Google OAuth login.
- Board overview dashboard — totals, progress, workload, insights.

[Unreleased]: https://github.com/aumputthipong/Kanban-management/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/aumputthipong/Kanban-management/releases/tag/v0.4.0
[0.3.0]: https://github.com/aumputthipong/Kanban-management/releases/tag/v0.3.0
[0.2.0]: https://github.com/aumputthipong/Kanban-management/releases/tag/v0.2.0
[0.1.0]: https://github.com/aumputthipong/Kanban-management/releases/tag/v0.1.0
