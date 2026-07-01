# 0002 — sqlc for data access, no ORM

**Status:** Accepted

## Context

The backend needs a data-access layer over PostgreSQL. The usual options in Go
are an ORM (GORM), a query builder, or hand-written SQL with a codegen step
(sqlc). The project values type safety and predictable SQL over developer
convenience — a Kanban board with realtime broadcasts has queries where an N+1
or a silently-wrong join is a real bug, not a footnote.

## Decision

Use **sqlc**. Queries live in `backend/database/queries.sql`, are executed
against a real Postgres at codegen time, and compile to Go with concrete return
types. The generated code in `internal/db/` is never edited by hand — changing a
query means editing the SQL and re-running `make sqlc`.

The service layer is the only place business logic lives; handlers call services,
services call sqlc-generated code. No repository abstraction on top — sqlc *is*
the repository.

## Consequences

- SQL files are verbose, and every query change is a two-step (edit SQL →
  regenerate). Accepted in exchange for the wins below.
- Renaming a column produces a compile error, not a runtime 500.
- No ORM magic: no lazy-loading N+1s, no hidden query generation. Every query is
  visible SQL you can paste into `psql` to debug.
- When a query gets genuinely gnarly, it's still plain SQL wrapped in a
  service-level method — the escape hatch is "write better SQL," never "reach for
  an ORM."

See also `docs/ARCHITECTURE.md → Why sqlc, not GORM`.
