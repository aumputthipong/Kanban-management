# 0007 — WebSocket event tags are constants, checked across both languages

**Status:** Accepted

## Context

The `type` tag on a board broadcast is a contract between two codebases that
share no compiler. The frontend had a proper enum:

```ts
export const WS_EVENT = { CardMoved: "CARD_MOVED", ... } as const;
```

The backend had string literals, written out at each of eleven call sites across
four handler files. AGENTS.md instructed contributors to "update the enum on the
Go and TypeScript sides together" — but there was no Go enum to update.

The contract had already drifted, in both directions:

- `COLUMN_RENAMED` — the frontend had a listener, a store action
  (`renameColumnInStore`), and a unit test for it. Nothing had emitted the event
  since the REST write path started sending `COLUMN_UPDATED` instead. Three
  pieces of live-looking code, all unreachable.
- `CARD_DONE_TOGGLED` — declared in the enum, commented "client → server only",
  and dead since the WebSocket write path was deleted (ADR 0003 follow-up).

Neither drift was noticed by anything. That is the actual problem: a mistyped
tag compiles on both sides, passes `go vet`, passes `tsc`, passes every test,
and surfaces only as a board that quietly stops updating for other people. The
`due_date` broadcast defect found the same week had the same shape — the WS
boundary is the one place in this codebase where nothing checks that two sides
agree.

## Decision

**Broadcast tags are `core.WSEvent` constants. Raw strings are rejected in CI.**

1. `backend/internal/core/wsevent.go` declares the eight events the REST write
   path emits. `core` was chosen over `websocket` deliberately: the handler
   package depends on the `Broadcaster` interface precisely so it never imports
   the hub, and putting the constants in `websocket` would have undone that.

2. `scripts/check-ws-events.mjs` (Node built-ins only, mirroring
   `check-comment-budget.mjs`) fails the build on three conditions:
   - a Go constant with no matching entry in `types/wsEvents.ts`
   - an entry in `types/wsEvents.ts` with no Go constant
   - a string literal passed at an `emit` / `emitTo` call site

   Wired into `make verify` and the CI `comments` job.

3. The two drifted events and their frontend remnants were deleted.

The third check exists because Go's named string types do not give the safety
one might expect: an untyped literal is assignable to `core.WSEvent`, so
`h.emit(id, "CARD_MOVD", ...)` would still compile. The type documents intent;
the script is what enforces it.

## Consequences

- Adding an event now requires touching both files, and CI says so by name if
  only one is touched.
- A typo at a call site fails the build instead of silently disabling a feature.
- The check is textual, not semantic: it verifies that tags match, not that
  payload fields do. Payload shape stays covered by handler tests such as
  `TestUpdateCard_OmittedDueDate_BroadcastsStoredValue`.

## Alternatives considered

**Generate the TypeScript enum from the Go constants.** Correct by construction,
and rejected: a codegen step is another piece of infrastructure to install,
document, and keep working, for a project with one maintainer. The failure it
prevents beyond what the checker catches is narrow.

**A shared schema (protobuf / JSON Schema) for the whole WS payload.** Solves
tag *and* field drift, at the cost of a build pipeline and a new IDL to learn.
Filed under the same reasoning as the ORM and GraphQL entries in AGENTS.md —
revisit if payload drift causes a real defect.
