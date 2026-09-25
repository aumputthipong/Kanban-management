# Code notes

The *why* behind code that would otherwise need a comment. Code stays lean
(see [ADR 0011](adr/0011-minimal-comments.md)); anything longer than a one-line
trap lives here, grouped by area. Update the note in the same PR that changes
the behaviour.

## Frontend

### Board

- **Card detail modal** auto-saves per field — there is no Save button. A commit
  sends only the changed field ([ADR 0009](adr/0009-card-patch-sends-changed-fields.md)).
  The scrim has no `backdrop-blur`: blurring the whole board is a per-frame GPU
  composite that makes the modal feel janky.
- **Create task modal** is deliberately different from the edit modal: the fast
  path is title + Enter, subtasks stay collapsed, and it sends one
  `CARD_CREATED`. The assignee defaults to the creator; "unassigned" must be a
  deliberate click. It is over the 200-line guideline on purpose — it is one form.
- **DONE columns** default to collapsed, remembered per board + column in
  localStorage. The collapsed strip is still a drop target (drag a card onto it
  to close it).
- **Column colours** ("Solid Cap"): the cap/body formulas live in
  `ColumnOptionsModal` and are shared with `Column` so the preview is exact.
- **Danger zone** exposes only "stash" (recoverable). Transfer-ownership and
  permanent delete from settings are out of scope for now.
- **Dashboard stats** treat the last column as Done. The "bottleneck" insight is
  separate from the Team tab's ownership counts (`useBoardOwnership`).
- **Activity feed** merges `card.updated` rows from the same actor on the same
  card within 10 minutes, so rapid field edits don't flood the feed.

### Calendar

- Month grid only — day/week/agenda views were never built, so there is no view
  switcher.
- Status and "My tasks" filters are local to the calendar. Priority / tag /
  assignee reuse the board store filters so they stay in sync with the Board tab.
- Clicking a pill opens a read-only quick view; "open task" goes to the full
  editable modal.

### Members & invites

- Members are added by exact email only — there is no user directory to browse.
- Opening the invite modal guarantees a usable link: it reuses the active one or
  mints a replacement. The link stays hidden until the modal is opened (less
  leak on screen-share). "New link" revokes the old one.
- `/invite/<token>` for a logged-out visitor: `apiClient`'s 401 bounce goes to
  `/login?redirect=/invite/<token>`, so the join completes after login.

### Planning (Notes)

- Capture mutations are optimistic fire-and-forget with a toast on failure —
  during a meeting, capture speed beats confirming trivial writes. The one
  exception is retyping: promoted items are frozen server-side, so a failed
  type change is reverted by hand.
- The type chip is display-only; changing type lives in the row's overflow menu.
  The old cycle-on-click was undiscoverable and made REQ → DEC two clicks — don't
  bring it back.
- No cmd-1 / D / S shortcuts: each collided with a browser default. The keyboard
  surface is Enter / Escape / arrows only.
- The expanded row holds one free-text note. Acceptance criteria and dev notes
  live on the card, because they were rarely filled during capture.
- The session filter lives in the URL via `history.replaceState` so it survives
  reload and copy-paste without a Next router re-render (and without the Suspense
  boundary `useSearchParams` would force).
- The open-questions callout is read-only: the model has no "answered" status yet.
- Comment threads load lazily per row and refetch on tab focus instead of polling.
- The export dialog's editable "Task" footer is kept in localStorage so the same
  framing carries over between meetings.

### My Work

- Completing a task is a delayed commit (5 s undo): a move into DONE has no clean
  server-side reversal, so it is safer to never send it than to undo it. Pending
  cards are filtered out of refetches so they can't reappear.
- Snooze is optimistic and reuses `PATCH /cards/:id { due_date }`; undo sends
  `""` to clear a date that was originally empty.
- "Done today" is session-local — the API has no such counter.

### Project list & landing

- "Recent" sort keeps the backend order (per-user `last_accessed_at`). Sorting
  by `updated_at` on the client would let other members' edits reshuffle it.
- The landing page probes `/me/settings` server-side and redirects signed-in
  visitors to their default landing; any failure renders the marketing page.

## Backend

### Routing & middleware

- Order in `routes.go` matters: `RequestID` → `ClientIPResolver` (before the
  logger and every rate limiter, which key off it) → `SentryRecoverer` (before
  chi's `Recoverer`, so the panic is captured before it becomes a 500) →
  `RequestLogger` (redacts OAuth `code`/`state` and the WS ticket).
- `/metrics` is unauthenticated on purpose — restrict it at the network layer.
  Swagger UI is disabled in production: a public API map is a roadmap for attackers.
- `/health` memoizes the DB ping for 1 s so a fast uptime monitor can't hammer the pool.
- Rate limits: auth 20/min per IP; demo 30/hour (each call seeds a whole board,
  leaves room for an office NAT). An empty limiter key would put every caller in
  one bucket, so the RemoteAddr fallback is required.

### Auth

- A refresh-token issuance failure at login is logged, not fatal: the user keeps
  the access token until it expires.
- Refresh returns an undifferentiated 401 for invalid *and* expired tokens, so
  valid-but-expired tokens can't be probed. Login does the same for
  "wrong password" vs "OAuth-only account" (anti-enumeration).
- Logout always returns 204, even if revoking fails — a transient DB error must
  not trap the user in a logged-in state.
- The WS ticket carries `aud=ws`: `Parse` rejects it (a ticket leaked from a URL
  must not open the REST API) and `ParseWSTicket` rejects session tokens.

### Activity & broadcasts

- `Record` (sync) is used when a broadcast needs the new row's id/`created_at`;
  `RecordAsync` is fire-and-forget on a background context and drops the job
  when the queue is full — the audit log is best-effort after the mutation commits.
- Broadcast payloads are built from the stored row, never the request: a field
  the client omitted would be `null`, and receiving stores spread the payload
  over their copy.

### Members, invites, subtasks

- A board has at most one live invite link (revoke + create in one transaction).
  Accepting is idempotent via `ON CONFLICT`, so a double-clicked link can't 500.
- Subtask edits by a member without edit rights get 403, not 404 — they can
  already see the card, so there is nothing left to hide.
- Removing a member evicts their sockets: membership is only checked at the
  WS handshake.

### Planning

- `PromoteItem` is the one cross-table transactional write. It locks the item
  (`FOR UPDATE`) so two concurrent promoters can't both see `live`. Dropped
  items must be un-dropped first. The card lands at the top of the first TODO
  column. Only `planning.item_promoted` is logged — no `card.created`, which
  would double-count.
- Promoted items can't be retyped: the card already carries the original meaning.

### Demo sandbox

- Sandboxes live 24 h and are purged hourly by an in-process loop (a sweep is a
  handful of deletes, not worth a cron job). Purge order: boards first (the
  cascade takes columns, cards, members, tags, planning), then tables whose user
  FK has no `ON DELETE`. Sandbox creation is not one transaction; the purge
  collects half-seeded sandboxes.

### Known issues

- **T6** — `CreateColumn` reads positions and writes a midpoint in Go without a
  lock, so concurrent creates can share a position. The fix belongs in SQL;
  `TestCreateColumn_ConcurrentCreates_CanCollideOnPosition_T6` documents it.
