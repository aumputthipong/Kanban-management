# 0008 — Coupling we are deliberately not cutting

**Status:** Accepted

## Context

A coupling audit of the codebase produced six findings. Two were acted on
(ADR 0007, and giving `ActivityHandler` an interface). This ADR records why the
other four were left alone, so the next contributor — human or agent — does not
re-derive the same list and re-propose the same work.

The audit also found the parts that are already right, and they are worth
stating: no package under `service/`, `db/` or `mapper/` imports `handler/`;
the `Broadcaster` interface keeps handlers ignorant of the hub; `mapper/` had
exactly one leak of a sqlc row to the wire, since fixed.

## Decision

Each item below was judged on three questions: **has it caused a defect yet**,
**would a mistake be caught by anything**, and **is the indirection worth its
reading cost**. Loosening coupling is not free — every interface is another file
to open when following a call, and a one-maintainer project is far more likely to
be hurt by over-abstraction than by tight coupling.

### `BoardServicer` stays as one 25-method interface

It spans five domains (board, my-work, member, card, user), so every handler
depends on far more than it calls, and `MockBoardService` must implement all of
it. That is an Interface Segregation violation, and it is real friction — but
it is friction, not defects. No bug has been traced to it.

Splitting now also risks splitting along the wrong lines: the seams that look
obvious today may not be the ones the project grows into. Revisit when adding a
method to `BoardServicer` has been actively annoying three times, at which point
the right boundaries will be evident rather than guessed.

### No generated type contract for WebSocket payloads

Go emits `map[string]any`; the frontend parses to `any`. Nothing checks that
field names agree — this is what let the `due_date` broadcast defect through.

The complete fix is codegen or a shared IDL, which is more infrastructure than
this project should carry (see ADR 0007's alternatives). ADR 0007 covers the
common half of the risk — the event tag. Field-level shape is covered by handler
tests that assert the emitted payload, which is cheaper and more targeted.

### Client components may keep calling `apiClient` directly

Three client components (`MyWorkTaskModal`, `CalendarTaskModal`,
`CreateProjectModal`) build URLs inline rather than going through the
`lib/*Api.ts` modules, which couples UI files to endpoint paths. Server
components and route pages doing the same are fine — fetching is their job.

No defect, low cost either way. Move a call into its `lib/` module when editing
that component for another reason; do not open a PR to move them all.

### `renameColumnInStore`-style dead code is a known cost of loose coupling

Two sides that agree only by convention cannot tell each other when one stops
listening. The remnants found in this audit are deleted, and ADR 0007's checker
prevents that particular recurrence. The general lesson stands: the looser a
seam, the more it needs a test or a check standing in for the compiler.

## Consequences

- Anyone proposing these four again should bring evidence the situation changed
  — a defect traced to it, or the third repetition of the same friction.
- The bar used here ("has it caused a defect yet") is intentionally higher than
  a textbook one. It is calibrated to a small team, and should be raised if the
  team grows.
