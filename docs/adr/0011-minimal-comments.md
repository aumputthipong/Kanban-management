# 0011 — Minimal comments; explanations live in docs

**Status:** Accepted — supersedes [0006](0006-comment-budget.md)

## Context

ADR 0006 capped comment blocks at 3–4 lines. The walls of prose went away, but
the code still read as over-explained: nearly every function, hook and state
variable opened with one to three lines of "why", often retelling history
("the old cycle-on-click was…") or restating the next line. The volume made the
code feel generated rather than written, and the few comments that mattered
were lost among the rest.

Most of that text was not wrong — it was in the wrong place. Rationale is read
once, when someone asks "why is it like this?"; code is read every day.

## Decision

- **Default is no comment.** Names carry the meaning.
- Allowed in code: short section labels, a one-line API doc when the signature
  is not enough, a one-line trap at the line where a mistake would be made
  (pointing to docs for the why), and tool directives.
- Rationale and history move to `docs/CODE-NOTES.md` (per-area notes) or to an
  ADR when it is a real decision.
- `make check-comments` caps every block at **2 content lines**. Swagger
  annotations and tool directives are exempt.

Rejected: keeping 0006's trap exception. A trap still fits one line when the
explanation lives in docs; the long form was the part that read as noise.

## Consequences

- `docs/CODE-NOTES.md` and the ADRs are now the only record of *why*. A change
  that invalidates a note must update it in the same PR.
- Godoc is no longer required on exported Go symbols. `golangci-lint`'s standard
  set does not enforce it, so nothing breaks.
