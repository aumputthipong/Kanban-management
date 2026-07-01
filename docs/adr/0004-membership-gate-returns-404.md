# 0004 — Board membership gate returns 404, not 403

**Status:** Accepted

## Context

Board-scoped routes are guarded by a membership gate
(`middleware.RequireBoardMember`) before any role check. The gate has to answer
"is this authenticated user allowed to touch this board?" — and the HTTP status
it returns on failure leaks information.

The intuitive answer is 403 Forbidden: the user is authenticated but not
permitted. But 403 confirms *the board exists*. An attacker with a valid account
could walk board IDs and watch which flip from 404 (no such board) to 403 (exists,
you're just not on it) — a membership/existence oracle.

## Decision

The membership gate returns **404 Not Found** for every failure that isn't a
clean auth error — non-member, nonexistent board, and malformed board ID all
collapse to the same 404:

- A caller who isn't a member of the board gets 404, indistinguishable from a
  board that doesn't exist.
- A malformed `boardID` (not a UUID) is treated as 404 before it reaches the DB,
  rather than surfacing Postgres's `22P02` as a 500.
- The 403 path is reserved for the **role** gate that runs *after* membership is
  proven: a confirmed member who lacks the required role (e.g. a member trying an
  owner-only action) correctly gets 403.

This is a hard rule, not a preference — see `AGENTS.md`. Regression coverage:
`TestRequireBoardMember_NonMember_Returns404`.

## Consequences

- No enumeration oracle: membership and existence are unobservable to a
  non-member.
- The two-layer split stays clean — **membership → 404, role → 403** — so the
  status code itself tells you which gate failed.
- Slightly less "helpful" errors for a legitimate user who mistypes a board ID
  (they see 404, not 403), which is the correct trade for not leaking existence.

Implemented in `backend/internal/middleware/board_access.go`. See also
`docs/ARCHITECTURE.md → Permission model`.
