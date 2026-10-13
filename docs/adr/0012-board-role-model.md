# 0012 — Board role model: guard what destroys, leave collaboration open

**Status:** Accepted

## Context

Every board has three roles: `owner` > `manager` > `member`. ADR 0004 split
enforcement into two gates: membership (404) and role (403). That split only
covers routes that carry `boardID` in the URL. Routes that touch a row by its
own ID (`/api/cards/:id`, `/api/columns/:id`, `/api/planning/...`) have to work
out the board themselves, and each handler did its own check.

An audit in October 2026 found that several of those handlers checked
membership but not role. The UI hid the buttons, so the gaps were invisible
in normal use, but the API allowed them:

- a manager could add someone as a second `owner` (fixed in #255)
- any member could delete any card, and rename or delete columns
- managers could demote or remove other managers
- any member could delete a planning session, which cascades every note in it

The deeper question was which actions should be restricted at all. Turtask
targets small teams. In a small team, locking everyday actions to a role adds
friction without making anything safer, while a single destructive click can
erase other people's work.

## Decision

**1. One place for the rules.** Permission rules are pure functions in
[`backend/internal/core/permissions.go`](../../backend/internal/core/permissions.go)
(`CanDeleteCard`, `CanManageMember`, …). Middleware and handlers call them
after resolving the board and the caller's role. A route addressed by row ID
resolves the board first, so a non-member still gets 404 (ADR 0004).
[`frontend/src/lib/permissions.ts`](../../frontend/src/lib/permissions.ts)
mirrors the same functions to hide controls. It is UI only; the backend
decides.

**2. A strict hierarchy for people.** There is exactly one owner, and
ownership moves only through an explicit transfer (the old owner becomes
manager). Nobody, including managers, can grant `owner`. A manager may add,
remove or re-role **members only**, and may grant only `member`. Promoting or
demoting a manager is the owner's call. This is the same "act only below your
own rank" rule that Discord uses for its role hierarchy.

**3. Guard what destroys, leave collaboration open.**

- *Guarded:* deleting a card (its creator or manager+; an assignee can edit but
  not delete), creating, renaming or deleting columns (manager+, since they
  shape everyone's board), and deleting a planning session (manager+, or its
  creator while the session is still empty).
- *Open to every member:* moving cards and marking them done, which are the
  core Kanban interactions (Trello leaves them open too), and everything inside
  Planning: notes, edits, comments and promotion to a card.

Planning is open on purpose. A session is meant to be opened and typed into
live during a meeting, often by one person writing up everyone's points. The
person who typed a note is therefore not its owner, so "creator may edit" rules
would block the note-taker without protecting anyone. Notes can be parked
("พักไว้ก่อน") and restored instead of deleted. Every change is in the activity
log, which is how the team keeps people accountable. The one guarded action is
deleting a whole session, the only way to erase many people's notes in one
click.

## Consequences

- A new rule is one function plus a table test in `core`, and a by-ID handler
  can't skip it by accident: the role comes out of the same helper that checks
  membership (`cardContext`, `columnContext`, `authorizeMemberTarget`).
- The frontend mirror can drift. Unlike WebSocket events (ADR 0007), no check
  compares `permissions.go` and `permissions.ts`. A drift only shows or hides a
  button the backend will then reject; it can't widen access.
- Open actions rely on the activity log instead of prevention. There is no undo
  for an edit, only a record of who made it.
- Revisit this ADR if Planning gains a "finalized" session state or private
  drafts. A finalized session is where tighter edit rules would start to mean
  something.

See `docs/ARCHITECTURE.md → Permission model` for the full action matrix.
