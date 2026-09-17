# 0009 — Card PATCH sends only the changed field, and clears with sentinels

**Status:** Accepted

## Context

The card modal auto-saves one field at a time, but every save sent the whole
form: title, description, due date, assignee, priority, estimate and tags. The
modal does not follow broadcasts, so a modal left open holds stale values. When
two people had the same card open, the second save wrote the first person's
edit back to its old value, silently.

Fixing that exposed a second bug. The form sent `null` to clear a field, but Go
decodes JSON `null` and an omitted key to the same nil pointer, and the handler
treats nil as "leave unchanged". Unassigning a card, removing a due date or an
estimate only changed the screen; a reload brought the old value back.

## Decision

- **The client sends only the field the user just committed.** `useCardForm`
  passes the committed field name up; `handleUpdateCard` builds the body and the
  optimistic store patch from that field alone. Concurrent edits to *different*
  fields of one card therefore both survive. Two edits to the *same* field are
  last-write-wins, which we accept for a small-team tool.
- **Clearing uses a sentinel, not `null`.** `""` clears `assignee_id`,
  `priority` and `due_date` (validators allow `eq=` alongside the real rule, and
  the handler stores NULL, since these are uuid/enum/date columns). `0` clears
  `estimated_hours`, stored as NULL; an estimate of zero hours carries no
  meaning. `description` keeps its existing rule: `""` is stored as `""`.
- **The open modal still does not follow broadcasts.** Showing other people's
  edits inside an open modal is deferred: it is a convenience, whereas losing
  data was not acceptable.

We did not introduce a tri-state optional JSON type to tell `null` from omitted.
It would change the decoding convention for every PATCH DTO; the sentinel keeps
the existing convention and fixes the one DTO that needed clearing.

## Consequences

- Any new caller of `PATCH /api/cards/{id}` must send only what it changes, and
  must use the sentinels to clear. Sending `null` is a no-op.
- `estimated_hours: 0` can never be stored; it always means "no estimate".
