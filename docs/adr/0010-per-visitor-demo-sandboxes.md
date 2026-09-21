# 0010 — The demo button mints a per-visitor sandbox, not a shared login

**Status:** Accepted

## Context

The repository already seeded a shared demo account (`demo@turtask.app`,
published in the README) so a visitor could look at real content without
registering. That covers "can a stranger see the product", but not the audience
that actually matters for a portfolio project: a recruiter who opens the link,
spends about a minute, and will not type credentials from a README.

Two things were wrong with the shared account as the answer.

**It degrades.** One account means one board. Whatever the last visitor did —
emptied a column, deleted the board, renamed everything to "asdf" — is what the
next visitor sees. The demo is at its best on the day it is seeded and gets
worse from there, silently, exactly when nobody is watching.

**The landing page did not lead anywhere.** Its only call to action linked to
`/dashboard`, and the dashboard page fetched boards without catching the 401, so
a logged-out visitor got the error boundary rather than the login page.

The alternative we rejected was a read-only demo mode: one shared account with
every mutation blocked. For a Kanban board that inverts the demo — the product
*is* dragging cards, and a drag that ends in an error toast reads worse than no
demo at all.

## Decision

- **`POST /api/auth/demo` provisions a throwaway user and its own board.** Each
  call creates a `users` row with `is_demo = true` and a
  `demo_expires_at` 24 hours out, seeds a private copy of the sample board, and
  issues the normal session cookies through the existing `issueSession`. The
  response carries `board_id` so the client lands the visitor on the board
  itself rather than an empty dashboard.
- **One fixture, two callers.** `service.SeedSampleBoard` builds the sample
  board; `cmd/seed` uses it for the shared account and `DemoService` for each
  sandbox, so the two demos cannot drift apart.
- **The endpoint is unauthenticated and rate-limited instead.** Minting the
  sandbox *is* the entry point, so there is no credential to check. Because a
  call writes a user plus a whole seeded board, `DemoRateLimit` (5/hour/IP)
  overrides the `/api/auth/*` group's 20/min.
- **Expired sandboxes are swept in-process.** `StartPurgeLoop` runs hourly off
  the server's context. The delete order is forced by the schema:
  boards first (one cascade takes columns, cards, members, tags, planning and
  that board's activities), then `activities`, `planning_item_comments` and
  `time_logs`, whose user references have no `ON DELETE CASCADE` and would
  otherwise abort the user delete.
- **The session is labelled in the UI.** `/api/auth/me` returns `is_demo`, and
  the project layout renders an orientation bar naming what is worth trying.

## Consequences

- Demo data accumulates between sweeps: at the 5/hour/IP cap, a sandbox is about
  one user, one board, four columns, eight cards and a planning session. If that
  ever becomes a real cost, the lever is `DemoValidity`, not the feature.
- `users` now has two columns that mean nothing for real accounts. The
  alternative, a separate `demo_users` table, would have forced a join or a
  union into every identity lookup for a boolean.
- The purge's delete order encodes today's FK graph. A new table with a bare
  `REFERENCES users(id)` will start failing the sweep; the integration tests in
  `demo_service_integration_test.go` are what catch that.
- A sandbox is single-player. Realtime still demos — two browser tabs are two
  connections to the same board room — but "watch a teammate move a card" does
  not, beyond the seeded companion member's avatars.
